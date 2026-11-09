package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// Iconos de programas y portadas de juegos. Todo se guarda en caché en disco para no volver a
// pedirlo. Las portadas se descargan de la CDN pública de Steam (solo imágenes JPEG, con tope de
// tamaño); los juegos que no son de Steam usan el icono de su .exe.

var mediaMu sync.Mutex

func mediaDir(sub string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	d := filepath.Join(base, "SouthTweaks", sub)
	_ = os.MkdirAll(d, 0o700)
	return d
}

func hashKey(s string) string {
	h := sha1.Sum([]byte(strings.ToLower(s)))
	return hex.EncodeToString(h[:8])
}

// exeFromCommand extrae la ruta del .exe de un comando de inicio («"C:\x\a.exe" /min» o «C:\x\a.exe -x»).
func exeFromCommand(cmd string) string {
	c := strings.TrimSpace(cmd)
	if c == "" {
		return ""
	}
	if c[0] == '"' {
		if i := strings.Index(c[1:], `"`); i >= 0 {
			c = c[1 : 1+i]
		}
	} else if i := strings.Index(strings.ToLower(c), ".exe"); i >= 0 {
		c = c[:i+4]
	}
	if strings.Contains(c, "%") {
		if p, err := windows.UTF16PtrFromString(c); err == nil {
			buf := make([]uint16, 1024)
			if n, err := windows.ExpandEnvironmentStrings(p, &buf[0], uint32(len(buf))); err == nil && n > 0 {
				c = windows.UTF16ToString(buf)
			}
		}
	}
	return c
}

// AppIcons devuelve ruta -> data URL PNG del icono del programa (vacío si no se pudo).
func AppIcons(paths []string) map[string]string {
	out := map[string]string{}
	dir := mediaDir("icons")
	var todo []string
	for _, p := range paths {
		if p == "" || strings.ContainsAny(p, "|\r\n") {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(dir, hashKey(p)+".png")); err == nil && len(b) > 0 {
			out[p] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
		} else if _, err := os.Stat(p); err == nil {
			todo = append(todo, p)
		}
	}
	if len(todo) == 0 {
		return out
	}
	// Receta fija: las rutas viajan por una variable de entorno, nunca dentro del script.
	const script = `Add-Type -AssemblyName System.Drawing
foreach ($p in ($env:ST_ICON_PATHS -split '\|')) {
  try {
    $i = [System.Drawing.Icon]::ExtractAssociatedIcon($p)
    $ms = New-Object IO.MemoryStream
    $i.ToBitmap().Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
    $p + '|' + [Convert]::ToBase64String($ms.ToArray())
  } catch {}
}`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	cmd.Env = append(os.Environ(), "ST_ICON_PATHS="+strings.Join(todo, "|"))
	b, _ := cmd.Output()
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		i := strings.LastIndex(line, "|")
		if i <= 0 {
			continue
		}
		p, b64 := line[:i], line[i+1:]
		png, err := base64.StdEncoding.DecodeString(b64)
		if err != nil || len(png) == 0 {
			continue
		}
		_ = os.WriteFile(filepath.Join(dir, hashKey(p)+".png"), png, 0o600)
		out[p] = "data:image/png;base64," + b64
	}
	return out
}

var (
	acfInstallRe = regexp.MustCompile(`(?i)"installdir"\s+"([^"]+)"`)
	acfAppIDRe   = regexp.MustCompile(`(?i)"appid"\s+"(\d+)"`)
)

// steamAppID busca el appid del juego leyendo los appmanifest_*.acf de su biblioteca de Steam.
func steamAppID(exePath string) string {
	low := strings.ToLower(exePath)
	i := strings.Index(low, `\steamapps\common\`)
	if i < 0 {
		return ""
	}
	root := exePath[:i] + `\steamapps`
	rest := exePath[i+len(`\steamapps\common\`):]
	dir, _, _ := strings.Cut(rest, `\`)
	files, _ := filepath.Glob(filepath.Join(root, "appmanifest_*.acf"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		m := acfInstallRe.FindSubmatch(b)
		if m == nil || !strings.EqualFold(string(m[1]), dir) {
			continue
		}
		if id := acfAppIDRe.FindSubmatch(b); id != nil {
			return string(id[1])
		}
	}
	return ""
}

var coverClient = &http.Client{Timeout: 12 * time.Second}

func downloadJPEG(url string) []byte {
	resp, err := coverClient.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil || len(b) < 1000 || b[0] != 0xFF || b[1] != 0xD8 { // firma JPEG
		return nil
	}
	return b
}

// GameCover devuelve la portada (data URL) o "" si no hay: el frontend dibuja una de reemplazo.
func GameCover(exePath string) string {
	id := steamAppID(exePath)
	if id == "" {
		return ""
	}
	mediaMu.Lock()
	defer mediaMu.Unlock()
	file := filepath.Join(mediaDir("covers"), "steam_"+id+".jpg")
	if b, err := os.ReadFile(file); err == nil && len(b) > 0 {
		return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b)
	}
	for _, u := range []string{
		fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900_2x.jpg", id),
		fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", id),
		fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", id),
	} {
		if b := downloadJPEG(u); b != nil {
			_ = os.WriteFile(file, b, 0o600)
			return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b)
		}
	}
	return ""
}
