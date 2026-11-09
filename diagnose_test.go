package main

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"
)

func TestParseBitLocker(t *testing.T) {
	v, err := parseBitLocker("C:|On|FullyEncrypted|100\nD:|Off|FullyDecrypted|0\n")
	if err != nil || len(v) != 2 || !v[0].encrypted() || v[1].encrypted() {
		t.Fatalf("parseo incorrecto: %+v %v", v, err)
	}
	if _, err := parseBitLocker("ERROR|Element not found."); err == nil || !strings.Contains(err.Error(), "Element not found") {
		t.Fatalf("un error de PowerShell debe propagarse, no tratarse como «0 unidades»: %v", err)
	}
	if _, err := parseBitLocker("basura"); err == nil {
		t.Fatal("una respuesta que no es de BitLocker debe fallar")
	}
}

func TestHumanErrorExplains(t *testing.T) {
	tw, _ := findTweak("game_mode")
	cases := map[string]error{
		"administrador":      errNeedAdmin,
		"denegó el acceso":   fmt.Errorf("HKLM\\x\\y: %w", syscall.ERROR_ACCESS_DENIED),
		"sobrescribiendo":    errors.New("verificación fallida en HKLM\\a\\b"),
		"No se pudo aplicar": errors.New("algo raro"),
	}
	for want, err := range cases {
		if got := humanError(tw, err); !strings.Contains(got, want) {
			t.Errorf("humanError(%v) = %q, debe contener %q", err, got, want)
		}
	}
}

// Solo lectura: todos los ajustes del catálogo deben poder explicarse en este equipo.
func TestDiagnoseEveryTweak(t *testing.T) {
	e := &Engine{dir: t.TempDir(), st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: map[string]string{}}
	valid := map[string]bool{"applied": true, "partial": true, "not_applied": true, "unavailable": true, "once": true}
	count := map[string]int{}
	for _, v := range e.List() {
		if !valid[v.Status] || len(v.Detail) < 20 {
			t.Errorf("%s: estado %q / detalle %q no sirve", v.ID, v.Status, v.Detail)
		}
		count[v.Status]++
		if v.Status == "unavailable" && !strings.Contains(v.Detail, "No aplica") {
			t.Errorf("%s: un ajuste no disponible debe explicar por qué", v.ID)
		}
		t.Logf("%-26s %-12s %s", v.ID, v.Status, v.Detail)
	}
	t.Logf("estados en este equipo: %v", count)
}
