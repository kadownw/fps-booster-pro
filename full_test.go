package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Auditoría funcional completa. Todas estas pruebas MODIFICAN el sistema de verdad (y lo restauran), por
// eso solo corren como administrador y con FBP_ADMIN_TESTS=1. Cada una compara el estado antes y después.

func needAdminTests(t *testing.T) {
	t.Helper()
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() {
		t.Skip("define FBP_ADMIN_TESTS=1 y ejecuta como administrador")
	}
}

func keyID(c RegChange) string {
	return strings.ToLower(c.Root + `\` + c.Path + `\` + c.Name)
}

// universe: todas las claves que algún ajuste puede tocar (estáticas y las ya expandidas en este equipo).
func universe() map[string]RegChange {
	u := map[string]RegChange{}
	for _, tw := range catalog() {
		for _, c := range tw.Changes {
			if !strings.Contains(c.Path, "{") {
				u[keyID(c)] = c
			}
		}
		for _, c := range expand(tw) {
			u[keyID(c)] = c
		}
	}
	return u
}

func snapshotUniverse(u map[string]RegChange) map[string]RegState {
	out := map[string]RegState{}
	for k, c := range u {
		st, _ := readState(c)
		out[k] = st
	}
	return out
}

func diffUniverse(before, after map[string]RegState) []string {
	var bad []string
	for k, b := range before {
		if a := after[k]; !sameState(a, b) {
			bad = append(bad, fmt.Sprintf("%s: antes=%+v ahora=%+v", k, b, a))
		}
	}
	sort.Strings(bad)
	return bad
}

// Cada ajuste del catálogo se aplica de verdad, se verifica con el diagnóstico y se revierte.
func TestFullAuditEveryTweak(t *testing.T) {
	needAdminTests(t)
	e, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	if len(e.st.Items) != 0 {
		t.Fatalf("hay backups previos (%d): no se corre la prueba para no mezclar estados", len(e.st.Items))
	}
	uni := universe()
	before := snapshotUniverse(uni)

	var okList, skipList, failList []string
	for _, tw := range catalog() {
		tw := tw
		t.Run(tw.ID, func(t *testing.T) {
			switch {
			case tw.Once:
				skipList = append(skipList, tw.ID+" (acción única: se prueba aparte)")
				return
			case !supported(tw):
				skipList = append(skipList, tw.ID+" (no aplica a este equipo: "+whyUnsupported(tw)+")")
				return
			case tw.Fix && len(expand(tw)) == 0:
				skipList = append(skipList, tw.ID+" (arreglo sin nada que corregir en este equipo)")
				return
			}
			regs := expand(tw)
			rb := make([]RegState, len(regs))
			for i, c := range regs {
				rb[i], _ = readState(c)
			}
			tasksBefore := map[string]string{}
			for p, st := range existingTasks(tw) {
				tasksBefore[p] = st
			}
			var nvBefore []NvOrig
			if len(tw.Nvidia) > 0 {
				nvBefore, _ = nvReadOriginals(tw.Nvidia)
			}
			prevPlan, _ := activeScheme()
			fail := func(format string, a ...any) {
				failList = append(failList, tw.ID+": "+fmt.Sprintf(format, a...))
				t.Errorf(format, a...)
			}

			if err := e.Apply(tw.ID); err != nil {
				fail("aplicar: %v", err)
				return
			}
			// 1) el sistema quedó como pide el ajuste
			for i, c := range regs {
				if st, _ := readState(c); !satisfied(c, st) {
					if c.Optional { // Windows protege este valor: omitirlo es lo esperado
						t.Logf("valor protegido por Windows, omitido: %s", c.Name)
						continue
					}
					fail("%s no quedó aplicado (era %+v, ahora %+v)", c.Name, rb[i], st)
				}
			}
			if !tw.Fix {
				if d := e.diagnose(tw, e.isActive(tw)); d.Status != "applied" {
					fail("el diagnóstico dice %q después de aplicar: %s", d.Status, d.Detail)
				}
			}
			if len(tw.Nvidia) > 0 && !nvIsActive(tw.Nvidia) {
				fail("el perfil NVIDIA no quedó aplicado")
			}
			if tw.Power {
				if g, _ := activeScheme(); g != planTarget(tw) {
					fail("el plan activo es %s y debía ser %s", g, planTarget(tw))
				}
			}
			// 2) revertir y comprobar contra el original
			if err := e.Revert(tw.ID); err != nil {
				fail("revertir: %v", err)
				return
			}
			for i, c := range regs {
				if st, _ := readState(c); !sameState(st, rb[i]) {
					fail("%s no volvió al original (%+v vs %+v)", c.Name, st, rb[i])
				}
			}
			invalidateTasks()
			for p, was := range tasksBefore {
				if now := existingTasks(tw)[p]; now != was && !(was == "Running" && now == "Ready") {
					fail("la tarea %s era %q y quedó %q", p, was, now)
				}
			}
			if len(tw.Nvidia) > 0 {
				after, _ := nvReadOriginals(tw.Nvidia)
				for i := range nvBefore {
					if after[i] != nvBefore[i] {
						fail("el ajuste NVIDIA 0x%X no volvió (%+v vs %+v)", nvBefore[i].ID, after[i], nvBefore[i])
					}
				}
			}
			if tw.Power {
				if g, _ := activeScheme(); g != prevPlan {
					fail("el plan activo no volvió (%s vs %s)", g, prevPlan)
				}
			}
			if _, left := e.st.Items[tw.ID]; left {
				fail("quedó un backup sin borrar")
			}
			okList = append(okList, tw.ID)
		})
	}
	if bad := diffUniverse(before, snapshotUniverse(uni)); len(bad) > 0 {
		t.Errorf("después de TODO quedaron %d valores distintos al inicio:\n  %s", len(bad), strings.Join(bad, "\n  "))
	}
	if left := len(e.st.Items); left != 0 {
		t.Errorf("quedaron %d backups sin revertir", left)
	}
	t.Logf("APLICADOS Y REVERTIDOS OK: %d | OMITIDOS: %d | FALLARON: %d", len(okList), len(skipList), len(failList))
	for _, s := range skipList {
		t.Logf("  omitido: %s", s)
	}
	for _, s := range failList {
		t.Logf("  FALLÓ: %s", s)
	}
}

// El modo simulación no debe tocar NADA del sistema, aunque se apliquen todos los ajustes.
func TestFullSimulationTouchesNothing(t *testing.T) {
	uni := universe()
	before := snapshotUniverse(uni)
	plan, _ := activeScheme()
	e := &Engine{dir: t.TempDir(), st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: map[string]string{}, sim: true}
	for _, tw := range catalog() {
		if tw.Once {
			if _, err := e.RunOnce(tw.ID); err != nil {
				t.Errorf("simulación de %s: %v", tw.ID, err)
			}
			continue
		}
		if err := e.Apply(tw.ID); err != nil {
			t.Errorf("simulación de %s: %v", tw.ID, err)
		}
	}
	for _, f := range e.ListFixes() {
		if out := e.RunFix(f.ID); out.Error != "" || !out.Noop {
			t.Errorf("simulación del arreglo %s: %+v", f.ID, out)
		}
	}
	if bad := diffUniverse(before, snapshotUniverse(uni)); len(bad) > 0 {
		t.Fatalf("la simulación cambió %d valores reales: %v", len(bad), bad)
	}
	if g, _ := activeScheme(); g != plan {
		t.Fatalf("la simulación cambió el plan de energía: %s -> %s", plan, g)
	}
}

// Inicio de Windows: desactivar y volver a activar una entrada real, y dejar el valor EXACTO de antes.
func TestFullStartupToggle(t *testing.T) {
	needAdminTests(t)
	list := ListStartup()
	var target *StartupEntry
	for i := range list {
		if list[i].Scope == "user" {
			target = &list[i]
			break
		}
	}
	if target == nil {
		t.Skip("no hay entradas de inicio del usuario")
	}
	s, _ := findSource("user")
	rc := RegChange{Root: "HKCU", Path: s.Approved, Name: target.Name, Kind: "bin"}
	orig, err := readState(rc)
	if err != nil {
		t.Fatal(err)
	}
	defer writeState(rc, orig) // restaura el valor exacto pase lo que pase
	was := approvedEnabled(s, target.Name)
	if err := SetStartup(target.ID, !was); err != nil {
		t.Fatalf("cambiar: %v", err)
	}
	if approvedEnabled(s, target.Name) == was {
		t.Fatal("el cambio no se reflejó")
	}
	if err := SetStartup(target.ID, was); err != nil {
		t.Fatalf("volver: %v", err)
	}
	if approvedEnabled(s, target.Name) != was {
		t.Fatal("no volvió al estado original")
	}
	for _, bad := range []string{"user|", "|x", "nada", "machine|NoExisteEstaEntrada_zz", "inventado|x"} {
		if err := SetStartup(bad, false); err == nil {
			t.Errorf("aceptó la entrada inválida %q", bad)
		}
	}
	t.Logf("«%s» se desactivó y se volvió a activar; valor original restaurado", target.Name)
}

// Limpieza: con TEMP redirigido a una carpeta propia, borra solo lo viejo de esa carpeta.
func TestFullCleanupIsolated(t *testing.T) {
	needAdminTests(t)
	dir := filepath.Join(t.TempDir(), "mi_temp")
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	old := time.Now().Add(-3 * time.Hour)
	mk := func(rel string, age time.Time) string {
		p := filepath.Join(dir, rel)
		os.WriteFile(p, []byte("12345"), 0o644)
		os.Chtimes(p, age, age)
		return p
	}
	viejo, reciente := mk("viejo.tmp", old), mk("reciente.tmp", time.Now())
	mk(filepath.Join("sub", "otro.tmp"), old)
	t.Setenv("TEMP", dir)
	t.Setenv("TMP", dir)
	var found bool
	for _, it := range ScanCleanup() {
		if it.ID == "temp_user" {
			found = true
			if it.Files != 2 || it.Bytes != 10 {
				t.Errorf("la vista previa debía ver 2 archivos / 10 bytes y vio %d / %d", it.Files, it.Bytes)
			}
		}
	}
	if !found {
		t.Fatal("no apareció temp_user")
	}
	res, err := RunCleanup([]string{"temp_user"})
	if err != nil || res.Freed != 10 {
		t.Fatalf("RunCleanup: liberó %d, err=%v", res.Freed, err)
	}
	if _, err := os.Stat(viejo); err == nil {
		t.Error("el archivo viejo no se borró")
	}
	if _, err := os.Stat(reciente); err != nil {
		t.Error("el archivo reciente (menos de 1 h) se borró y debía respetarse")
	}
	if _, err := RunCleanup([]string{"no_existe"}); err == nil {
		t.Error("una categoría inexistente debía dar error")
	}
}

func TestFullTelemetryAndSysInfo(t *testing.T) {
	info := systemInfo()
	if info.CPU == "" || info.RAMGB < 1 || info.Windows == "" || info.Cores < 1 || len(info.GPUs) == 0 {
		t.Fatalf("systemInfo incompleto: %+v", info)
	}
	l := liveUsage()
	enrichLive(&l)
	if l.RAMTotal < 1 || len(l.Disks) == 0 || l.CPU < 0 || l.CPU > 100 {
		t.Fatalf("lectura en vivo incompleta: %+v", l)
	}
	t.Logf("%s | %s | %.1f GB | %s | CPU %.0f%% RAM %.0f%% GPU=%v", info.CPU, info.GPUs[0], info.RAMGB, info.Windows, l.CPU, l.RAM, l.GPU != nil)
}

func TestFullTimerResolution(t *testing.T) {
	coarse, fine, ok := timerLimits()
	if !ok || fine == 0 || fine > coarse {
		t.Fatalf("límites inválidos: %d %d %v", coarse, fine, ok)
	}
	cur, ok := setTimerBoost(true)
	if !ok || cur > coarse || cur == 0 {
		t.Fatalf("no se pudo subir la resolución: actual=%d ok=%v", cur, ok)
	}
	if _, ok := setTimerBoost(false); !ok {
		t.Fatal("no se pudo liberar")
	}
	t.Logf("resolución: peor %.2f ms, mejor %.2f ms; al subir quedó en %.2f ms", float64(coarse)/1e4, float64(fine)/1e4, float64(cur)/1e4)
}

// De punta a punta: un «juego» real (OpenGL) en una carpeta de Steam falsa -> detección, ETW y sesión guardada.
func TestFullGameDetectionFPSAndSessions(t *testing.T) {
	needAdminTests(t)
	root := t.TempDir()
	gameDir := filepath.Join(root, "steamapps", "common", "GLTest")
	os.MkdirAll(gameDir, 0o755)
	exe := filepath.Join(gameDir, "gltest.exe")
	if out, err := exec.Command("go", "build", "-o", exe, "./tools/gltest").CombinedOutput(); err != nil {
		t.Fatalf("no se pudo compilar gltest: %v\n%s", err, out)
	}
	cmd := exec.Command(exe, "40")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	time.Sleep(2 * time.Second)
	g := scanRunningGame()
	if g == nil || !strings.EqualFold(g.Exe, "gltest.exe") || g.Name != "GLTest" || g.Pid != uint32(cmd.Process.Pid) {
		t.Fatalf("el detector no reconoció el juego: %+v", g)
	}
	base := time.Now()
	offset := time.Duration(0)
	m := NewGameMonitor(t.TempDir(), func() (Live, bool) { return Live{CPU: 40, RAM: 50}, true })
	m.now = func() time.Time { return base.Add(offset) }
	m.tick() // detecta y empieza a medir
	if cur := m.Current(); cur == nil || cur.Game != "GLTest" {
		t.Fatalf("Current no devolvió la sesión: %+v", cur)
	}
	time.Sleep(5 * time.Second)
	cmd.Process.Kill()
	cmd.Wait()
	offset = 90 * time.Second // la sesión «duró» más de 30 s
	time.Sleep(time.Second)
	m.tick()
	m.tick() // 2 lecturas sin el proceso cierran la sesión
	s := m.Sessions()
	if len(s) != 1 {
		t.Fatalf("se esperaba 1 sesión guardada y hay %d", len(s))
	}
	if s[0].AvgFPS < 20 || s[0].Low01FPS <= 0 || s[0].Low1FPS <= 0 || s[0].Low01FPS > s[0].AvgFPS {
		t.Fatalf("FPS incoherentes: %+v", s[0])
	}
	lib := gameLibrary(s)
	if len(lib) != 1 || !lib[0].Measured {
		t.Fatalf("la biblioteca no refleja la sesión: %+v", lib)
	}
	t.Logf("sesión: %s | %.0f FPS medios | 1%% low %.0f | 0,1%% low %.0f | CPU %.0f%%", s[0].Game, s[0].AvgFPS, s[0].Low1FPS, s[0].Low01FPS, s[0].AvgCPU)
	// y que se guardó en disco
	m2 := NewGameMonitor(filepath.Dir(m.path), nil)
	if len(m2.Sessions()) != 1 {
		t.Error("la sesión no se guardó en disco")
	}
}
