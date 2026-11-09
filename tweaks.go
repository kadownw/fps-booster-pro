package main

import (
	"strings"
	"sync"
	"time"
)

// Catálogo de tweaks. Todos son reversibles: el motor guarda el valor original
// antes de tocar nada y lo restaura al revertir. Ninguno promete FPS.

const (
	schemeHighPerf = "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"
)

type RegChange struct {
	Root string `json:"root"` // "HKCU" | "HKLM"
	Path string `json:"path"`
	Name string `json:"name"`
	Kind string `json:"kind"` // "dword" | "sz"
	D    uint32 `json:"d,omitempty"`
	S    string `json:"s,omitempty"`
	// MustExist: si la clave no existe en este equipo se omite el cambio (no se crean claves nuevas;
	// se usa en servicios y loggers que pueden no estar instalados).
	MustExist bool `json:"mustExist,omitempty"`
	// RAMKB: el valor es la RAM total en KB (se calcula al aplicar).
	RAMKB bool `json:"ramKB,omitempty"`
	// Hex: valor binario (REG_BINARY) en hexadecimal; con Kind "bin".
	Hex string `json:"hex,omitempty"`
	// KV: el valor es una lista "clave=valor;" y solo se agrega/reemplaza esta clave (se conservan las demás).
	KV string `json:"kv,omitempty"`
	// IfExists: se omite si el valor no existe en este equipo (drivers que no lo soportan).
	IfExists bool `json:"ifExists,omitempty"`
	// Date: valor de fecha dinámico en UTC: "now" o "+35d" (se calcula al aplicar).
	Date string `json:"date,omitempty"`
	// Delete: el valor debe NO existir (se borra; el backup guarda el original para poder devolverlo).
	Delete bool `json:"delete,omitempty"`
	// IfCurrent: el cambio solo se hace si el valor DWORD actual es exactamente este (p. ej. servicios
	// deshabilitados que un arreglo vuelve a poner en manual; no toca los que ya están bien).
	IfCurrent *uint32 `json:"ifCurrent,omitempty"`
	// Optional: si Windows protege este valor y rechaza la escritura (acceso denegado), se omite y el resto
	// del ajuste sigue; el diagnóstico lo cuenta como «omitido» y no como pendiente.
	Optional bool `json:"optional,omitempty"`
}

type Tweak struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Admin       bool        `json:"admin"`              // requiere permisos de administrador
	Reboot      bool        `json:"reboot"`             // requiere reiniciar para tener efecto
	Power       bool        `json:"power"`              // cambia el plan de energía (powercfg)
	Vendor      string      `json:"vendor,omitempty"`   // "nvidia" | "amd" | "intel": se aplica a cada GPU de ese fabricante
	Risk        string      `json:"risk"`               // "safe" | "moderate"
	MinBuild    int         `json:"minBuild,omitempty"` // build mínima de Windows (0 = cualquiera)
	MaxBuild    int         `json:"maxBuild,omitempty"` // build máxima de Windows (0 = sin límite)
	Tasks       []string    `json:"tasks,omitempty"`    // tareas programadas que se deshabilitan
	Changes     []RegChange `json:"changes"`
	// PlanID: con Power=true, "south" = plan propio "South's Core power plan".
	PlanID string `json:"planId,omitempty"`
	// Nvidia: ajustes del perfil global del driver NVIDIA (NvAPI DRS).
	Nvidia []NvSetting `json:"nvidia,omitempty"`
	// Cmd: ajustes que Windows solo permite cambiar con un comando (p. ej. hibernación).
	Cmd *CmdSpec `json:"-"`
	// PS: ajuste cuyo estado solo se consulta/cambia con un comando de PowerShell fijo (definido aquí, nunca armado con texto del usuario).
	PS *PSSpec `json:"-"`
	// Once: acción que se ejecuta una vez y no deja nada que revertir (botón «Aplicar», sin interruptor).
	Once bool `json:"once,omitempty"`
	// Fix: arreglo de un problema de Windows (pestaña Fixes); no aparece en Optimizar. Icon: icono de su tarjeta.
	Fix  bool   `json:"fix,omitempty"`
	Icon string `json:"icon,omitempty"`
}

