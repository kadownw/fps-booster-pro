package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	// Tasks: estado original de cada tarea programada ("Ready", "Disabled"...).
	Tasks map[string]string `json:"tasks,omitempty"`
	// DeletePlan: plan de energía creado por la app, que se borra al revertir.
	DeletePlan string `json:"deletePlan,omitempty"`
	// Nvidia: valores previos del perfil global del driver NVIDIA.
	Nvidia []NvOrig `json:"nvidia,omitempty"`
	// Cmd: estado previo y argumentos para volver (solo powercfg).
	CmdOrig   *RegState `json:"cmdOrig,omitempty"`
	PSOrig    *string   `json:"psOrig,omitempty"`
	PSRevert  string    `json:"psRevert,omitempty"`
	CmdRevert []string  `json:"cmdRevert,omitempty"`
	At        string    `json:"at"`
}

// ensure agrega al backup lo que todavía no tiene (original de cada valor y tarea del ajuste).
// Devuelve true si cambió. Se llama antes de tocar nada.
func (bk *Backup) ensure(t Tweak) (bool, error) {
	changed := false
	if t.Power && bk.PrevPower == "" {
		g, err := activeScheme()
		if err != nil {
			return false, err
		}
		bk.PrevPower, changed = g, true
		if t.PlanID == "south" && !schemeExists(southPlanGUID) {
			bk.DeletePlan = southPlanGUID // lo crea la app: al revertir se borra
		}
	}
	if len(t.Nvidia) > 0 && bk.Nvidia == nil {
		orig, err := nvReadOriginals(t.Nvidia)
		if err != nil {
			return false, err
		}
		bk.Nvidia, changed = orig, true
	}
	if t.PS != nil && bk.PSOrig == nil {
		cur, err := psState(t.PS)
		if err != nil {
			return false, err
		}
		bk.PSOrig, bk.PSRevert, changed = &cur, t.PS.Revert, true
	}
	if t.Cmd != nil && bk.CmdOrig == nil {
		st, err := readState(t.Cmd.State)
		if err != nil {
			return false, err
		}
		bk.CmdOrig, bk.CmdRevert, changed = &st, t.Cmd.Revert, true
	}
	known := map[string]bool{}
	for _, it := range bk.Items {
		known[it.Change.Root+"\\"+it.Change.Path+"\\"+it.Change.Name] = true
	}
	for _, c := range expand(t) {
		if known[c.Root+"\\"+c.Path+"\\"+c.Name] {
			continue
		}
		orig, err := readState(c)
		if err != nil {
			return false, err
		}
		bk.Items = append(bk.Items, BackupItem{Change: c, Orig: orig})
		changed = true
	}
	for p, st := range existingTasks(t) {
		if _, ok := bk.Tasks[p]; ok {
			continue
		}
		if bk.Tasks == nil {
			bk.Tasks = map[string]string{}
		}
		bk.Tasks[p], changed = st, true
	}
	return changed, nil
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
	runs       map[string]string // última ejecución de acciones únicas (RFC3339)
	tier       func() string     // nivel de la cuenta (nil = sin restricciones, p. ej. en pruebas)
}

