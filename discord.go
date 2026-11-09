package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Inicio de sesión con Discord (OAuth2, flujo «implicit»: no hace falta ningún secreto en la app).
//
// Configuración (una sola vez): crear una aplicación en https://discord.com/developers/applications,
// copiar su «Application ID» y agregar en OAuth2 → Redirects exactamente:
//     http://127.0.0.1:53682/callback
// La app pide solo el permiso «identify» (nombre, id y avatar). No lee mensajes, servidores ni correo.
// No se guarda ningún token: se usa una vez para leer el perfil y se descarta.
//
// El tier sale de una lista de IDs de Discord (tiers.json): Propietario, Pro Tier o Free Tier.

const (
	discordPort    = 53682
	discordRedir   = "http://127.0.0.1:53682/callback"
	defaultTierURL = "https://raw.githubusercontent.com/kadownw/fps-booster-pro/main/tiers.json"
)

var snowflakeRe = regexp.MustCompile(`^\d{15,25}$`)

type DiscordUser struct {
	LoggedIn   bool   `json:"loggedIn"`
	Configured bool   `json:"configured"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"` // data URL
	Tier       string `json:"tier"`   // "Free Tier" | "Pro Tier" | "Propietario"
	Error      string `json:"error"`
}

type discordConfig struct {
	ClientID string `json:"clientId"`
	TierURL  string `json:"tierUrl,omitempty"`
}

type tierList struct {
	Owners []string `json:"owners"`
	Pro    []string `json:"pro"`
}

type discordSaved struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

var discordMu sync.Mutex

func discordDir() string { return mediaDir("discord") }

func loadDiscordConfig() discordConfig {
	var c discordConfig
	if b, err := os.ReadFile(filepath.Join(discordDir(), "config.json")); err == nil {
		json.Unmarshal(b, &c)
	}
	return c
}

// DiscordSetClientID guarda el Application ID (solo dígitos).
func DiscordSetClientID(id string) string {
	id = strings.TrimSpace(id)
	if !snowflakeRe.MatchString(id) {
		return "El Application ID son solo números (15 a 25 dígitos)."
	}
	c := loadDiscordConfig()
	c.ClientID = id
	b, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(filepath.Join(discordDir(), "config.json"), b, 0o600); err != nil {
		return err.Error()
	}
	return ""
}

func tierFor(id string, list tierList) string {
	for _, o := range list.Owners {
		if o == id {
			return "Propietario"
		}
	}
	for _, p := range list.Pro {
		if p == id {
			return "Pro Tier"
		}
	}
	return "Free Tier"
}

func fetchTiers(urlStr string) tierList {
	var l tierList
	if urlStr == "" {
		urlStr = defaultTierURL
	}
	cl := &http.Client{Timeout: 8 * time.Second}
	if resp, err := cl.Get(urlStr); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			json.Unmarshal(b, &l)
		}
	}
	// Lista local opcional (útil sin conexión o para pruebas).
	if b, err := os.ReadFile(filepath.Join(discordDir(), "tiers.json")); err == nil {
		var loc tierList
		if json.Unmarshal(b, &loc) == nil {
			l.Owners = append(l.Owners, loc.Owners...)
			l.Pro = append(l.Pro, loc.Pro...)
		}
	}
	return l
}

// DiscordState devuelve el usuario guardado (sin red; el tier se recalcula al iniciar la app).
func DiscordState() DiscordUser {
	cfg := loadDiscordConfig()
	u := DiscordUser{Configured: cfg.ClientID != ""}
	b, err := os.ReadFile(filepath.Join(discordDir(), "user.json"))
	if err != nil {
		return u
	}
	var s discordSaved
	if json.Unmarshal(b, &s) != nil || s.ID == "" {
		return u
	}
	u.LoggedIn, u.ID, u.Name, u.Avatar = true, s.ID, s.Name, s.Avatar
	u.Tier = tierFor(s.ID, fetchTiers(cfg.TierURL))
	return u
}

