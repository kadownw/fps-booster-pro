package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Acciones únicas: se ejecutan una vez y no dejan un valor que se pueda restaurar. Por eso no
// tienen interruptor sino un botón «Aplicar» con confirmación. Cada acción es una receta fija
// (nunca se arma un comando con texto del usuario) y solo corre con permisos de administrador
// cuando la receta lo exige.

var driveRe = regexp.MustCompile(`^[A-Z]:$`)

func loadRuns(dir string) map[string]string {
	m := map[string]string{}
	if b, err := os.ReadFile(filepath.Join(dir, "actions.json")); err == nil {
		json.Unmarshal(b, &m) // si está dañado se empieza vacío
	}
	return m
}

func (e *Engine) saveRuns() {
	b, err := json.MarshalIndent(e.runs, "", "  ")
	if err != nil {
		return
	}
	p := filepath.Join(e.dir, "actions.json")
	if os.WriteFile(p+".tmp", b, 0o600) == nil {
		os.Rename(p+".tmp", p)
	}
}

func runExe(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// RunOnce ejecuta una acción única y devuelve un resumen legible.
func (e *Engine) RunOnce(id string) (summary string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	defer func() { e.log("run", id, err) }()

	t, ok := findTweak(id)
	if !ok || !t.Once {
		return "", fmt.Errorf("%q no es una acción única", id)
	}
	if e.sim {
		return "Simulación: no se ejecutó nada.", nil
	}
	if t.Admin && !isAdmin() {
		return "", errNeedAdmin
	}
	switch id {
	case "flush_dns":
		if out, err := runExe("ipconfig", "/flushdns"); err != nil {
			return "", fmt.Errorf("ipconfig /flushdns: %s", out)
		}
		summary = "Caché de DNS vaciada."
	case "retrim_ssd":
		var done, failed []string
		for _, d := range disks() {
			if !driveRe.MatchString(d.Letter) {
				continue
			}
			// defrag /L = ReTrim: en un HDD responde con error y se omite, no daña nada.
			if _, err := runExe("defrag.exe", d.Letter, "/L"); err != nil {
				failed = append(failed, d.Letter)
			} else {
				done = append(done, d.Letter)
			}
		}
		if len(done) == 0 {
			return "", errors.New("no se encontró ningún SSD compatible (los discos HDD se omiten)")
		}
		summary = "TRIM ejecutado en " + strings.Join(done, ", ")
		if len(failed) > 0 {
			summary += " (omitidos: " + strings.Join(failed, ", ") + ")"
		}
	default:
		return "", fmt.Errorf("acción sin receta: %s", id)
	}
	e.runs[id] = time.Now().Format(time.RFC3339)
	e.saveRuns()
	return summary, nil
}
