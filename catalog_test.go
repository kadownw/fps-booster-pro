package main

import (
	"strings"
	"testing"
	"time"
)

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
		if !tw.Power && len(tw.Changes) == 0 && len(tw.Tasks) == 0 && len(tw.Nvidia) == 0 && tw.Cmd == nil && !tw.Once {
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
			if strings.Contains(c.Path, `Services\`) && c.Name == "Start" {
				if !c.MustExist || c.D < 2 || c.D > 4 {
					t.Errorf("%s: servicio %s mal definido (MustExist/Start)", tw.ID, c.Path)
				}
			}
			if c.Kind != "dword" && c.Kind != "sz" {
				t.Errorf("%s: tipo desconocido %s", tw.ID, c.Kind)
			}
		}
		if len(tw.Tasks) > 0 && !tw.Admin {
			t.Errorf("%s: las tareas programadas requieren Admin", tw.ID)
		}
		for _, p := range tw.Tasks {
			if !taskPathRe.MatchString(p) {
				t.Errorf("%s: ruta de tarea inválida %q", tw.ID, p)
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

func TestSetTaskRejectsBadPaths(t *testing.T) {
	for _, p := range []string{`\x'; calc; '`, `Microsoft\x`, "", `\a\\b`} {
		if err := setTask(p, false); err == nil {
			t.Errorf("aceptó la ruta %q", p)
		}
	}
}

func TestMustExistSkipsMissingKeys(t *testing.T) {
	tw := Tweak{ID: "x", Changes: []RegChange{svc("ServicioQueNoExiste_zz", 4)}}
	if n := len(expand(tw)); n != 0 {
		t.Fatalf("debía omitir la clave inexistente, devolvió %d cambios", n)
	}
}

func TestListIsReasonablyFast(t *testing.T) {
	e, _ := NewEngine()
	start := time.Now()
	list := e.List()
	if len(list) < 40 {
		t.Fatalf("catálogo incompleto: %d", len(list))
	}
	if d := time.Since(start); d > 15*time.Second {
		t.Fatalf("List tardó %v", d)
	}
	t.Logf("%d ajustes, List en %v", len(list), time.Since(start))
}