func DiscordLogout() {
	os.Remove(filepath.Join(discordDir(), "user.json"))
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Con el flujo implicit el token llega en el fragmento (#...) y el navegador no lo envía al
// servidor: esta página lo reenvía a /token en el mismo equipo.
const callbackPage = `<!doctype html><meta charset="utf-8"><title>South Tweaks</title>
<body style="font-family:system-ui;background:#0b0c0e;color:#e8eaee;display:grid;place-items:center;height:100vh;margin:0">
<div id="m">Conectando con South Tweaks…</div>
<script>
const p = new URLSearchParams(location.hash.slice(1));
fetch('/token', {method:'POST', body: JSON.stringify({token: p.get('access_token')||'', state: p.get('state')||'', error: new URLSearchParams(location.search).get('error')||''})})
 .then(()=>{ document.getElementById('m').textContent = 'Listo. Ya podés volver a South Tweaks y cerrar esta pestaña.'; })
 .catch(()=>{ document.getElementById('m').textContent = 'No se pudo completar. Volvé a intentarlo desde la app.'; });
</script>`

func openBrowser(u string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

// DiscordLogin abre el navegador, espera la autorización (hasta 3 min) y guarda el perfil.
func DiscordLogin() DiscordUser {
	discordMu.Lock()
	defer discordMu.Unlock()
	cfg := loadDiscordConfig()
	res := DiscordUser{Configured: cfg.ClientID != ""}
	if cfg.ClientID == "" {
		res.Error = "Falta el Application ID de tu aplicación de Discord."
		return res
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", discordPort))
	if err != nil {
		res.Error = "El puerto 53682 está ocupado. Cerrá otras ventanas de inicio de sesión e intentá de nuevo."
		return res
	}
	state := randHex(16)
	tokenCh := make(chan string, 1)
	errCh := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, callbackPage)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Host != fmt.Sprintf("127.0.0.1:%d", discordPort) {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		var in struct{ Token, State, Error string }
		if json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&in) != nil {
			http.Error(w, "mal", http.StatusBadRequest)
			return
		}
		switch {
		case in.Error != "":
			errCh <- "Cancelaste el inicio de sesión."
		case in.State != state || in.Token == "":
			errCh <- "La respuesta de Discord no es válida (state distinto)."
		default:
			tokenCh <- in.Token
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	q := url.Values{"client_id": {cfg.ClientID}, "redirect_uri": {discordRedir}, "response_type": {"token"}, "scope": {"identify"}, "state": {state}}
	if err := openBrowser("https://discord.com/oauth2/authorize?" + q.Encode()); err != nil {
		res.Error = "No se pudo abrir el navegador."
		return res
	}
	var token string
	select {
	case token = <-tokenCh:
	case msg := <-errCh:
		res.Error = msg
		return res
	case <-time.After(3 * time.Minute):
		res.Error = "Se agotó el tiempo de espera."
		return res
	}

	user, err := fetchDiscordUser(token)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	b, _ := json.Marshal(user)
	_ = os.WriteFile(filepath.Join(discordDir(), "user.json"), b, 0o600)
	return DiscordState()
}

func fetchDiscordUser(token string) (discordSaved, error) {
	cl := &http.Client{Timeout: 12 * time.Second}
	req, _ := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := cl.Do(req)
	if err != nil {
		return discordSaved{}, errors.New("no se pudo conectar con Discord")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return discordSaved{}, errors.New("Discord rechazó la sesión (" + resp.Status + ")")
	}
	var d struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		GlobalName    string `json:"global_name"`
		Avatar        string `json:"avatar"`
		Discriminator string `json:"discriminator"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&d) != nil || !snowflakeRe.MatchString(d.ID) {
		return discordSaved{}, errors.New("respuesta de Discord ilegible")
	}
	name := d.GlobalName
	if name == "" {
		name = d.Username
	}
	avatarURL := ""
	if d.Avatar != "" && regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(d.Avatar) {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=128", d.ID, d.Avatar)
	} else if n, err := strconv.ParseUint(d.ID, 10, 64); err == nil {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", (n>>22)%6)
	}
	out := discordSaved{ID: d.ID, Name: name}
	if avatarURL != "" {
		if r2, err := cl.Get(avatarURL); err == nil {
			defer r2.Body.Close()
			if img, err := io.ReadAll(io.LimitReader(r2.Body, 1<<20)); err == nil && r2.StatusCode == 200 {
				out.Avatar = "data:image/png;base64," + base64.StdEncoding.EncodeToString(img)
			}
		}
	}
	return out, nil
}
