package main

import (
	"os"
	"testing"
)

func TestFPSPlanDefinition(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range fpsPlanValues() {
		if !validGUID(v.Sub) || !validGUID(v.Setting) {
			t.Errorf("GUID inválido en %s", v.Name)
		}
		if seen[v.Setting] {
			t.Errorf("ajuste repetido: %s", v.Name)
		}
		seen[v.Setting] = true
	}
	if !validGUID(fpsPlanGUID) {
		t.Fatal("GUID del plan inválido")
	}
	if err := runAllowedPowercfg([]string{"/delete", "x"}); err == nil {
		t.Fatal("aceptó un comando de powercfg fuera de la lista blanca")
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
func TestFPSPlanRoundTrip(t *testing.T) {
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
	if err := e.Apply("fps_power_plan"); err != nil {
		t.Fatal(err)
	}
	if g, _ := activeScheme(); g != fpsPlanGUID {
		t.Fatalf("plan activo %s", g)
	}
	checked := 0
	for _, v := range fpsPlanValues() {
		got, ok := readACValue(fpsPlanGUID, v.Sub, v.Setting)
		if !ok {
			t.Logf("no existe en este equipo: %s", v.Name)
			continue
		}
		checked++
		if got != v.AC {
			t.Errorf("%s = %d, esperado %d", v.Name, got, v.AC)
		}
	}
	t.Logf("%d ajustes verificados", checked)
	if err := e.Revert("fps_power_plan"); err != nil {
		t.Fatal(err)
	}
	if g, _ := activeScheme(); g != prev {
		t.Fatalf("no volvió al plan anterior: %s vs %s", g, prev)
	}
	invalidateSchemes()
	if schemeExists(fpsPlanGUID) {
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
