package main

import (
	"os"
	"testing"
)

func TestOnceActions(t *testing.T) {
	e, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	// una acción reversible no se puede ejecutar como acción única, y viceversa
	if _, err := e.RunOnce("game_mode"); err == nil {
		t.Fatal("RunOnce aceptó un ajuste reversible")
	}
	if err := e.Apply("flush_dns"); err == nil {
		t.Fatal("Apply aceptó una acción única")
	}
	if _, err := e.RunOnce("no_existe"); err == nil {
		t.Fatal("aceptó un id inexistente")
	}
	// en simulación no ejecuta nada
	e.SetSimulation(true)
	msg, err := e.RunOnce("flush_dns")
	if err != nil || msg == "" {
		t.Fatalf("simulación: %q %v", msg, err)
	}
	if _, has := e.runs["flush_dns"]; has {
		t.Fatal("la simulación registró una ejecución")
	}
	e.SetSimulation(false)
	// toda acción única del catálogo tiene receta
	for _, tw := range catalog() {
		if tw.Once && !tw.Admin {
			t.Errorf("%s: las acciones únicas actuales requieren admin", tw.ID)
		}
	}
}

func TestOnceFlushDNSReal(t *testing.T) {
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() {
		t.Skip("define FBP_ADMIN_TESTS=1 y ejecuta como administrador")
	}
	e, _ := NewEngine()
	msg, err := e.RunOnce("flush_dns")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(msg)
	if e.runs["flush_dns"] == "" {
		t.Fatal("no registró la ejecución")
	}
	delete(e.runs, "flush_dns") // la prueba no debe quedar como una ejecución del usuario
	e.saveRuns()
}
