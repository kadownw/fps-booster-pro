package main

import "testing"

func TestFixChecksReadOnly(t *testing.T) {
	e := &Engine{dir: t.TempDir(), st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: map[string]string{}}
	fixes := e.ListFixes()
	if len(fixes) != 9 {
		t.Fatalf("se esperaban 9 arreglos, hay %d", len(fixes))
	}
	for _, f := range fixes {
		r := e.FixCheck(f.ID)
		if r.Error != "" {
			t.Errorf("%s: %s", f.ID, r.Error)
			continue
		}
		t.Logf("%-20s problemas=%d healthy=%q", f.ID, len(r.Problems), r.Healthy)
		for _, p := range r.Problems {
			t.Logf("      · %s", p)
		}
		if f.Icon == "" || f.Title == "" || f.Description == "" {
			t.Errorf("%s: faltan datos de la tarjeta", f.ID)
		}
	}
}

func TestEveryFixHasSpecAndTier(t *testing.T) {
	for _, tw := range catalog() {
		if !tw.Fix {
			continue
		}
		if _, ok := fixSpecs[tw.ID]; !ok {
			t.Errorf("%s: no tiene comprobación", tw.ID)
		}
		if !tweakAllowed(tierFree, tw.ID) {
			t.Errorf("%s: debería estar disponible para Free", tw.ID)
		}
		if tweakAllowed("", tw.ID) {
			t.Errorf("%s: sin sesión no se permite nada", tw.ID)
		}
	}
	for id := range fixSpecs {
		if tw, ok := findTweak(id); !ok || !tw.Fix {
			t.Errorf("%s: spec sin arreglo en el catálogo", id)
		}
	}
}

func TestDeleteChangeSemantics(t *testing.T) {
	c := policyGone(`SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate`, "NoAutoUpdate")
	if desired(c).Exists || !satisfied(c, RegState{}) || satisfied(c, RegState{Exists: true, Kind: "dword", D: 1}) {
		t.Fatal("un cambio Delete debe cumplirse solo cuando el valor NO existe")
	}
}

func TestIfCurrentOnlyTouchesDisabledServices(t *testing.T) {
	// Servicios que están bien (no deshabilitados) no deben entrar en la lista de cambios.
	tw := Tweak{ID: "x", Changes: svcFixes(startManual, "Spooler", "ServicioQueNoExiste_zz")}
	for _, c := range expand(tw) {
		if st, _ := readState(c); st.D == startDisabled {
			continue
		}
		t.Errorf("%s: se incluyó aunque no estaba deshabilitado", c.Path)
	}
}
