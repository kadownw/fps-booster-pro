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

// runPSEnv ejecuta un script FIJO de PowerShell pasándole datos por variables de entorno (nunca
// dentro del texto del script), para que ningún valor externo pueda inyectar comandos.
func runPSEnv(script string, env ...string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// RunOnce ejecuta una acción única y devuelve un resumen que describe lo que REALMENTE pasó:
// cada acción vuelve a consultar el sistema después de actuar. Si no había nada que hacer, o la
// consulta falló, no se registra como ejecutada.
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
	noop := false // true = no había nada que hacer: no se marca como ejecutada
	switch id {
	case "flush_dns":
		if out, err := runExe("ipconfig", "/flushdns"); err != nil {
			return "", fmt.Errorf("ipconfig /flushdns falló: %s", out)
		}
		summary = "ipconfig /flushdns terminó con éxito: la caché de DNS se vació."
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
			return "", errors.New("defrag /L no pudo ejecutarse en ninguna unidad: no hay SSD compatible, o el servicio «Optimizar unidades» (defragsvc) está deshabilitado")
		}
		summary = "defrag /L terminó con éxito (código 0) en " + strings.Join(done, ", ")
		if len(failed) > 0 {
			summary += ". No se pudo en " + strings.Join(failed, ", ") + " (discos HDD o sin soporte)"
		}
	case "remove_copilot_app":
		summary, noop, err = removeAppxReport("*Copilot*", "Copilot")
		if err != nil {
			return "", err
		}
	case "remove_gamebar_app":
		// Solo la superposición Xbox Game Bar: NO se tocan Gaming Services ni la app de Xbox
		// (los necesitan Game Pass y varios juegos de la Tienda).
		summary, noop, err = removeAppxReport("Microsoft.XboxGamingOverlay", "Xbox Game Bar")
		if err != nil {
			return "", err
		}
	case "remove_bloat_apps":
		summary, noop, err = removeBloat()
		if err != nil {
			return "", err
		}
	case "bitlocker_off":
		summary, noop, err = bitlockerOff()
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("acción sin receta: %s", id)
	}
	if !noop {
		e.runs[id] = time.Now().Format(time.RFC3339)
		e.saveRuns()
	}
	return summary, nil
}

// ---- BitLocker ----

type blVolume struct {
	Mount, Protection, Status, Pct string
}

func (v blVolume) encrypted() bool {
	return strings.EqualFold(v.Protection, "On") || v.Protection == "1" || !strings.EqualFold(v.Status, "FullyDecrypted")
}

// parseBitLocker interpreta las líneas «unidad|protección|estado|%» del script de consulta.
func parseBitLocker(out string) ([]blVolume, error) {
	var vols []blVolume
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "ERROR|") {
			return nil, errors.New(strings.TrimSpace(strings.TrimPrefix(line, "ERROR|")))
		}
		f := strings.Split(line, "|")
		if len(f) < 3 {
			return nil, fmt.Errorf("respuesta inesperada de BitLocker: %q", line)
		}
		v := blVolume{Mount: strings.TrimSpace(f[0]), Protection: strings.TrimSpace(f[1]), Status: strings.TrimSpace(f[2])}
		if len(f) > 3 {
			v.Pct = strings.TrimSpace(f[3])
		}
		vols = append(vols, v)
	}
	return vols, nil
}

const blQueryScript = `$ErrorActionPreference = 'Stop'
try { Get-BitLockerVolume | ForEach-Object { '{0}|{1}|{2}|{3}' -f $_.MountPoint, $_.ProtectionStatus, $_.VolumeStatus, $_.EncryptionPercentage } }
catch { 'ERROR|' + ($_.Exception.Message -replace '\s+', ' '); exit 3 }`

var mountRe = regexp.MustCompile(`^[A-Za-z]:\\?$`)

func queryBitLocker() ([]blVolume, error) {
	out, err := runPSEnv(blQueryScript)
	vols, perr := parseBitLocker(out)
	if perr != nil {
		return nil, perr
	}
	if err != nil && len(vols) == 0 {
		return nil, fmt.Errorf("PowerShell terminó con error: %v", err)
	}
	return vols, nil
}

