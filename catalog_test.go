package main

import "testing"

func TestCatalogIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, tw := range catalog() {
		if seen[tw.ID] {
			t.Errorf("ID duplicado: %s", tw.ID)
		}
		seen[tw.ID] = true
		if tw.Title == "" || tw.Description == "" || tw.Category == "" {
			t.Errorf("%s: faltan textos", tw.ID)
		}
		if !tw.Power && len(tw.Changes) == 0 {
			t.Errorf("%s: sin cambios", tw.ID)
		}
		usesHKLM := false
		for _, c := range tw.Changes {
			if c.Root != "HKCU" && c.Root != "HKLM" {
				t.Errorf("%s: raíz no permitida %s", tw.ID, c.Root)
			}
			if c.Root == "HKLM" {
				usesHKLM = true
			}
			if c.Kind != "dword" && c.Kind != "sz" {
				t.Errorf("%s: tipo desconocido %s", tw.ID, c.Kind)
			}
		}
		if usesHKLM && !tw.Admin {
			t.Errorf("%s: escribe en HKLM pero no marca Admin", tw.ID)
		}
		if tw.Risk != "safe" && tw.Risk != "moderate" {
			t.Errorf("%s: riesgo inválido %q", tw.ID, tw.Risk)
		}
	}
}

// Todos los ajustes que no necesitan admin se aplican y revierten de verdad en HKCU.
func TestRoundTripAllUserTweaks(t *testing.T) {
	e, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	for _, tw := range catalog() {
		if tw.Admin || tw.Power || !supported(tw) {
			continue
		}
		before := make([]RegState, len(expand(tw)))
		for i, c := range expand(tw) {
			if before[i], err = readState(c); err != nil {
				t.Fatalf("%s: %v", tw.ID, err)
			}
		}
		if err := e.Apply(tw.ID); err != nil {
			t.Fatalf("apply %s: %v", tw.ID, err)
		}
		if err := e.Revert(tw.ID); err != nil {
			t.Fatalf("revert %s: %v", tw.ID, err)
		}
		for i, c := range expand(tw) {
			if st, _ := readState(c); !sameState(st, before[i]) {
				t.Fatalf("%s: %s no volvió al original", tw.ID, c.Name)
			}
		}
	}
}
