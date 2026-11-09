package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var errNeedAdmin = errors.New("este ajuste requiere ejecutar como administrador")

type BackupItem struct {
	Change RegChange `json:"change"`
	Orig   RegState  `json:"orig"`
}

type Backup struct {
	Items     []BackupItem `json:"items,omitempty"`
	PrevPower string       `json:"prevPower,omitempty"`
	At        string       `json:"at"`
}

type store struct {
	Items map[string]*Backup `json:"items"`
}

// Engine aplica y revierte tweaks: guardar original -> aplicar -> verificar -> (rollback si falla).
type Engine struct {
	mu         sync.Mutex
	dir        string
	st         store
	sim        bool
	simApplied map[string]bool
}

func NewEngine() (*Engine, error) {
	base, err := os.UserCacheDir() // %LOCALAPPDATA%
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, "FPSBoosterPro")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	e := &Engine{dir: dir, st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}}
	if b, err := os.ReadFile(filepath.Join(dir, "backup.json")); err == nil {
		if err := json.Unmarshal(b, &e.st); err != nil {
			return nil, fmt.Errorf("backup.json dañado (no se toca nada): %w", err)
		}
		if e.st.Items == nil {
			e.st.Items = map[string]*Backup{}
		}
	}
	return e, nil
}

func (e *Engine) save() error {
	b, err := json.MarshalIndent(e.st, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(e.dir, "backup.json")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (e *Engine) log(action, id string, err error) {
	rec := map[string]any{"t": time.Now().Format(time.RFC3339), "action": action, "id": id, "sim": e.sim}
	if err != nil {
		rec["error"] = err.Error()
	}
	b, _ := json.Marshal(rec)
	f, ferr := os.OpenFile(filepath.Join(e.dir, "changes.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if ferr != nil {
		return
	}
	defer f.Close()
	f.Write(append(b, '\n'))
}

func (e *Engine) SetSimulation(v bool) {
	e.mu.Lock()
	e.sim = v
	e.mu.Unlock()
}

func (e *Engine) Simulation() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sim
}

// IsActive indica si el sistema ya está en el estado que el tweak busca.
func (e *Engine) isActive(t Tweak) bool {
	if e.sim {
		return e.simApplied[t.ID]
	}
	if t.Power {
		g, err := activeScheme()
		return err == nil && g == schemeHighPerf
	}
	for _, c := range expand(t) {
		st, err := readState(c)
		if err != nil || !sameState(st, desired(c)) {
			return false
		}
	}
	return len(expand(t)) > 0
}

type TweakView struct {
	Tweak
	Active   bool `json:"active"`   // el sistema está en el estado del tweak
	CanUndo  bool `json:"canUndo"`  // hay backup para revertir
	Blocked  bool `json:"blocked"`  // requiere admin y no lo tenemos
	Disabled bool `json:"disabled"` // no aplica a este equipo (p. ej. no hay GPU de ese fabricante)
}

func (e *Engine) List() []TweakView {
	e.mu.Lock()
	defer e.mu.Unlock()
	adm := isAdmin()
	var out []TweakView
	for _, t := range catalog() {
		_, has := e.st.Items[t.ID]
		out = append(out, TweakView{
			Tweak:    t,
			Active:   e.isActive(t),
			CanUndo:  has || (e.sim && e.simApplied[t.ID]),
			Blocked:  t.Admin && !adm && !e.sim,
			Disabled: !t.Power && len(expand(t)) == 0,
		})
	}
	return out
}

func (e *Engine) Apply(id string) (err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	defer func() { e.log("apply", id, err) }()

	t, ok := findTweak(id)
	if !ok {
		return fmt.Errorf("ajuste desconocido: %s", id)
	}
	if e.sim {
		e.simApplied[id] = true
		return nil
	}
	if t.Admin && !isAdmin() {
		return errNeedAdmin
	}
	if !t.Power && len(expand(t)) == 0 {
		return errors.New("este ajuste no aplica a tu equipo")
	}
	bk, had := e.st.Items[id]
	if !had {
		bk = &Backup{At: time.Now().Format(time.RFC3339)}
		if t.Power {
			g, err := activeScheme()
			if err != nil {
				return err
			}
			bk.PrevPower = g
		} else {
			for _, c := range expand(t) {
				orig, err := readState(c)
				if err != nil {
					return err
				}
				bk.Items = append(bk.Items, BackupItem{Change: c, Orig: orig})
			}
		}
		e.st.Items[id] = bk
		if err := e.save(); err != nil {
			delete(e.st.Items, id)
			return fmt.Errorf("no se pudo guardar el backup, no se aplicó nada: %w", err)
		}
	}
	if err := e.doApply(t); err != nil {
		rbErr := e.restore(bk)
		if rbErr == nil {
			delete(e.st.Items, id)
			e.save()
			return fmt.Errorf("%w (se revirtió lo hecho)", err)
		}
		return fmt.Errorf("%w; además falló el rollback: %v (el backup se conservó)", err, rbErr)
	}
	return nil
}

func (e *Engine) doApply(t Tweak) error {
	if t.Power {
		if err := setScheme(schemeHighPerf); err != nil {
			return err
		}
		g, err := activeScheme()
		if err != nil || g != schemeHighPerf {
			return errors.New("no se pudo verificar el plan de energía")
		}
		return nil
	}
	for _, c := range expand(t) {
		if err := writeState(c, desired(c)); err != nil {
			return fmt.Errorf("%s\\%s\\%s: %w", c.Root, c.Path, c.Name, err)
		}
		st, err := readState(c)
		if err != nil || !sameState(st, desired(c)) {
			return fmt.Errorf("verificación fallida en %s\\%s\\%s", c.Root, c.Path, c.Name)
		}
	}
	return nil
}

func (e *Engine) restore(bk *Backup) error {
	var errs []error
	if bk.PrevPower != "" {
		if err := setScheme(bk.PrevPower); err != nil {
			errs = append(errs, err)
		}
	}
	for _, it := range bk.Items {
		if err := writeState(it.Change, it.Orig); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (e *Engine) Revert(id string) (err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	defer func() { e.log("revert", id, err) }()
	return e.revertLocked(id)
}

func (e *Engine) revertLocked(id string) error {
	if e.sim {
		delete(e.simApplied, id)
		return nil
	}
	bk, ok := e.st.Items[id]
	if !ok {
		return errors.New("no hay backup de este ajuste, no se toca nada")
	}
	if t, found := findTweak(id); found && t.Admin && !isAdmin() {
		return errNeedAdmin
	}
	if err := e.restore(bk); err != nil {
		return fmt.Errorf("no se pudo restaurar todo: %w (el backup se conservó)", err)
	}
	delete(e.st.Items, id)
	return e.save()
}

func (e *Engine) RevertAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	var errs []error
	ids := make([]string, 0)
	if e.sim {
		for id := range e.simApplied {
			ids = append(ids, id)
		}
	} else {
		for id := range e.st.Items {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		err := e.revertLocked(id)
		e.log("revert", id, err)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}