// PSSpec: estado = salida de State (texto); Want = valor que busca el ajuste; Apply/Revert = comandos fijos.
type PSSpec struct {
	State, Want   string
	Apply, Revert string
}

// CmdSpec: comando permitido (solo powercfg) con su estado observable en el registro.
type CmdSpec struct {
	Apply  []string  // argumentos de powercfg para aplicar
	Revert []string  // argumentos de powercfg para volver al estado original
	State  RegChange // valor que refleja el estado; Kind/D = estado deseado
}

const (
	gpuPlaceholder  = "{GPU}"
	nicPlaceholder  = "{NIC}"  // cada adaptador de red físico (clase Net)
	dispPlaceholder = "{DISP}" // cada dispositivo de video PCI (para MSI)
)

// expand reemplaza {GPU} por la clave de cada adaptador del fabricante, calcula valores
// dinámicos y omite los cambios marcados MustExist cuya clave no existe en este equipo.
// expandMemo: durante List() (que pide expand varias veces por ajuste) se reutiliza el resultado. Fuera de List
// siempre se lee el sistema de nuevo, porque aplicar o revertir cambia lo que expand devuelve.
var expandMemo struct {
	sync.Mutex
	m map[string][]RegChange
}

func beginExpandMemo() {
	expandMemo.Lock()
	expandMemo.m = map[string][]RegChange{}
	expandMemo.Unlock()
}

func endExpandMemo() {
	expandMemo.Lock()
	expandMemo.m = nil
	expandMemo.Unlock()
}

func expand(t Tweak) []RegChange {
	expandMemo.Lock()
	if expandMemo.m != nil {
		if v, ok := expandMemo.m[t.ID]; ok {
			expandMemo.Unlock()
			return v
		}
	}
	expandMemo.Unlock()
	out := expandUncached(t)
	expandMemo.Lock()
	if expandMemo.m != nil {
		expandMemo.m[t.ID] = out
	}
	expandMemo.Unlock()
	return out
}

func expandUncached(t Tweak) []RegChange {
	var base []RegChange
	if t.Vendor == "" {
		for _, c := range t.Changes {
			if strings.Contains(c.Path, nicPlaceholder) { // un cambio por adaptador de red físico
				for _, sub := range nicAdapters() {
					cc := c
					cc.Path = strings.Replace(c.Path, nicPlaceholder, sub, 1)
					base = append(base, cc)
				}
				continue
			}
			if strings.Contains(c.Path, dispPlaceholder) { // un cambio por dispositivo de video PCI
				for _, inst := range displayDevices() {
					cc := c
					cc.Path = strings.Replace(c.Path, dispPlaceholder, inst, 1)
					base = append(base, cc)
				}
				continue
			}
			base = append(base, c)
		}
	} else {
		for _, a := range gpuAdapters() {
			if a.Vendor != t.Vendor {
				continue
			}
			for _, c := range t.Changes {
				c.Path = strings.Replace(c.Path, gpuPlaceholder, a.Sub, 1)
				base = append(base, c)
			}
		}
	}
	var out []RegChange
	now := time.Now()
	for _, c := range base {
		if c.MustExist && !keyExists(c) {
			continue
		}
		if c.IfExists {
			if st, err := readState(c); err != nil || !st.Exists {
				continue
			}
		}
		if c.IfCurrent != nil {
			if st, err := readState(c); err != nil || !st.Exists || st.Kind != "dword" || st.D != *c.IfCurrent {
				continue
			}
		}
		if c.Date != "" {
			c.S = dateFor(c.Date, now)
		}
		if c.RAMKB {
			total, _, _ := memory()
			kb := total / 1024
			if kb > 0xffffffff {
				kb = 0xffffffff
			}
			c.D = uint32(kb)
		}
		out = append(out, c)
	}
	return out
}

func dw(root, path, name string, v uint32) RegChange {
	return RegChange{Root: root, Path: path, Name: name, Kind: "dword", D: v}
}
func sz(root, path, name, v string) RegChange {
	return RegChange{Root: root, Path: path, Name: name, Kind: "sz", S: v}
}

const (
	gameBar    = `Software\Microsoft\GameBar`
	gameCfg    = `System\GameConfigStore`
	gameDVR    = `Software\Microsoft\Windows\CurrentVersion\GameDVR`
	personal   = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	mouseKey   = `Control Panel\Mouse`
	desktopKey = `Control Panel\Desktop`
	bgApps     = `Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`
	gfxDrivers = `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`
	sysProfile = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
	prioCtl    = `SYSTEM\CurrentControlSet\Control\PriorityControl`
)

