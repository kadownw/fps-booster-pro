package main

import (
	"os"
	"strings"
	"testing"
)

func TestSouthPlanTable(t *testing.T) {
	if len(southPlanTable) < 100 {
		t.Fatalf("la tabla del plan es demasiado corta: %d", len(southPlanTable))
	}
	seen := map[string]bool{}
	for _, s := range southPlanTable {
		if !validGUID(s.Sub) || !validGUID(s.Setting) {
			t.Errorf("GUID inválido en %q", s.Name)
		}
		if seen[s.Sub+s.Setting] {
			t.Errorf("ajuste repetido: %s", s.Name)
		}
		seen[s.Sub+s.Setting] = true
	}
	if !validGUID(southPlanGUID) {
		t.Fatal("GUID del plan inválido")
	}
	if southPlanName != "South's Core power plan" {
		t.Fatalf("nombre del plan: %q", southPlanName)
	}
	// el procesador debe quedar siempre disponible con corriente
	found := false
	for _, s := range southPlanTable {
		if s.Name == "Minimum processor state" && s.AC == 100 {
			found = true
		}
	}
	if !found {
		t.Error("el plan no fija el estado mínimo del procesador al 100 % con corriente")
	}
	if err := runAllowedPowercfg([]string{"/delete", "x"}); err == nil {
		t.Fatal("aceptó un comando de powercfg fuera de la lista blanca")
	}
}

func TestSouthPlanFilters(t *testing.T) {
	all := southSettings(false)
	for _, s := range all {
		if s.Group == "Battery" || s.Group == "Power buttons and lid" {
			t.Errorf("no debería tocar el grupo %q (%s)", s.Group, s.Name)
		}
	}
	noPark := southSettings(true)
	for _, s := range noPark {
		if strings.Contains(strings.ToLower(s.Name), "core parking") {
			t.Errorf("en CPUs híbridas/X3D no debe tocar estacionamiento: %s", s.Name)
		}
	}
	if len(noPark) >= len(all) {
		t.Error("el filtro de estacionamiento no quitó nada")
	}
	// con batería solo se aplica hardware
	for _, s := range all {
		if dcGroups[s.Group] && s.Group == "Sleep" {
			t.Error("Sleep no debe aplicarse con batería")
		}
	}
	for _, g := range []string{"Sleep", "Display"} {
		if dcGroups[g] {
			t.Errorf("%s no debe estar entre los grupos con valor de batería", g)
		}
	}
}

func TestReadACValueParsesAnyLanguage(t *testing.T) {
	// Formato de `powercfg /qh` en inglés y español: los dos últimos índices son AC y DC.
	en := "Power Setting GUID: x (USB)\n  Possible Setting Index: 000\n  Current AC Power Setting Index: 0x00000000\n  Current DC Power Setting Index: 0x00000001\n"
	es := "GUID de configuración: x\n  Índice de configuración de corriente alterna actual: 0x00000064\n  Índice de configuración de corriente continua actual: 0x00000005\n"
	for in, want := range map[string]uint32{en: 0, es: 100} {
		all := hexIdxRe.FindAllStringSubmatch(in, -1)
		if len(all) < 2 {
			t.Fatalf("sin índices en %q", in)
		}
		if got := all[len(all)-2][1]; (got == "00000000") != (want == 0) {
			t.Fatalf("AC mal leído: %s", got)
		}
	}
}

// Crea el plan, lo activa, verifica valores, revierte y comprueba que se borró y volvió el anterior.
func TestSouthPlanRoundTrip(t *testing.T) {
	if schemeExists(southPlanGUID) {
		// La prueba crea el plan y verifica que se borre al revertir; si el usuario ya lo tiene instalado
		// (p. ej. como su plan activo) no se puede distinguir, así que no se toca.
		t.Skip("el plan South's Core ya existe en este equipo: no se prueba para no borrar el plan del usuario")
	}
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() {
		t.Skip("define FBP_ADMIN_TESTS=1 y ejecuta como administrador")
	}
	e, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	prev, err := activeScheme()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply("south_power_plan"); err != nil {
		t.Fatal(err)
	}
	if g, _ := activeScheme(); g != southPlanGUID {
		t.Fatalf("plan activo %s", g)
	}
	checked, mismatched := 0, 0
	for _, v := range southSettings(cpuNeedsParking()) {
		got, ok := readACValue(southPlanGUID, v.Sub, v.Setting)
		if !ok {
			continue // el ajuste no existe en este equipo
		}
		checked++
		if got != v.AC {
			mismatched++
			t.Errorf("%s = %d, esperado %d", v.Name, got, v.AC)
		}
	}
	t.Logf("%d ajustes verificados, %d distintos", checked, mismatched)
	if checked < 60 {
		t.Errorf("se verificaron muy pocos ajustes: %d", checked)
	}
	if err := e.Revert("south_power_plan"); err != nil {
		t.Fatal(err)
	}
	if g, _ := activeScheme(); g != prev {
		t.Fatalf("no volvió al plan anterior: %s vs %s", g, prev)
	}
	invalidateSchemes()
	if schemeExists(southPlanGUID) {
		t.Fatal("el plan creado no se borró al revertir")
	}
}

func TestHibernateRoundTrip(t *testing.T) {
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() {
		t.Skip("define FBP_ADMIN_TESTS=1 y ejecuta como administrador")
	}
	tw, _ := findTweak("hibernate_off")
	before, _ := readState(tw.Cmd.State)
	e, _ := NewEngine()
	if err := e.Apply("hibernate_off"); err != nil {
		t.Fatal(err)
	}
	if st, _ := readState(tw.Cmd.State); !sameState(st, desired(tw.Cmd.State)) {
		t.Fatalf("no quedó desactivada: %+v", st)
	}
	if err := e.Revert("hibernate_off"); err != nil {
		t.Fatal(err)
	}
	if after, _ := readState(tw.Cmd.State); after.D != before.D {
		t.Fatalf("no volvió al estado original: %+v vs %+v", after, before)
	}
}