func NewEngine() (*Engine, error) {
	base, err := os.UserCacheDir() // %LOCALAPPDATA%
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, "SouthTweaks")
	if _, err := os.Stat(dir); os.IsNotExist(err) { // nombre anterior de la app: se migran backups y registros
		if old := filepath.Join(base, "FPSBoosterPro"); dirExists(old) {
			_ = os.Rename(old, dir)
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	e := &Engine{dir: dir, st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: loadRuns(dir)}
	if b, err := os.ReadFile(filepath.Join(dir, "backup.json")); err == nil {
		if err := json.Unmarshal(b, &e.st); err != nil {
			return nil, fmt.Errorf("backup.json dañado (no se toca nada): %w", err)
		}
		if e.st.Items == nil {
			e.st.Items = map[string]*Backup{}
		}
		if migrateBackups(&e.st) {
			_ = e.save()
		}
	}
	return e, nil
}

// tweakAliases: ajustes que se unieron en otro. Si el usuario tenía aplicado el viejo, su backup pasa al
// nuevo para que se pueda revertir desde la tarjeta unificada.
var tweakAliases = map[string]string{
	"nvidia_perf_profile": "nvidia_profile", "nv_pstate": "nvidia_profile",
	"mouse_accel": "mouse_raw_curve", "bg_apps_policy": "bg_apps", "update_pause": "updates_pause",
	"telemetry_tasks": "telemetry", "svc_telemetry": "telemetry", "autologgers": "telemetry",
	"error_tasks": "error_reporting", "svc_error_reporting": "error_reporting",
	"search_web": "search_local", "search_cloud_off": "search_local",
	"svc_unneeded": "svc_max_performance", "svc_background_manual": "svc_max_performance",
	"svc_search_manual": "svc_max_performance", "svc_print_manual": "svc_max_performance",
}

// mergeBackup agrega a dst lo que src guardó y dst todavía no tiene (el valor original más antiguo gana).
func mergeBackup(dst, src *Backup) {
	known := map[string]bool{}
	for _, it := range dst.Items {
		known[it.Change.Root+"\\"+it.Change.Path+"\\"+it.Change.Name] = true
	}
	for _, it := range src.Items {
		if k := it.Change.Root + "\\" + it.Change.Path + "\\" + it.Change.Name; !known[k] {
			dst.Items = append(dst.Items, it)
			known[k] = true
		}
	}
	for p, st := range src.Tasks {
		if _, ok := dst.Tasks[p]; !ok {
			if dst.Tasks == nil {
				dst.Tasks = map[string]string{}
			}
			dst.Tasks[p] = st
		}
	}
	have := map[uint32]bool{}
	for _, o := range dst.Nvidia {
		have[o.ID] = true
	}
	for _, o := range src.Nvidia {
		if !have[o.ID] {
			dst.Nvidia = append(dst.Nvidia, o)
		}
	}
	if dst.PrevPower == "" {
		dst.PrevPower, dst.DeletePlan = src.PrevPower, src.DeletePlan
	}
	if dst.At == "" || (src.At != "" && src.At < dst.At) {
		dst.At = src.At
	}
}

// migrateBackups mueve los backups de ids viejos al ajuste que los absorbió. Devuelve true si cambió algo.
func migrateBackups(st *store) bool {
	changed := false
	for old, now := range tweakAliases {
		bk, ok := st.Items[old]
		if !ok {
			continue
		}
		dst, ok := st.Items[now]
		if !ok {
			dst = &Backup{At: bk.At}
			st.Items[now] = dst
		}
		mergeBackup(dst, bk)
		delete(st.Items, old)
		changed = true
	}
	return changed
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
	if t.Once {
		return false // una acción única no deja un estado que se pueda consultar
	}
	if e.sim {
		return e.simApplied[t.ID]
	}
	if t.Power {
		g, err := activeScheme()
		return err == nil && g == planTarget(t)
	}
	if len(t.Nvidia) > 0 {
		if !nvIsActive(t.Nvidia) {
			return false
		}
		if len(t.Changes) == 0 {
			return true
		}
		// ajuste mixto (perfil del driver + valores de registro): sigue con los valores de registro
	}
	if t.PS != nil {
		cur, err := psState(t.PS)
		return err == nil && cur == t.PS.Want
	}
	if t.Cmd != nil {
		st, err := readState(t.Cmd.State)
		return err == nil && sameState(st, desired(t.Cmd.State))
	}
	regs := expand(t)
	for _, c := range regs {
		st, err := readState(c)
		if err != nil || !satisfied(c, st) {
			return false
		}
	}
	tasks := existingTasks(t)
	for _, st := range tasks {
		if st != "Disabled" {
			return false
		}
	}
	return len(regs)+len(tasks) > 0
}

type TweakView struct {
	Tweak
	Active   bool   `json:"active"`            // el sistema está en el estado del tweak
	CanUndo  bool   `json:"canUndo"`           // hay backup para revertir
	Blocked  bool   `json:"blocked"`           // requiere admin y no lo tenemos
	Disabled bool   `json:"disabled"`          // no aplica a este equipo (p. ej. no hay GPU de ese fabricante)
	LastRun  string `json:"lastRun,omitempty"` // última vez que se ejecutó una acción única
	Locked   bool   `json:"locked"`            // el nivel de la cuenta no lo incluye (Free Tier)
	Status   string `json:"status"`            // applied | partial | not_applied | unavailable | once
	Detail   string `json:"detail"`            // explicación del estado, leída del sistema
}

func (e *Engine) List() []TweakView {
	e.mu.Lock()
	defer e.mu.Unlock()
	adm := isAdmin()
	var out []TweakView
	for _, t := range catalog() {
		_, has := e.st.Items[t.ID]
		active := e.isActive(t)
		diag := e.diagnose(t, active)
		out = append(out, TweakView{
			Tweak:    t,
			Active:   active,
			Status:   diag.Status,
			Detail:   diag.Detail,
			CanUndo:  has || (e.sim && e.simApplied[t.ID]),
			Blocked:  t.Admin && !adm && !e.sim,
			Disabled: !supported(t),
			LastRun:  e.runs[t.ID],
			Locked:   e.tier != nil && !tweakAllowed(e.tier(), t.ID),
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
	if e.tier != nil && !tweakAllowed(e.tier(), id) {
		return errTier
	}
	if t.Once {
		return errors.New("esta es una acción única: usá el botón Aplicar")
	}
	if e.sim {
		e.simApplied[id] = true
		return nil
	}
	if t.Admin && !isAdmin() {
		return errNeedAdmin
	}
	if !supported(t) {
		return errors.New("este ajuste no aplica a tu equipo")
	}
	// Los planes de energía se excluyen entre sí: antes de activar uno se revierte el otro, así
	// cada backup guarda el plan original del usuario.
	if t.Power {
		for _, o := range catalog() {
			if o.Power && o.ID != id {
				if _, applied := e.st.Items[o.ID]; applied {
					if err := e.revertLocked(o.ID); err != nil {
						return fmt.Errorf("no se pudo revertir %q antes de cambiar de plan: %w", o.Title, err)
					}
				}
			}
		}
	}
	bk, had := e.st.Items[id]
	if !had {
		bk = &Backup{At: time.Now().Format(time.RFC3339)}
	}
	changed, err := bk.ensure(t)
	if err != nil {
		return err
	}
	if !had || changed {
		e.st.Items[id] = bk
		if err := e.save(); err != nil {
			if !had {
				delete(e.st.Items, id)
			}
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
		target := planTarget(t)
		if t.PlanID == "south" {
			if _, err := ensureSouthPlan(); err != nil {
				return err
			}
		}
		if err := setScheme(target); err != nil {
			return err
		}
		g, err := activeScheme()
		if err != nil || g != target {
			return errors.New("no se pudo verificar el plan de energía")
		}
		return nil
	}
	if len(t.Nvidia) > 0 {
		if err := nvApply(t.Nvidia); err != nil {
			return err
		}
		if len(t.Changes) == 0 {
			return nil
		}
		// ajuste mixto: después del perfil del driver se escriben los valores de registro
	}
	if t.PS != nil {
		if err := runAllowedPS(t.PS.Apply); err != nil {
			return err
		}
		if cur, err := psState(t.PS); err != nil || cur != t.PS.Want {
			return errors.New("no se pudo verificar el cambio")
		}
		return nil
	}
	if t.Cmd != nil {
		if err := runAllowedPowercfg(t.Cmd.Apply); err != nil {
			return err
		}
		if st, err := readState(t.Cmd.State); err != nil || !sameState(st, desired(t.Cmd.State)) {
			return errors.New("no se pudo verificar el cambio")
		}
		return nil
	}
	for _, c := range expand(t) {
		want := desired(c)
		if c.KV != "" { // lista clave=valor;: se conserva el resto de las claves
			cur, _ := readState(c)
			want = RegState{Exists: true, Kind: "sz", S: mergeKV(cur.S, c.KV)}
		}
		if err := writeState(c, want); err != nil {
			return fmt.Errorf("%s\\%s\\%s: %w", c.Root, c.Path, c.Name, err)
		}
		st, err := readState(c)
		if err != nil || !sameState(st, want) {
			return fmt.Errorf("verificación fallida en %s\\%s\\%s", c.Root, c.Path, c.Name)
		}
	}
	for p, st := range existingTasks(t) {
		if st == "Disabled" {
			continue
		}
		if err := setTask(p, false); err != nil {
			return err
		}
		if now := existingTasks(t)[p]; now != "Disabled" {
			return fmt.Errorf("verificación fallida en la tarea %s (estado %q)", p, now)
		}
	}
	return nil
}

func (e *Engine) restore(bk *Backup) error {
	var errs []error
	if bk.PrevPower != "" {
		if err := setScheme(bk.PrevPower); err != nil {
			errs = append(errs, err)
		} else if bk.DeletePlan != "" && bk.DeletePlan != bk.PrevPower && schemeExists(bk.DeletePlan) {
			if err := deleteScheme(bk.DeletePlan); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if bk.Nvidia != nil {
		if err := nvRestore(bk.Nvidia); err != nil {
			errs = append(errs, err)
		}
	}
	if bk.PSOrig != nil && bk.PSRevert != "" {
		if spec := psSpecForRevert(bk.PSRevert); spec != nil && *bk.PSOrig != spec.Want {
			if err := runAllowedPS(bk.PSRevert); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if bk.CmdOrig != nil && len(bk.CmdRevert) > 0 {
		// Solo se ejecuta el comando de vuelta si el estado original era distinto del aplicado
		// (p. ej., la hibernación estaba activada antes).
		if bk.CmdOrig.Exists && bk.CmdOrig.D != 0 {
			if err := runAllowedPowercfg(bk.CmdRevert); err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, it := range bk.Items {
		if err := writeState(it.Change, it.Orig); err != nil {
			errs = append(errs, err)
		}
	}
	for p, st := range bk.Tasks {
		if st == "Disabled" {
			continue // ya estaba deshabilitada antes: se deja así
		}
		if err := setTask(p, true); err != nil {
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
	if bad := verifyRestore(bk); len(bad) > 0 {
		return fmt.Errorf("se intentó restaurar pero al volver a leer %s no coincide con el original (el backup se conservó)", strings.Join(bad, ", "))
	}
	delete(e.st.Items, id)
	return e.save()
}

// verifyRestore relee cada valor del backup y devuelve los que NO volvieron a su original.
func verifyRestore(bk *Backup) []string {
	var bad []string
	for _, it := range bk.Items {
		st, err := readState(it.Change)
		if err != nil || !sameState(st, it.Orig) {
			bad = append(bad, it.Change.Name)
		}
	}
	if len(bad) > 3 {
		bad = append(bad[:3], "…")
	}
	return bad
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

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