// Los ajustes de Windows del catálogo coinciden con prácticas conocidas (guías de latencia,
// documentación de MMCSS); los valores se reimplementaron de forma independiente.
var moderate = map[string]bool{"hags": true, "games_priority": true, "foreground_boost": true,
	"net_throttle": true, "bg_apps": true, "mmcss_extra": true,
	"power_throttle": true, "sticky_keys": true, "telemetry": true, "error_reporting": true,
	"copilot": true, "ai_recall": true, "svc_xbox_manual": true, "fast_startup": true, "svchost_split": true,
	"location_off": true, "cortana": true, "south_power_plan": true, "nvidia_profile": true, "hibernate_off": true,
	"mpo_off": true, "flip_immediate": true, "nic_power_off": true, "updates_pause": true,
	"memory_compression_off": true, "timer_resolution": true, "nv_hdcp_off": true, "amd_graphics_perf": true, "gpu_msi_mode": true, "nv_driver_prefs": true, "svc_max_performance": true,
	"fix_valorant_hvci": true, "fix_defender": true, "fix_store_update": true, "ntfs_lastaccess_off": true,
	"maintenance_off": true, "prevent_device_encryption": true, "remove_bloat_apps": true}

func catalog() []Tweak {
	list := append(rawCatalog(), extraCatalog()...)
	list = append(list, servicesCatalog()...)
	list = append(list, paragonCatalog()...)
	list = append(list, ultimateCatalog()...)
	list = append(list, setupCatalog()...)
	list = append(list, maxPerfCatalog()...)
	list = append(list, cleanCatalog()...)
	list = append(list, fixesCatalog()...)
	dedupeMaxPerf(list)
	for i := range list {
		list[i].Risk = "safe"
		if moderate[list[i].ID] {
			list[i].Risk = "moderate"
		}
	}
	return list
}

