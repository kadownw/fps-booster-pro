package main

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

func TestProfilesReferenceRealTweaks(t *testing.T) {
	for _, p := range profileDefs {
		if len(p.Tweaks) == 0 {
			t.Errorf("perfil %s vacío", p.ID)
		}
		seen := map[string]bool{}
		for _, id := range p.Tweaks {
			if _, ok := findTweak(id); !ok {
				t.Errorf("perfil %s: ajuste inexistente %q", p.ID, id)
			}
			if seen[id] {
				t.Errorf("perfil %s: %q repetido", p.ID, id)
			}
			seen[id] = true
		}
	}
	// el perfil sin admin no debe pedir admin
	for _, p := range profiles() {
		if p.ID == "user" && p.NeedsAdmin {
			t.Error("el perfil 'user' no debería necesitar administrador")
		}
	}
}

func TestProfileInSimulation(t *testing.T) {
	e, _ := NewEngine()
	e.SetSimulation(true)
	res, err := e.ApplyProfile("user")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Failed) != 0 || len(res.Applied) == 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if err := e.RevertAll(); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ApplyProfile("nope"); err == nil {
		t.Fatal("aceptó un perfil inexistente")
	}
}

func TestCleanDirRespectsAgeAndSymlinks(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "sub", "viejo.tmp")
	fresh := filepath.Join(root, "nuevo.tmp")
	os.MkdirAll(filepath.Dir(old), 0o755)
	os.WriteFile(old, make([]byte, 1000), 0o644)
	os.WriteFile(fresh, make([]byte, 500), 0o644)
	past := time.Now().Add(-5 * time.Hour)
	os.Chtimes(old, past, past)

	// carpeta externa que NO se debe tocar, enlazada desde root
	outside := t.TempDir()
	keep := filepath.Join(outside, "importante.txt")
	os.WriteFile(keep, []byte("no borrar"), 0o644)
	os.Chtimes(keep, past, past)
	_ = os.Symlink(outside, filepath.Join(root, "enlace")) // puede fallar sin privilegios: el test sigue válido

	b, n := scanDir(root, time.Hour)
	if n != 1 || b != 1000 {
		t.Fatalf("scan: %d archivos, %d bytes (esperaba 1 y 1000)", n, b)
	}
	freed, _ := cleanDir(root, time.Hour)
	if freed != 1000 {
		t.Fatalf("liberó %d, esperaba 1000", freed)
	}
	if _, err := os.Stat(old); err == nil {
		t.Fatal("no borró el archivo viejo")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("borró un archivo reciente")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("siguió un enlace y borró fuera de la carpeta")
	}
}

func TestRunCleanupRejectsUnknown(t *testing.T) {
	if _, err := RunCleanup([]string{`C:\Windows`, "../x"}); err == nil {
		t.Fatal("aceptó categorías que no están en la lista fija")
	}
}

