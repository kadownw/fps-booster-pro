package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows/registry"
)

// Plan de energía "South's Core power plan".
//
// Se crea como un plan nuevo (nunca se modifica el plan del usuario) a partir de una tabla de
// valores (powerplan_south_table.go). Al revertir, vuelve el plan que el usuario tenía activo y este
// se borra. Reglas de aplicación:
//   - Con corriente (AC) se aplican los valores del plan. Con batería (DC) solo los de hardware
//     (procesador, discos, USB, PCIe, Wi-Fi...): suspensión y pantalla en batería no se tocan.
//   - No se tocan los grupos "Batería" ni "Botones de inicio/apagado y tapa": son acciones del equipo
//     (qué hace la tapa o el botón de encendido) y no tienen relación con el rendimiento.
//   - En CPUs híbridas (P/E) y Ryzen X3D de dos CCD no se tocan los ajustes de estacionamiento de
//     núcleos: ahí Windows los usa para elegir los núcleos correctos.

const (
	schemeBalanced = "381b4222-f694-41f0-9685-ff5bb260df2e"
)

var (
	x3dDualCCD = regexp.MustCompile(`(?i)ryzen 9 \d{4}x3d`)
	hexIdxRe   = regexp.MustCompile(`0x([0-9a-fA-F]{8})`)
)

func systemInfoCPUName() string {
	return regString(registry.LOCAL_MACHINE, cpuKeyPath, "ProcessorNameString")
}

func cpuIsHybrid() bool { return cpuDetail().Hybrid }

// cpuNeedsParking: CPUs donde el estacionamiento de núcleos es parte del funcionamiento correcto.
func cpuNeedsParking() bool {
	if x3dDualCCD.MatchString(systemInfoCPUName()) {
		return true
	}
	return cpuIsHybrid()
}

var skipGroups = map[string]bool{
	"Battery":                           true,
	"Power buttons and lid":             true,
	"Settings belonging to no subgroup": false,
}

// Grupos donde también se aplica el valor con batería (DC): solo hardware.
var dcGroups = map[string]bool{
	"Processor power management": true, "Hard disk": true, "USB settings": true, "PCI Express": true,
	"Wireless Adapter Settings": true, "Idle Resiliency": true, "Interrupt Steering Settings": true,
	"Graphics settings": true,
}

// southSettings devuelve la tabla filtrada para este equipo y si cada ajuste lleva valor DC.
func southSettings(parkingMatters bool) []southSetting {
	var out []southSetting
	for _, s := range southPlanTable {
		if skipGroups[s.Group] {
			continue
		}
		if parkingMatters && strings.Contains(strings.ToLower(s.Name), "core parking") {
			continue
		}
		out = append(out, s)
	}
	return out
}

var (
	schemesMu  sync.Mutex
	schemesAt  time.Time
	schemesMap map[string]string
)

// listSchemes devuelve GUID -> nombre de los planes instalados (caché corta).
func listSchemes() map[string]string {
	schemesMu.Lock()
	defer schemesMu.Unlock()
	if schemesMap != nil && time.Since(schemesAt) < 10*time.Second {
		return schemesMap
	}
	m := map[string]string{}
	out, err := powercfg("/list")
	if err == nil {
		for _, ln := range strings.Split(out, "\n") {
			g := guidRe.FindString(ln)
			if g == "" {
				continue
			}
			name := ""
			if i, j := strings.Index(ln, "("), strings.LastIndex(ln, ")"); i >= 0 && j > i {
				name = ln[i+1 : j]
			}
			m[strings.ToLower(g)] = name
		}
	}
	schemesMap, schemesAt = m, time.Now()
	return m
}

func invalidateSchemes() {
	schemesMu.Lock()
	schemesMap = nil
	schemesMu.Unlock()
}

func schemeExists(g string) bool { _, ok := listSchemes()[strings.ToLower(g)]; return ok }

func validGUID(g string) bool { return len(g) == 36 && guidRe.MatchString(g) }

