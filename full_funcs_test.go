package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func runDiscordScenario(t *testing.T, tokenState string) DiscordUser {
	t.Helper()
	urlCh := make(chan string, 1)
	orig := openBrowser
	openBrowser = func(u string) error { urlCh <- u; return nil }
	defer func() { openBrowser = orig }()
	done := make(chan DiscordUser, 1)
	go func() { done <- DiscordLogin() }()
	var authURL string
	select {
	case authURL = <-urlCh:
	case <-time.After(5 * time.Second):
		t.Fatal("DiscordLogin no pidió abrir el navegador")
	}
	u, err := url.Parse(authURL)
	if err != nil || u.Host != "discord.com" || u.Path != "/oauth2/authorize" {
		t.Fatalf("URL de autorización inesperada: %s", authURL)
	}
	q := u.Query()
	if q.Get("client_id") != defaultClientID || q.Get("redirect_uri") != discordRedir || q.Get("response_type") != "token" || q.Get("scope") != "identify" || len(q.Get("state")) < 16 {
		t.Fatalf("parámetros OAuth incorrectos: %v", q)
	}
	// la página de retorno existe y reenvía el fragmento al servidor local
	resp, err := http.Get(discordRedir)
	if err != nil {
		t.Fatalf("el servidor local no responde: %v", err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(page), "/token") {
		t.Fatalf("página de retorno incorrecta (%d)", resp.StatusCode)
	}
	state := q.Get("state")
	if tokenState != "" {
		state = tokenState
	}
	body, _ := json.Marshal(map[string]string{"token": "token-de-prueba-invalido", "state": state})
	post := func(b []byte) int {
		r, err := http.Post(discordRedir[:len(discordRedir)-len("/callback")]+"/token", "application/json", strings.NewReader(string(b)))
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		return r.StatusCode
	}
	if tokenState != "" { // primero la petición falsa: debe rechazarse con 400 y dejar el login vivo
		if code := post(body); code != http.StatusBadRequest {
			t.Fatalf("el state falso debía dar 400 y dio %d", code)
		}
		good, _ := json.Marshal(map[string]string{"token": "token-de-prueba-invalido", "state": q.Get("state")})
		body = good
	}
	post(body)
	post(body) // una segunda respuesta no debe bloquear al servidor
	select {
	case res := <-done:
		return res
	case <-time.After(30 * time.Second):
		t.Fatal("DiscordLogin no terminó")
	}
	return DiscordUser{}
}

func TestFullDiscordLoopbackFlow(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	// 1) una respuesta con state falso se IGNORA: no cancela el inicio de sesión que espera el usuario
	// 2) la respuesta verdadera (state correcto) con un token inválido llega hasta Discord y se rechaza
	res := runDiscordScenario(t, "estado-falso")
	if res.LoggedIn || res.Error == "" {
		t.Fatalf("un token inventado no puede iniciar sesión: %+v", res)
	}
	if _, err := os.Stat(discordDir() + `\user.json`); err == nil {
		t.Fatal("se guardó una sesión con un token inválido")
	}
	t.Logf("con un token inventado: %q", res.Error)
}

func TestFullDiscordDefaultsAndTiers(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	if c := loadDiscordConfig(); c.ClientID != "1569198430893510716" {
		t.Fatalf("Application ID por defecto: %s", c.ClientID)
	}
	if u := DiscordState(); u.LoggedIn {
		t.Fatal("sin user.json no debe haber sesión")
	}
	l := fetchTiers("")
	if len(l.Owners) < 2 {
		t.Fatalf("tiers.json de GitHub: se esperaban los 2 propietarios y llegaron %v (¿sin internet o sin subir?)", l.Owners)
	}
	for _, id := range []string{"341296273846042625", "750325701898403912"} {
		if tierFor(id, l) != "Propietario" {
			t.Errorf("%s debería ser Propietario", id)
		}
	}
	if tierFor("111111111111111111", l) != "Free Tier" {
		t.Error("un ID desconocido debe ser Free Tier")
	}
}

func TestFullRunFixOnHealthySystem(t *testing.T) {
	needAdminTests(t)
	e := &Engine{dir: t.TempDir(), st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: map[string]string{}}
	for _, id := range []string{"fix_valorant_cfg", "fix_valorant_hvci", "fix_installer_2502"} {
		chk := e.FixCheck(id)
		if len(chk.Problems) > 0 {
			t.Logf("%s: este equipo SÍ tiene el problema (%v): se omite para no cambiar nada", id, chk.Problems)
			continue
		}
		out := e.RunFix(id)
		if out.Error != "" || !out.Noop || !strings.Contains(out.Message, "No se cambió nada") {
			t.Errorf("%s: sin problema debía ser un no-op: %+v", id, out)
		}
		if _, ran := e.runs[id]; ran {
			t.Errorf("%s: un no-op no debe quedar registrado como ejecutado", id)
		}
	}
	if out := e.RunFix("no_existe"); out.Error == "" {
		t.Error("un arreglo inexistente debe dar error")
	}
	if r := e.FixCheck("no_existe"); r.Error == "" {
		t.Error("FixCheck de un arreglo inexistente debe dar error")
	}
}

func TestFullNvidiaPanelFix(t *testing.T) {
	needAdminTests(t)
	e := &Engine{dir: t.TempDir(), st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: map[string]string{}}
	out := e.RunFix("fix_nvidia_panel")
	defer exec.Command("taskkill", "/IM", "nvcplui.exe", "/F").Run()
	t.Logf("resultado: error=%q mensaje=%q líneas=%v", out.Error, out.Message, out.Lines)
	if out.Error == "" && !strings.Contains(strings.Join(out.Lines, " "), "se abrió") {
		t.Errorf("dijo que anduvo pero no confirmó que el panel se abrió: %+v", out)
	}
}

func TestFullRestorePointMessage(t *testing.T) {
	needAdminTests(t)
	err := CreateRestorePoint()
	if err == nil {
		t.Log("se creó un punto de restauración")
		return
	}
	if st, ok := regServiceStart("VSS"); ok && st == startDisabled && !strings.Contains(err.Error(), "VSS") {
		t.Fatalf("con VSS deshabilitado el mensaje debe explicarlo: %v", err)
	}
	t.Logf("mensaje: %v", err)
}

// Acciones únicas: solo se ejecutan de verdad las inocuas; las que desinstalan cosas se comprueban en solo lectura.
func TestFullOnceActions(t *testing.T) {
	needAdminTests(t)
	e := &Engine{dir: t.TempDir(), st: store{Items: map[string]*Backup{}}, simApplied: map[string]bool{}, runs: map[string]string{}}

	msg, err := e.RunOnce("flush_dns")
	t.Logf("flush_dns -> %q err=%v", msg, err)
	if err != nil || e.runs["flush_dns"] == "" {
		t.Errorf("flush_dns debía andar y registrarse: %v", err)
	}

	msg, err = e.RunOnce("retrim_ssd")
	t.Logf("retrim_ssd -> %q err=%v", msg, err)
	if err != nil && !strings.Contains(err.Error(), "defrag") {
		t.Errorf("retrim_ssd: el error debe explicar la causa: %v", err)
	}

	// BitLocker: si Windows no deja consultar, debe decirlo y NO registrar la acción como hecha.
	msg, err = e.RunOnce("bitlocker_off")
	t.Logf("bitlocker_off -> %q err=%v", msg, err)
	if err != nil && e.runs["bitlocker_off"] != "" {
		t.Error("una consulta fallida no puede quedar registrada como ejecutada")
	}
	if err != nil && !strings.Contains(err.Error(), "No se pudo consultar BitLocker") {
		t.Errorf("el error no explica el motivo: %v", err)
	}

	// Copilot y Game Bar: solo se ejecutan si NO están instalados (así no se desinstala nada real).
	for _, c := range []struct{ id, pat string }{{"remove_copilot_app", "*Copilot*"}, {"remove_gamebar_app", "Microsoft.XboxGamingOverlay"}} {
		pk, qerr := appxList(c.pat)
		if qerr != nil || len(pk) > 0 {
			t.Logf("%s: hay %d paquetes instalados (o no se pudo consultar: %v): NO se ejecuta para no desinstalar nada", c.id, len(pk), qerr)
			continue
		}
		msg, err = e.RunOnce(c.id)
		if err != nil || !strings.Contains(msg, "No se cambió nada") || e.runs[c.id] != "" {
			t.Errorf("%s sin paquetes debía ser un no-op: %q %v", c.id, msg, err)
		}
	}

	// Apps preinstaladas: SOLO lectura. Informa cuáles quitaría.
	var present []string
	for _, b := range bloatApps {
		if pk, err := appxList(b[0]); err == nil && len(pk) > 0 {
			present = append(present, b[1])
		}
	}
	t.Logf("remove_bloat_apps quitaría (no se ejecutó): %v", present)
}
