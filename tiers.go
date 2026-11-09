package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Niveles de acceso. «Propietario» y «Pro Tier» lo tienen todo; «Free Tier» solo los ajustes básicos
// (seguros y fáciles de entender). El Propietario puede mirar la app «como» Free o Pro para ver
// lo que ve cada nivel. Las restricciones también se aplican en el motor, no solo en la pantalla.

const (
	tierFree  = "Free Tier"
	tierPro   = "Pro Tier"
	tierOwner = "Propietario"
)

var errTier = errors.New("esta función es de Pro Tier (el upgrade llega pronto)")

// freeTweaks: lo que incluye Free Tier.
var freeTweaks = map[string]bool{
	"game_mode": true, "game_dvr": true, "mouse_raw_curve": true, "transparency": true, "menu_delay": true,
	"visual_fx": true, "kbd_latency": true, "explorer_startup": true,
	"startup_sound_off": true, "search_local": true, "content_delivery": true, "ads": true,
	"news_interests": true, "flush_dns": true,
}

// tweakAllowed: tier "" = sin sesión iniciada (no se permite nada).
func tweakAllowed(tier, id string) bool {
	if tier != "" && isFixID(id) { // los arreglos de Windows están en todos los niveles
		return true
	}
	switch tier {
	case tierPro, tierOwner:
		return true
	case tierFree:
		return freeTweaks[id]
	}
	return false
}

func fullAccess(tier string) bool { return tier == tierPro || tier == tierOwner }

func validTier(t string) bool { return t == tierFree || t == tierPro || t == tierOwner }

func loadViewAs() string {
	b, err := os.ReadFile(filepath.Join(discordDir(), "viewas.json"))
	if err != nil {
		return ""
	}
	var v struct{ Tier string }
	if json.Unmarshal(b, &v) != nil || !validTier(v.Tier) {
		return ""
	}
	return v.Tier
}

func saveViewAs(t string) {
	b, _ := json.Marshal(struct{ Tier string }{t})
	_ = os.WriteFile(filepath.Join(discordDir(), "viewas.json"), b, 0o600)
}

func isFixID(id string) bool {
	_, ok := fixSpecs[id] // mapa estático: no reconstruye el catálogo en cada comprobación
	return ok
}
