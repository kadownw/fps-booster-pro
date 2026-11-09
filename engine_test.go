package main

import "testing"

// Prueba real contra HKCU (sin admin): aplica, verifica y revierte, y comprueba que el valor
// original vuelve idéntico (incluido "no existía").
func TestRoundTripHKCU(t *testing.T) {
	e, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"transparency", "mouse_accel", "menu_delay", "game_dvr"} {
		tw, _ := findTweak(id)
		before := make([]RegState, len(expand(tw)))
		for i, c := range expand(tw) {
			before[i], err = readState(c)
			if err != nil {
				t.Fatalf("%s: %v", id, err)
			}
		}
		if err := e.Apply(id); err != nil {
			t.Fatalf("apply %s: %v", id, err)
		}
		for _, c := range expand(tw) {
			st, _ := readState(c)
			if !sameState(st, desired(c)) {
				t.Fatalf("%s: no quedó aplicado %s", id, c.Name)
			}
		}
		if err := e.Revert(id); err != nil {
			t.Fatalf("revert %s: %v", id, err)
		}
		for i, c := range expand(tw) {
			st, _ := readState(c)
			if !sameState(st, before[i]) {
				t.Fatalf("%s: %s no volvió al original: %+v vs %+v", id, c.Name, st, before[i])
			}
		}
	}
}

func TestAdminTweaksBlockedWithoutAdmin(t *testing.T) {
	if isAdmin() {
		t.Skip("corriendo como admin")
	}
	e, _ := NewEngine()
	if err := e.Apply("hags"); err != errNeedAdmin {
		t.Fatalf("esperaba errNeedAdmin, obtuve %v", err)
	}
}

func TestSimulationTouchesNothing(t *testing.T) {
	e, _ := NewEngine()
	e.SetSimulation(true)
	tw, _ := findTweak("transparency")
	before, _ := readState(tw.Changes[0])
	if err := e.Apply("transparency"); err != nil {
		t.Fatal(err)
	}
	after, _ := readState(tw.Changes[0])
	if !sameState(before, after) {
		t.Fatal("la simulación modificó el registro")
	}
}

func TestRejectsBadGUID(t *testing.T) {
	if err := setScheme("x; calc.exe"); err == nil {
		t.Fatal("aceptó un GUID inválido")
	}
}
