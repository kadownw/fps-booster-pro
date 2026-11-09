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

// Plan de energía propio: "FPS Booster Pro — Rendimiento estable".
//
// Objetivo: rendimiento cercano al máximo con frametimes estables, sin desactivar los estados
// de reposo del procesador (C-states). La idea es que la CPU suba de frecuencia rápido y baje
// despacio (menos oscilación de reloj entre frames), y que los buses (USB, PCIe, Wi-Fi, disco) no
// entren en ahorro de energía durante el juego. Solo se cambian los valores con corriente (AC);
// con batería se conserva el comportamiento del plan base.
//
// GUIDs de los ajustes verificados con `powercfg /qh` en Windows 11.

const (
	fpsPlanGUID    = "f9b5b0a1-7c3e-4d2a-9b1e-fb0057e5ab1e"
	fpsPlanName    = "FPS Booster Pro - Rendimiento estable"
	schemeBalanced = "381b4222-f694-41f0-9685-ff5bb260df2e"
	subProcessor   = "54533251-82be-4824-96c1-47b60b740d00"
	subUSB         = "2a737441-1930-4402-8d77-b2bebba308a3"
	subPCIe        = "501a4d13-42af-4429-9fd1-a8218c268e20"
	subDisk        = "0012ee47-9041-4b5d-9b77-535fba8b1442"
	subWireless    = "19cbb8fa-5279-450e-9fac-8a3d5fedd0c1"
	setCoreParkMin = "0cc5b647-c1df-4637-891a-dec35c318583"
)

type planValue struct {
	Sub, Setting, Name string
	AC                 uint32
	Why                string
}

func fpsPlanValues() []planValue {
	v := []planValue{
		{subProcessor, "893dee8e-2bef-41e0-89c6-b55d0929964c", "Estado mínimo del procesador", 100, "Evita que la frecuencia caiga entre frames; los estados de reposo (C-states) siguen activos"},
		{subProcessor, "bc5038f7-23e0-4960-96da-33abaf5935ec", "Estado máximo del procesador", 100, ""},
		{subProcessor, "be337238-0d82-4146-a960-4f3749d470c7", "Modo de boost", 2, "Agresivo"},
		{subProcessor, "36687f9e-e3a5-4dbf-b1dc-15eb381c6863", "Preferencia de energía/rendimiento (EPP)", 0, "Prioriza rendimiento"},
		{subProcessor, "465e1f50-b610-473a-ab58-00d1077dc418", "Política de aumento de rendimiento", 2, "Rocket: sube directo al máximo"},
		{subProcessor, "40fbefc7-2e9d-4d25-a185-0cfd8574bac6", "Política de reducción de rendimiento", 1, "Single: baja de a un paso (menos oscilación)"},
		{subProcessor, "06cadf0e-64ed-448a-8927-ce7bf90eb35d", "Umbral de aumento", 30, "Reacciona antes a la carga"},
		{subProcessor, "12a0ab44-fe28-4fa9-b3bd-4b64f44960a6", "Umbral de reducción", 10, "Baja solo con poca carga"},
		{subProcessor, "984cf492-3bed-4488-a8f9-4286c97bf5aa", "Tiempo de aumento", 1, "Un intervalo de chequeo"},
		{subProcessor, "d8edeb9b-95cf-4f95-a73c-b061973693c8", "Tiempo de reducción", 3, "Espera tres intervalos antes de bajar"},
		{subProcessor, "4e4450b3-6179-4e91-b8f1-5bb9938f81a1", "Duty cycling del procesador", 0, "Desactivado"},
		{subProcessor, "5d76a2ca-e8c0-402f-a133-2158492d58ad", "Desactivar reposo del procesador", 0, "No: se conservan los C-states (temperatura y consumo en reposo)"},
		{subProcessor, "94d3a615-a899-4ac5-ae2b-e4d8f634367f", "Política de refrigeración", 1, "Activa: ventiladores antes que bajar frecuencia"},
		{subUSB, "48e6b7a6-50f5-4782-a5d4-53bb8f07e226", "Suspensión selectiva de USB", 0, "Desactivada: mouse y teclado no se duermen"},
		{subUSB, "d4e98f31-5ffe-4ce1-be31-1b38b384c009", "Administración de energía de enlace USB 3", 0, "Desactivada"},
		{subPCIe, "ee12f906-d277-404b-b6da-e5fa1a576df5", "Administración de energía del estado de vínculo PCIe (ASPM)", 0, "Desactivada: sin latencia al despertar el enlace"},
		{subWireless, "12bbebe6-58d6-4636-95bb-3217ef867c1a", "Modo de ahorro de energía del adaptador inalámbrico", 0, "Máximo rendimiento"},
		{subDisk, "6738e2c4-e8a5-4a42-b16a-e040e769756e", "Apagar el disco tras", 0, "Nunca"},
	}
	// Estacionamiento de núcleos: se desactiva solo en CPUs de núcleos homogéneos. En híbridas
	// (P/E) y en Ryzen X3D de dos CCD el estacionamiento lo usa el planificador para elegir los
	// núcleos correctos, así que se respeta.
	if !cpuNeedsParking() {
		v = append(v, planValue{subProcessor, setCoreParkMin, "Mínimo de núcleos sin estacionar", 100, "Todos los núcleos disponibles (CPU de núcleos homogéneos)"})
	}
	return v
}

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
	name := systemInfoCPUName()
	if x3dDualCCD.MatchString(name) {
		return true
	}
	return cpuIsHybrid()
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

// ensureFPSPlan crea (si hace falta) y configura el plan propio. Devuelve los ajustes que el
// equipo no soporta (se omiten sin fallar).
func ensureFPSPlan() (skipped []string, err error) {
	if !schemeExists(fpsPlanGUID) {
		src := ""
		for _, c := range []string{schemeBalanced, schemeHighPerf} {
			if schemeExists(c) {
				src = c
				break
			}
		}
		if src == "" { // equipos donde otra herramienta borró los planes estándar
			src, err = activeScheme()
			if err != nil {
				return nil, err
			}
		}
		if out, err := powercfg("/duplicatescheme", src, fpsPlanGUID); err != nil {
			return nil, fmt.Errorf("no se pudo crear el plan: %s", strings.TrimSpace(out))
		}
		invalidateSchemes()
		powercfg("/changename", fpsPlanGUID, fpsPlanName,
			"Rendimiento alto y estable para juegos. Creado por FPS Booster Pro; se borra al revertir.")
	}
	for _, v := range fpsPlanValues() {
		if !validGUID(v.Sub) || !validGUID(v.Setting) {
			return nil, errors.New("GUID inválido en la definición del plan")
		}
		if _, err := powercfg("/setacvalueindex", fpsPlanGUID, v.Sub, v.Setting, strconv.FormatUint(uint64(v.AC), 10)); err != nil {
			skipped = append(skipped, v.Name) // el ajuste no existe en este equipo
		}
	}
	return skipped, nil
}

// readACValue lee el valor AC de un ajuste del plan (para verificar).
func readACValue(scheme, sub, setting string) (uint32, bool) {
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
	v, err := strconv.ParseUint(all[len(all)-2][1], 16, 32)
	return uint32(v), err == nil
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
	if t.PlanID == "fps" {
		return fpsPlanGUID
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