func TestStartupToggleRoundTrip(t *testing.T) {
	const name = "FBP_PruebaInicio"
	run, err := registry.OpenKey(registry.CURRENT_USER, runBase+`\Run`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	if err := run.SetStringValue(name, `C:\Windows\System32\notepad.exe`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		run.DeleteValue(name)
		if k, err := registry.OpenKey(registry.CURRENT_USER, startupSources[0].Approved, registry.SET_VALUE); err == nil {
			k.DeleteValue(name)
			k.Close()
		}
	}()
	find := func() *StartupEntry {
		for _, e := range ListStartup() {
			if e.ID == "user|"+name {
				return &e
			}
		}
		return nil
	}
	if e := find(); e == nil || !e.Enabled {
		t.Fatalf("debía aparecer activa: %+v", e)
	}
	if err := SetStartup("user|"+name, false); err != nil {
		t.Fatal(err)
	}
	if e := find(); e == nil || e.Enabled {
		t.Fatalf("debía quedar desactivada: %+v", e)
	}
	if err := SetStartup("user|"+name, true); err != nil {
		t.Fatal(err)
	}
	if e := find(); e == nil || !e.Enabled {
		t.Fatalf("debía volver a activa: %+v", e)
	}
	if err := SetStartup("user|NoExiste_zz", false); err == nil {
		t.Fatal("aceptó una entrada inexistente")
	}
	if err := SetStartup("raro", false); err == nil {
		t.Fatal("aceptó un id inválido")
	}
}

func TestKVMerge(t *testing.T) {
	if got := mergeKV("", "SwapEffectUpgradeEnable=1"); got != "SwapEffectUpgradeEnable=1;" {
		t.Fatalf("vacío: %q", got)
	}
	cur := "VRROptimizeEnable=0;AutoHDREnable=1;"
	got := mergeKV(cur, "SwapEffectUpgradeEnable=1")
	if !kvHas(got, "SwapEffectUpgradeEnable=1") || !kvHas(got, "VRROptimizeEnable=0") || !kvHas(got, "AutoHDREnable=1") {
		t.Fatalf("no conservó las claves existentes: %q", got)
	}
	if got := mergeKV("SwapEffectUpgradeEnable=0;X=2;", "SwapEffectUpgradeEnable=1"); got != "SwapEffectUpgradeEnable=1;X=2;" {
		t.Fatalf("reemplazo: %q", got)
	}
	if kvHas("A=1;", "B=1") || kvHas("", "") {
		t.Fatal("kvHas aceptó algo que no está")
	}
}

func TestLinearCurvesAreWellFormed(t *testing.T) {
	for name, h := range map[string]string{"X": linearCurveX, "Y": linearCurveY} {
		if len(h) != 80 { // 5 puntos x 8 bytes x 2 caracteres: un valor impar no se puede escribir como REG_BINARY
			t.Errorf("curva %s: %d caracteres, esperaba 80", name, len(h))
		}
		if _, err := hex.DecodeString(h); err != nil {
			t.Errorf("curva %s no es hexadecimal: %v", name, err)
		}
	}
}

func TestDateValues(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	if got := dateFor("now", now); got != "2026-11-08T12:00:00Z" {
		t.Fatalf("now: %s", got)
	}
	if got := dateFor("+35d", now); got != "2026-12-13T12:00:00Z" {
		t.Fatalf("+35d: %s", got)
	}
	future := RegChange{Date: "+35d"}
	if !satisfied(future, RegState{Exists: true, Kind: "sz", S: dateFor("+35d", time.Now())}) {
		t.Fatal("una pausa vigente debe contar como aplicada")
	}
	if satisfied(future, RegState{Exists: true, Kind: "sz", S: dateFor("+1d", time.Now().AddDate(0, 0, -10))}) {
		t.Fatal("una pausa vencida no debe contar como aplicada")
	}
}

func TestRegBinaryRoundTrip(t *testing.T) {
	c := RegChange{Root: "HKCU", Path: `Software\SouthTweaksTest`, Name: "Bin", Kind: "bin", Hex: "00ff10"}
	defer func() {
		if k, err := registry.OpenKey(registry.CURRENT_USER, c.Path, registry.SET_VALUE); err == nil {
			k.DeleteValue(c.Name)
			k.Close()
		}
		registry.DeleteKey(registry.CURRENT_USER, c.Path)
	}()
	if err := writeState(c, desired(c)); err != nil {
		t.Fatal(err)
	}
	st, err := readState(c)
	if err != nil || !sameState(st, desired(c)) {
		t.Fatalf("lectura binaria: %+v %v", st, err)
	}
}

func TestNICAdaptersAreNotVirtual(t *testing.T) {
	for _, sub := range nicAdapters() {
		desc := regString(registry.LOCAL_MACHINE, netClassKeyPath+`\`+sub, "DriverDesc")
		if virtualNIC.MatchString(desc) {
			t.Errorf("adaptador virtual incluido: %s", desc)
		}
	}
	t.Logf("adaptadores físicos detectados: %d", len(nicAdapters()))
}

func TestPSOnlyRunsDeclaredCommands(t *testing.T) {
	if err := runAllowedPS("Get-Process"); err == nil {
		t.Fatal("ejecutó un comando de PowerShell que no está en el catálogo")
	}
	if err := runAllowedPS("Disable-MMAgent -MemoryCompression; calc"); err == nil {
		t.Fatal("aceptó un comando con algo extra pegado")
	}
	tw, _ := findTweak("memory_compression_off")
	if tw.PS == nil || !allowedPS(tw.PS.Apply) || !allowedPS(tw.PS.Revert) {
		t.Fatal("los comandos declarados deberían estar permitidos")
	}
}

func TestAppxPatternsAreFixed(t *testing.T) {
	for _, bad := range []string{"x'; calc; '", "a b", "", "a|b", "$(calc)"} {
		if _, err := removeAppx(bad); err == nil {
			t.Errorf("aceptó el patrón %q", bad)
		}
	}
}

func TestTimerResolutionLimits(t *testing.T) {
	coarse, fine, ok := timerLimits()
	if !ok || fine == 0 || fine > coarse {
		t.Fatalf("límites inválidos: %d %d %v", coarse, fine, ok)
	}
	if cur, ok := setTimerBoost(true); !ok || cur == 0 {
		t.Fatalf("no se pudo pedir la resolución máxima: %d %v", cur, ok)
	}
	setTimerBoost(false) // se libera siempre
	t.Logf("resolución: peor %.2f ms, mejor %.2f ms", float64(coarse)/1e4, float64(fine)/1e4)
}

func TestDisplayDevices(t *testing.T) {
	d := displayDevices()
	for _, p := range d {
		if !strings.HasPrefix(p, `PCI\`) {
			t.Errorf("ruta inesperada: %s", p)
		}
	}
	t.Logf("dispositivos de video PCI: %v", d)
}

func TestExpandSZKindIsPreserved(t *testing.T) {
	c := RegChange{Root: "HKCU", Path: `Software\SouthTweaksTest`, Name: "Esz"}
	defer registry.DeleteKey(registry.CURRENT_USER, c.Path)
	k, _, err := registry.CreateKey(registry.CURRENT_USER, c.Path, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	k.SetExpandStringValue("Esz", `%SystemRoot%\Media\x.wav`)
	k.Close()
	orig, _ := readState(c)
	if orig.Kind != "esz" {
		t.Fatalf("no detectó REG_EXPAND_SZ: %+v", orig)
	}
	writeState(c, RegState{Exists: true, Kind: "sz", S: ""})
	writeState(c, orig) // restaurar
	k2, _ := registry.OpenKey(registry.CURRENT_USER, c.Path, registry.QUERY_VALUE)
	defer k2.Close()
	_, typ, _ := k2.GetValue("Esz", nil)
	k3, _ := registry.OpenKey(registry.CURRENT_USER, c.Path, registry.SET_VALUE)
	k3.DeleteValue("Esz")
	k3.Close()
	if typ != registry.EXPAND_SZ {
		t.Fatalf("el tipo cambió al restaurar: %d", typ)
	}
}

func TestMaxPerfServicesAreSafeAndUnshared(t *testing.T) {
	mp, ok := findTweak("svc_max_performance")
	if !ok || len(mp.Changes) < 100 {
		t.Fatalf("svc_max_performance incompleto: %d cambios", len(mp.Changes))
	}
	protected := []string{"BFE", "mpssvc", "EventLog", "EventSystem", "PlugPlay", "SamSs", "Themes", "WlanSvc",
		"NlaSvc", "Dnscache", "AudioSrv", "Audiosrv", "AudioEndpointBuilder", "TokenBroker", "ClipSVC", "wuauserv",
		"UsoSvc", "VSS", "bthserv", "hidserv", "LanmanWorkstation", "WinDefend", "Winmgmt", "RpcSs", "DcomLaunch"}
	own := map[string]bool{}
	for _, c := range mp.Changes {
		if c.D != startDisabled {
			t.Fatalf("%s debe quedar deshabilitado", c.Path)
		}
		own[strings.ToLower(c.Path)] = true
		for _, p := range protected {
			if strings.EqualFold(c.Path, `SYSTEM\CurrentControlSet\Services\`+p) {
				t.Fatalf("servicio protegido en la lista: %s", p)
			}
		}
	}
	for _, o := range catalog() {
		if o.ID == mp.ID {
			continue
		}
		for _, c := range o.Changes {
			if own[strings.ToLower(c.Path)] && strings.EqualFold(c.Name, "Start") {
				t.Fatalf("%s comparte la clave %s con svc_max_performance", o.ID, c.Path)
			}
		}
	}
	if b, ok := findTweak("bitlocker_off"); !ok || !b.Once || !b.Admin {
		t.Fatal("bitlocker_off debe ser una acción única de administrador")
	}
}

func TestTierRules(t *testing.T) {
	if !tweakAllowed(tierFree, "game_mode") || tweakAllowed(tierFree, "svc_max_performance") || tweakAllowed(tierFree, "bitlocker_off") {
		t.Fatal("Free Tier: solo ajustes básicos")
	}
	if !tweakAllowed(tierPro, "svc_max_performance") || !tweakAllowed(tierOwner, "bitlocker_off") {
		t.Fatal("Pro y Propietario lo tienen todo")
	}
	if tweakAllowed("", "game_mode") {
		t.Fatal("sin sesión no se permite nada")
	}
	for id := range freeTweaks {
		tw, ok := findTweak(id)
		if !ok {
			t.Fatalf("freeTweaks apunta a un ajuste que no existe: %s", id)
		}
		if tw.Risk != "safe" {
			t.Fatalf("%s no es seguro y no puede ser básico", id)
		}
	}
}

func TestEngineBlocksFreeTier(t *testing.T) {
	e := &Engine{st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: map[string]string{}, sim: true}
	tier := tierFree
	e.tier = func() string { return tier }
	if err := e.Apply("svc_max_performance"); err != errTier {
		t.Fatalf("Free debe recibir errTier, recibió %v", err)
	}
	if _, err := e.RunOnce("bitlocker_off"); err != errTier {
		t.Fatalf("Free no puede ejecutar acciones Pro: %v", err)
	}
	locked := 0
	for _, v := range e.List() {
		if v.Locked {
			locked++
		}
	}
	if locked < 50 {
		t.Fatalf("Free debería ver muchos ajustes bloqueados, vio %d", locked)
	}
	tier = tierPro
	if err := e.Apply("svc_max_performance"); err == errTier {
		t.Fatal("Pro no debe estar bloqueado")
	}
}