const (
	gpuClass = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\` + gpuPlaceholder
	kbdKey   = `Control Panel\Keyboard`
)

func rawCatalog() []Tweak {
	return []Tweak{
		{
			ID: "game_mode", Title: "Modo Juego de Windows", Category: "Juegos",
			Description: "Windows prioriza el juego en primer plano y evita instalar actualizaciones o mostrar notificaciones mientras jugás.",
			Changes:     []RegChange{dw("HKCU", gameBar, "AutoGameModeEnabled", 1)},
		},
		{
			ID: "game_dvr", Title: "Desactivar grabación en segundo plano (Game DVR)", Category: "Juegos",
			Description: "Apaga la captura continua de la Xbox Game Bar, que graba en segundo plano y usa GPU y disco. La Game Bar sigue funcionando manualmente.",
			Changes: []RegChange{
				dw("HKCU", gameCfg, "GameDVR_Enabled", 0),
				dw("HKCU", gameDVR, "AppCaptureEnabled", 0),
			},
		},
		{
			ID: "fullscreen_opt", Title: "Desactivar optimizaciones de pantalla completa", Category: "Juegos",
			Description: "Hace que los juegos usen pantalla completa exclusiva real en vez de la ventana optimizada de Windows. Algunos juegos mejoran, otros no: probalo.",
			Changes: []RegChange{
				dw("HKCU", gameCfg, "GameDVR_FSEBehaviorMode", 2),
				dw("HKCU", gameCfg, "GameDVR_HonorUserFSEBehaviorMode", 1),
				dw("HKCU", gameCfg, "GameDVR_DXGIHonorFSEWindowsCompatible", 1),
				dw("HKCU", gameCfg, "GameDVR_FSEBehavior", 2),
			},
		},
		{
			ID: "transparency", Title: "Desactivar efectos de transparencia", Category: "Visual",
			Description: "Quita la transparencia de la barra de tareas y los menús. Libera algo de GPU en equipos con gráficos integrados. Solo cambia la apariencia.",
			Changes:     []RegChange{dw("HKCU", personal, "EnableTransparency", 0)},
		},
		{
			ID: "menu_delay", Title: "Menús sin demora", Category: "Visual",
			Description: "Elimina la demora al abrir los menús de Windows. Solo cambia la sensación de velocidad de la interfaz.",
			Changes:     []RegChange{sz("HKCU", desktopKey, "MenuShowDelay", "0")},
		},
		{
			ID: "bg_apps", Title: "Bloquear apps en segundo plano", Category: "Sistema", Admin: true,
			Description: "Impide que las apps de la Tienda corran en segundo plano, con el ajuste de tu usuario y la política del sistema para todos los usuarios. Libera algo de RAM y CPU. Algunas apps dejarán de notificar cuando no las uses.",
			Changes: []RegChange{
				dw("HKCU", bgApps, "GlobalUserDisabled", 1),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Search`, "BackgroundAppGlobalToggle", 0),
				dw("HKLM", `SOFTWARE\Policies\Microsoft\Windows\AppPrivacy`, "LetAppsRunInBackground", 2),
			},
		},
		{
			ID: "hags", Title: "Programación de GPU acelerada por hardware", Category: "GPU", Admin: true, Reboot: true,
			Description: "Activa HAGS. Requiere GPU y driver compatibles; en algunos equipos mejora la latencia y en otros no cambia nada.",
			Changes:     []RegChange{dw("HKLM", gfxDrivers, "HwSchMode", 2)},
		},
		{
			ID: "net_throttle", Title: "Quitar limitación de red multimedia", Category: "Red", Admin: true, Reboot: true,
			Description: "Desactiva el límite de paquetes que Windows aplica mientras se reproduce multimedia y prioriza las tareas en primer plano.",
			Changes: []RegChange{
				dw("HKLM", sysProfile, "NetworkThrottlingIndex", 0xffffffff),
				dw("HKLM", sysProfile, "SystemResponsiveness", 0),
			},
		},
		{
			ID: "games_priority", Title: "Prioridad alta para tareas de juegos", Category: "Sistema", Admin: true,
			Description: "Sube la prioridad de CPU y GPU que el planificador multimedia da a los juegos.",
			Changes: []RegChange{
				dw("HKLM", sysProfile+`\Tasks\Games`, "GPU Priority", 8),
				dw("HKLM", sysProfile+`\Tasks\Games`, "Priority", 6),
				sz("HKLM", sysProfile+`\Tasks\Games`, "Scheduling Category", "High"),
				sz("HKLM", sysProfile+`\Tasks\Games`, "SFIO Priority", "High"),
			},
		},
		{
			ID: "mmcss_extra", Title: "Planificador multimedia siempre activo", Category: "Sistema", Admin: true,
			Description: "Mantiene activo el planificador multimedia (MMCSS) sin modo perezoso, para respuesta más constante en juegos y audio.",
			Changes: []RegChange{
				dw("HKLM", sysProfile, "NoLazyMode", 1),
				dw("HKLM", sysProfile, "AlwaysOn", 1),
			},
		},
		{
			ID: "kbd_latency", Title: "Teclado: repetición rápida", Category: "Entrada",
			Description: "Reduce la demora y acelera la repetición de teclas. Solo cambia la sensación del teclado.",
			Changes: []RegChange{
				sz("HKCU", kbdKey, "KeyboardDelay", "0"),
				sz("HKCU", kbdKey, "KeyboardSpeed", "31"),
			},
		},
		{
			ID: "amd_ulps", Title: "AMD: desactivar ULPS", Category: "GPU", Admin: true, Reboot: true, Vendor: "amd",
			Description: "Desactiva el ahorro de energía ultra bajo (ULPS) del driver AMD, útil si hay microcortes o instabilidad con varias GPU.",
			Changes:     []RegChange{dw("HKLM", gpuClass, "EnableULPS", 0)},
		},
		{
			ID: "foreground_boost", Title: "Priorizar la aplicación en primer plano", Category: "Sistema", Admin: true,
			Description: "Ajusta el reparto de tiempo de CPU para favorecer lo que tenés en pantalla (Win32PrioritySeparation = 38).",
			Changes:     []RegChange{dw("HKLM", prioCtl, "Win32PrioritySeparation", 38)},
		},
	}
}

func findTweak(id string) (Tweak, bool) {
	for _, t := range catalog() {
		if t.ID == id {
			return t, true
		}
	}
	return Tweak{}, false
}