func bitlockerOff() (summary string, noop bool, err error) {
	vols, qerr := queryBitLocker()
	if qerr != nil {
		return "", false, fmt.Errorf("No se pudo consultar BitLocker en este equipo, así que no se cambió nada. Motivo: %v. (Suele pasar en ediciones de Windows sin BitLocker o con el servicio BDESVC deshabilitado.)", qerr)
	}
	var todo []blVolume
	for _, v := range vols {
		if v.encrypted() {
			todo = append(todo, v)
		}
	}
	if len(todo) == 0 {
		return fmt.Sprintf("BitLocker no está activo en ninguna de las %d unidades que Windows informó: no había nada que descifrar. No se cambió nada.", len(vols)), true, nil
	}
	var problems []string
	for _, v := range todo {
		if !mountRe.MatchString(v.Mount) {
			problems = append(problems, v.Mount+": punto de montaje no válido")
			continue
		}
		if out, derr := runPSEnv(`$ErrorActionPreference='Stop'; try { Disable-BitLocker -MountPoint $env:ST_MP | Out-Null } catch { 'ERROR|' + ($_.Exception.Message -replace '\s+',' '); exit 3 }`, "ST_MP="+v.Mount); derr != nil {
			problems = append(problems, v.Mount+": "+strings.TrimPrefix(out, "ERROR|"))
		}
	}
	after, qerr := queryBitLocker()
	if qerr != nil {
		return "", false, fmt.Errorf("se pidió quitar BitLocker pero no se pudo comprobar el resultado: %v", qerr)
	}
	var ok []string
	for _, v := range todo {
		for _, a := range after {
			if !strings.EqualFold(a.Mount, v.Mount) {
				continue
			}
			switch {
			case strings.EqualFold(a.Status, "FullyDecrypted"):
				ok = append(ok, a.Mount+" ya está descifrada")
			case strings.EqualFold(a.Status, "DecryptionInProgress"):
				ok = append(ok, a.Mount+" se está descifrando (sigue en segundo plano)")
			default:
				problems = append(problems, fmt.Sprintf("%s sigue en estado «%s»", a.Mount, a.Status))
			}
		}
	}
	if len(problems) > 0 {
		return "", false, fmt.Errorf("BitLocker no se pudo quitar en todas las unidades. %s. Lo que sí avanzó: %s", strings.Join(problems, "; "), strings.Join(ok, "; "))
	}
	return "Verificado con Windows: " + strings.Join(ok, "; ") + ".", false, nil
}

// ---- paquetes AppX ----

var appxPatternRe = regexp.MustCompile(`^[A-Za-z0-9.*]+$`)

func appxList(pattern string) ([]string, error) {
	if !appxPatternRe.MatchString(pattern) { // el patrón es fijo del código; se valida igual
		return nil, fmt.Errorf("patrón de paquete no permitido: %q", pattern)
	}
	out, err := runPSEnv(`$ErrorActionPreference='Stop'; try { Get-AppxPackage -AllUsers -Name $env:ST_PKG | ForEach-Object { $_.PackageFullName } } catch { 'ERROR|' + ($_.Exception.Message -replace '\s+',' '); exit 3 }`, "ST_PKG="+pattern)
	var pk []string
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "ERROR|") {
			return nil, errors.New(strings.TrimPrefix(l, "ERROR|"))
		}
		if l != "" {
			pk = append(pk, l)
		}
	}
	if err != nil && len(pk) == 0 {
		return nil, fmt.Errorf("PowerShell terminó con error: %v", err)
	}
	return pk, nil
}

// removeAppxReport quita los paquetes que coinciden con un patrón fijo y VUELVE A CONSULTAR cuántos
// quedan: el resultado se informa tal cual, sin dar por hecho que se quitó.
func removeAppxReport(pattern, label string) (summary string, noop bool, err error) {
	before, err := appxList(pattern)
	if err != nil {
		return "", false, fmt.Errorf("No se pudo consultar los paquetes de %s, así que no se cambió nada. Motivo: %v", label, err)
	}
	if len(before) == 0 {
		return fmt.Sprintf("%s no está instalado en este Windows: no había nada que desinstalar. No se cambió nada.", label), true, nil
	}
	var errs []string
	for _, pkg := range before {
		out, rerr := runPSEnv(`$ErrorActionPreference='Stop'; try { Remove-AppxPackage -Package $env:ST_PKGFULL -AllUsers } catch { 'ERROR|' + ($_.Exception.Message -replace '\s+',' '); exit 3 }`, "ST_PKGFULL="+pkg)
		if rerr != nil {
			errs = append(errs, pkg+": "+strings.TrimPrefix(out, "ERROR|"))
		}
	}
	after, aerr := appxList(pattern)
	if aerr != nil {
		return "", false, fmt.Errorf("se intentó desinstalar %s pero no se pudo comprobar el resultado: %v", label, aerr)
	}
	removed := len(before) - len(after)
	if len(after) > 0 {
		msg := fmt.Sprintf("%s: se quitaron %d de %d paquetes y quedan %d instalados", label, removed, len(before), len(after))
		if len(errs) > 0 {
			msg += ". Windows respondió: " + errs[0]
		}
		return "", false, errors.New(msg)
	}
	return fmt.Sprintf("Verificado con Windows: %s desinstalado (%d paquete/s quitados, 0 quedan).", label, removed), false, nil
}

// removeBloat quita la lista fija de apps y resume lo que Windows confirma de cada una.
func removeBloat() (summary string, noop bool, err error) {
	var removed, absent, failed []string
	for _, b := range bloatApps {
		_, nop, rerr := removeAppxReport(b[0], b[1])
		switch {
		case rerr != nil:
			failed = append(failed, b[1]+" ("+strings.TrimSpace(rerr.Error())+")")
		case nop:
			absent = append(absent, b[1])
		default:
			removed = append(removed, b[1])
		}
	}
	if len(failed) > 0 {
		msg := "Algunas apps no se pudieron quitar: " + strings.Join(failed, "; ") + "."
		if len(removed) > 0 {
			msg += " Sí se quitaron: " + strings.Join(removed, ", ") + "."
		}
		return "", false, errors.New(msg)
	}
	if len(removed) == 0 {
		return fmt.Sprintf("Ninguna de las %d apps de la lista está instalada: no había nada que quitar. No se cambió nada.", len(bloatApps)), true, nil
	}
	return fmt.Sprintf("Verificado con Windows: se quitaron %s. Ya no estaban instaladas: %d.", strings.Join(removed, ", "), len(absent)), false, nil
}
