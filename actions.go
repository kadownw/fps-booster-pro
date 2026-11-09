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
	if e.tier != nil && !tweakAllowed(e.tier(), id) {
		return "", errTier
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
	case "remove_copilot_app":
		n, err := removeAppx("*Copilot*")
		if err != nil {
			return "", err
		}
		summary = fmt.Sprintf("Copilot desinstalado (%d paquete/s).", n)
	case "remove_gamebar_app":
		// Solo la superposición Xbox Game Bar: NO se tocan Gaming Services ni la app de Xbox
		// (los necesitan Game Pass y varios juegos de la Tienda).
		n, err := removeAppx("Microsoft.XboxGamingOverlay")
		if err != nil {
			return "", err
		}
		summary = fmt.Sprintf("Xbox Game Bar desinstalada (%d paquete/s).", n)
	case "bitlocker_off":
		// Receta fija: solo unidades que tienen BitLocker activo o cifrado parcial.
		out, err := runPS(`$v = @(Get-BitLockerVolume -ErrorAction Stop | Where-Object { $_.ProtectionStatus -eq 'On' -or $_.VolumeStatus -ne 'FullyDecrypted' }); $v.Count; $v | ForEach-Object { Disable-BitLocker -MountPoint $_.MountPoint -ErrorAction SilentlyContinue | Out-Null }`)
		if err != nil && strings.TrimSpace(out) == "" {
			return "", errors.New("no se pudo consultar BitLocker (¿tu edición de Windows lo incluye?)")
		}
		first, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
		var n int
		fmt.Sscanf(strings.TrimSpace(first), "%d", &n)
		if n == 0 {
			summary = "Ninguna unidad tenía BitLocker activo."
		} else {
			summary = fmt.Sprintf("BitLocker se está quitando de %d unidad/es; el descifrado sigue en segundo plano.", n)
		}
	default:
		return "", fmt.Errorf("acción sin receta: %s", id)
	}
	e.runs[id] = time.Now().Format(time.RFC3339)
	e.saveRuns()
	return summary, nil
}

// removeAppx desinstala los paquetes AppX cuyo nombre coincide con un patrón FIJO (definido en el
// código, nunca escrito por el usuario) y devuelve cuántos había.
func removeAppx(pattern string) (int, error) {
	if !appxPatternRe.MatchString(pattern) {
		return 0, fmt.Errorf("patrón de paquete no permitido: %q", pattern)
	}
	out, err := runPS(fmt.Sprintf(`$p = @(Get-AppxPackage -AllUsers -Name '%s'); $p.Count; $p | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue`, pattern))
	if err != nil && strings.TrimSpace(out) == "" {
		return 0, errors.New("no se pudo desinstalar (¿falta administrador?)")
	}
	first, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	var n int
	fmt.Sscanf(strings.TrimSpace(first), "%d", &n)
	return n, nil
}

var appxPatternRe = regexp.MustCompile(`^[A-Za-z0-9.*]+$`)
