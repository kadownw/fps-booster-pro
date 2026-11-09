package main

import (
	"errors"
	"fmt"
	"strings"
)

// Perfiles: conjuntos de ajustes que se aplican de una vez. Cada uno pasa por el mismo motor
// (backup, verificación, rollback); lo que no aplica a este equipo o necesita admin se omite.

type Profile struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tweaks      []string `json:"tweaks"`
	NeedsAdmin  bool     `json:"needsAdmin"` // algún ajuste del perfil requiere admin
}

var profileDefs = []Profile{
	{
		ID: "gaming", Title: "Gaming",
		Description: "Plan de energía de rendimiento estable, perfil NVIDIA de baja latencia, Modo Juego, sin grabación en segundo plano, mouse y teclado, GPU y red.",
		Tweaks: []string{"fps_power_plan", "nvidia_profile", "game_mode", "game_dvr", "fullscreen_opt", "mouse_accel", "kbd_latency", "sticky_keys",
			"bg_apps", "hags", "net_throttle", "games_priority", "mmcss_extra", "foreground_boost", "power_throttle", "nv_pstate", "amd_ulps",
			"usb_recovery", "delivery_opt", "update_pause"},
	},
	{
		ID: "user", Title: "Rápido y sin admin",
		Description: "Solo ajustes de tu usuario: no pide permisos de administrador y no toca el sistema.",
		Tweaks: []string{"game_mode", "game_dvr", "fullscreen_opt", "mouse_accel", "kbd_latency", "sticky_keys", "transparency", "visual_fx",
			"menu_delay", "explorer_startup", "bg_apps", "search_web"},
	},
	{
		ID: "privacy", Title: "Privacidad y segundo plano",
		Description: "Menos telemetría, publicidad y tareas en segundo plano. No afecta tus programas.",
		Tweaks: []string{"content_delivery", "preinstalled_apps", "ads", "activity_history", "news_interests", "telemetry", "telemetry_tasks",
			"error_reporting", "error_tasks", "settings_ads", "find_my_device", "start_recommended", "meet_now", "copilot", "ai_recall"},
	},
}

func profiles() []Profile {
	out := make([]Profile, len(profileDefs))
	for i, p := range profileDefs {
		p.NeedsAdmin = false
		for _, id := range p.Tweaks {
			if t, ok := findTweak(id); ok && t.Admin {
				p.NeedsAdmin = true
			}
		}
		out[i] = p
	}
	return out
}

type ProfileResult struct {
	Applied []string `json:"applied"`
	Skipped []string `json:"skipped"` // motivo legible
	Failed  []string `json:"failed"`
}

// ApplyProfile aplica cada ajuste del perfil de a uno. Un fallo no detiene a los demás: el
// motor ya revirtió lo que ese ajuste alcanzó a cambiar.
func (e *Engine) ApplyProfile(id string) (ProfileResult, error) {
	var res ProfileResult
	var prof *Profile
	for i := range profileDefs {
		if profileDefs[i].ID == id {
			prof = &profileDefs[i]
		}
	}
	if prof == nil {
		return res, fmt.Errorf("perfil desconocido: %s", id)
	}
	for _, tid := range prof.Tweaks {
		t, ok := findTweak(tid)
		if !ok {
			return res, fmt.Errorf("el perfil %s referencia un ajuste inexistente: %s", id, tid)
		}
		if !supported(t) {
			res.Skipped = append(res.Skipped, t.Title+" (no aplica a tu equipo)")
			continue
		}
		e.mu.Lock()
		active := e.isActive(t)
		e.mu.Unlock()
		if active {
			res.Skipped = append(res.Skipped, t.Title+" (ya estaba activo)")
			continue
		}
		err := e.Apply(tid)
		switch {
		case err == nil:
			res.Applied = append(res.Applied, t.Title)
		case errors.Is(err, errNeedAdmin):
			res.Skipped = append(res.Skipped, t.Title+" (requiere administrador)")
		default:
			res.Failed = append(res.Failed, t.Title+": "+strings.TrimSpace(err.Error()))
		}
	}
	return res, nil
}

// ---- punto de restauración ----

// CreateRestorePoint crea un punto de restauración del sistema. Windows limita la frecuencia
// (por defecto uno cada 24 h) y exige que la Protección del sistema esté activa.
func CreateRestorePoint() error {
	if !isAdmin() {
		return errNeedAdmin
	}
	out, err := runPS(`Checkpoint-Computer -Description 'FPS Booster Pro' -RestorePointType MODIFY_SETTINGS -ErrorAction Stop`)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("no se pudo crear el punto de restauración: %s", msg)
	}
	return nil
}
