package main

import "strings"

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
	Changes     []RegChange `json:"changes"`
}

const gpuPlaceholder = "{GPU}"

// expand reemplaza {GPU} por la clave de cada adaptador del fabricante. Para tweaks sin
// fabricante devuelve los cambios tal cual.
func expand(t Tweak) []RegChange {
	if t.Vendor == "" {
		return t.Changes
	}
	var out []RegChange
	for _, a := range gpuAdapters() {
		if a.Vendor != t.Vendor {
			continue
		}
		for _, c := range t.Changes {
			c.Path = strings.Replace(c.Path, gpuPlaceholder, a.Sub, 1)
			out = append(out, c)
		}
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
	"net_throttle": true, "bg_apps": true, "nv_pstate": true, "mmcss_extra": true, "power_plan": true,
	"power_throttle": true, "sticky_keys": true, "telemetry": true, "error_reporting": true,
	"copilot": true, "ai_recall": true}

func catalog() []Tweak {
	list := append(rawCatalog(), extraCatalog()...)
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
			ID: "power_plan", Title: "Plan de energía: Alto rendimiento", Category: "Energía", Power: true,
			Description: "Activa el plan de Alto rendimiento para que el procesador no baje la frecuencia de más. En notebooks aumenta consumo y temperatura.",
		},
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
			ID: "mouse_accel", Title: "Desactivar aceleración del mouse", Category: "Entrada",
			Description: "Quita la precisión mejorada del puntero para que el mouse responda de forma lineal. Útil en shooters. Cierra sesión o reiniciá para aplicar del todo.",
			Changes: []RegChange{
				sz("HKCU", mouseKey, "MouseSpeed", "0"),
				sz("HKCU", mouseKey, "MouseThreshold1", "0"),
				sz("HKCU", mouseKey, "MouseThreshold2", "0"),
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
			ID: "bg_apps", Title: "Bloquear apps en segundo plano", Category: "Sistema",
			Description: "Impide que las apps de la Tienda corran en segundo plano. Libera algo de RAM y CPU. Algunas apps dejarán de notificar cuando no las uses.",
			Changes: []RegChange{
				dw("HKCU", bgApps, "GlobalUserDisabled", 1),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Search`, "BackgroundAppGlobalToggle", 0),
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
			ID: "nv_pstate", Title: "NVIDIA: evitar cambios de estado de energía", Category: "GPU", Admin: true, Reboot: true, Vendor: "nvidia",
			Description: "Evita que la GPU NVIDIA baje de estado de energía dinámicamente. Puede estabilizar el rendimiento; aumenta consumo en reposo.",
			Changes:     []RegChange{dw("HKLM", gpuClass, "DisableDynamicPstate", 1)},
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