// ensureSouthPlan crea (si hace falta) y configura el plan. Devuelve los ajustes que este equipo
// no soporta (se omiten sin fallar: depende del hardware y de la versión de Windows).
func ensureSouthPlan() (skipped []string, err error) {
	if !schemeExists(southPlanGUID) {
		src := ""
		for _, c := range []string{schemeBalanced, schemeHighPerf} {
			if schemeExists(c) {
				src = c
				break
			}
		}
		if src == "" { // equipos donde otra herramienta borró los planes estándar
			if src, err = activeScheme(); err != nil {
				return nil, err
			}
		}
		if out, err := powercfg("/duplicatescheme", src, southPlanGUID); err != nil {
			return nil, fmt.Errorf("no se pudo crear el plan: %s", strings.TrimSpace(out))
		}
		invalidateSchemes()
		powercfg("/changename", southPlanGUID, southPlanName,
			"Plan de rendimiento de South Tweaks. Se borra al revertir y vuelve tu plan anterior.")
	}
	for _, v := range southSettings(cpuNeedsParking()) {
		if !validGUID(v.Sub) || !validGUID(v.Setting) {
			return nil, errors.New("GUID inválido en la tabla del plan")
		}
		if _, err := powercfg("/setacvalueindex", southPlanGUID, v.Sub, v.Setting, strconv.FormatUint(uint64(v.AC), 10)); err != nil {
			skipped = append(skipped, v.Name) // el ajuste no existe en este equipo
			continue
		}
		if dcGroups[v.Group] {
			powercfg("/setdcvalueindex", southPlanGUID, v.Sub, v.Setting, strconv.FormatUint(uint64(v.DC), 10))
		}
	}
	return skipped, nil
}

// readPlanValue lee el valor AC (dc=false) o DC (dc=true) de un ajuste del plan.
func readPlanValue(scheme, sub, setting string, dc bool) (uint32, bool) {
	out, err := powercfg("/qh", scheme, sub, setting)
	if err != nil {
		return 0, false
	}
	// En cualquier idioma, los dos últimos valores 0x######## del bloque son el índice actual
	// con corriente (AC) y con batería (DC), en ese orden.
	all := hexIdxRe.FindAllStringSubmatch(out, -1)
	if len(all) < 2 {
		return 0, false
	}
	idx := len(all) - 2
	if dc {
		idx = len(all) - 1
	}
	v, err := strconv.ParseUint(all[idx][1], 16, 32)
	return uint32(v), err == nil
}

func readACValue(scheme, sub, setting string) (uint32, bool) {
	return readPlanValue(scheme, sub, setting, false)
}

func deleteScheme(g string) error {
	if !validGUID(g) {
		return fmt.Errorf("GUID inválido: %q", g)
	}
	if out, err := powercfg("/delete", g); err != nil {
		return fmt.Errorf("no se pudo borrar el plan: %s", strings.TrimSpace(out))
	}
	invalidateSchemes()
	return nil
}

// planTarget: GUID del plan que activa un ajuste de energía.
func planTarget(t Tweak) string {
	if t.PlanID == "south" {
		return southPlanGUID
	}
	return schemeHighPerf
}

// runAllowedPowercfg ejecuta solo comandos de powercfg de una lista blanca fija.
func runAllowedPowercfg(args []string) error {
	allowed := map[string]bool{"/hibernate off": true, "/hibernate on": true}
	key := strings.Join(args, " ")
	if !allowed[key] {
		return fmt.Errorf("comando no permitido: powercfg %s", key)
	}
	if out, err := powercfg(args...); err != nil {
		return fmt.Errorf("powercfg %s: %s", key, strings.TrimSpace(out))
	}
	return nil
}

// ---- comandos de PowerShell fijos ----

// allowedPS: solo se ejecutan los comandos Apply/Revert declarados en el catálogo (texto fijo del código).
func allowedPS(cmd string) bool {
	for _, t := range catalog() {
		if t.PS != nil && (t.PS.Apply == cmd || t.PS.Revert == cmd) {
			return true
		}
	}
	return false
}

func runAllowedPS(cmd string) error {
	if !allowedPS(cmd) {
		return fmt.Errorf("comando de PowerShell no permitido: %q", cmd)
	}
	if out, err := runPS(cmd); err != nil {
		return fmt.Errorf("%s: %s", cmd, strings.TrimSpace(out))
	}
	return nil
}

func psState(s *PSSpec) (string, error) {
	out, err := runPS(s.State)
	if err != nil {
		return "", fmt.Errorf("no se pudo leer el estado: %s", strings.TrimSpace(out))
	}
	return strings.TrimSpace(out), nil
}

func psSpecForRevert(revert string) *PSSpec {
	for _, t := range catalog() {
		if t.PS != nil && t.PS.Revert == revert {
			return t.PS
		}
	}
	return nil
}
