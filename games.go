package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Detector de juegos: cada 3 s mira los procesos en ejecución y reconoce juegos por la carpeta
// donde están instalados (Steam, Epic, Riot, Xbox, GOG, Ubisoft, EA, Rockstar, Roblox). Mientras
// hay un juego abierto, promedia el uso de CPU, RAM y GPU y la temperatura de la GPU, y al
// cerrarse guarda la sesión. NO mide FPS (eso requiere PresentMon o captura ETW).

var gameDirs = []string{
	`\steamapps\common\`, `\epic games\`, `\riot games\`, `\xboxgames\`, `\gog galaxy\games\`, `\gog games\`,
	`\ubisoft game launcher\games\`, `\ea games\`, `\origin games\`, `\rockstar games\`, `\roblox\versions\`,
}

// Ejecutables auxiliares que viven en la carpeta de un juego pero no son el juego.
var gameSkip = []string{
	"crash", "unins", "setup", "redist", "vcredist", "dxsetup", "dotnet", "easyanticheat", "battleye", "anticheat",
	"launcher", "webhelper", "cefsubprocess", "overlay", "updater", "installer", "riotclient", "vanguard", "vgc.exe",
	"vgtray", "bootstrapper", "helper", "service", "diagnostic", "report", "benchmark_launcher", "epicwebhelper",
}

func isGamePath(path string) bool {
	p := strings.ToLower(path)
	base := filepath.Base(p)
	for _, s := range gameSkip {
		if strings.Contains(base, s) {
			return false
		}
	}
	if strings.Contains(p, `\epic games\launcher\`) || strings.Contains(p, `\epic games\epiconlineservices\`) ||
		strings.Contains(p, `\riot games\riot client\`) || strings.Contains(p, `\riot games\vanguard\`) {
		return false
	}
	for _, d := range gameDirs {
		if strings.Contains(p, d) {
			return true
		}
	}
	return false
}

// gameName deduce un nombre legible a partir de la ruta.
func gameName(path string) string {
	parts := strings.Split(path, `\`)
	for i, s := range parts {
		if strings.EqualFold(s, "common") && i > 0 && strings.EqualFold(parts[i-1], "steamapps") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

type runningGame struct{ Name, Exe, Path string }

func scanRunningGame() *runningGame {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if windows.Process32First(snap, &pe) != nil {
		return nil
	}
	for {
		if pe.ProcessID > 4 {
			if h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pe.ProcessID); err == nil {
				buf := make([]uint16, 1024)
				n := uint32(len(buf))
				if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) == nil {
					path := windows.UTF16ToString(buf[:n])
					if isGamePath(path) {
						windows.CloseHandle(h)
						return &runningGame{Name: gameName(path), Exe: filepath.Base(path), Path: path}
					}
				}
				windows.CloseHandle(h)
			}
		}
		if windows.Process32Next(snap, &pe) != nil {
			break
		}
	}
	return nil
}

// ---------- sesiones ----------

type GameSession struct {
	Game        string  `json:"game"`
	Exe         string  `json:"exe"`
	Start       string  `json:"start"`
	DurationSec int     `json:"durationSec"`
	AvgCPU      float64 `json:"avgCpu"`
	MaxCPU      float64 `json:"maxCpu"`
	AvgRAM      float64 `json:"avgRam"`
	AvgGPU      float64 `json:"avgGpu"`
	AvgGPUTemp  float64 `json:"avgGpuTemp"`
	MaxGPUTemp  int     `json:"maxGpuTemp"`
	Samples     int     `json:"samples"`
}

type active struct {
	game     runningGame
	start    time.Time
	n        int
	cpu, ram float64
	gpu, tmp float64
	gpuN     int
	maxCPU   float64
	maxTmp   int
}

func (a *active) add(l Live) {
	a.n++
	a.cpu += l.CPU
	a.ram += l.RAM
	if l.CPU > a.maxCPU {
		a.maxCPU = l.CPU
	}
	if l.GPU != nil {
		a.gpuN++
		a.gpu += float64(l.GPU.Util)
		if l.GPU.Temp >= 0 {
			a.tmp += float64(l.GPU.Temp)
			if l.GPU.Temp > a.maxTmp {
				a.maxTmp = l.GPU.Temp
			}
		}
	}
}

func (a *active) snapshot(now time.Time) GameSession {
	s := GameSession{Game: a.game.Name, Exe: a.game.Exe, Start: a.start.Format(time.RFC3339),
		DurationSec: int(now.Sub(a.start).Seconds()), Samples: a.n, MaxCPU: a.maxCPU, MaxGPUTemp: a.maxTmp}
	if a.n > 0 {
		s.AvgCPU, s.AvgRAM = a.cpu/float64(a.n), a.ram/float64(a.n)
	}
	if a.gpuN > 0 {
		s.AvgGPU, s.AvgGPUTemp = a.gpu/float64(a.gpuN), a.tmp/float64(a.gpuN)
	}
	return s
}

type GameMonitor struct {
	mu       sync.Mutex
	path     string
	sessions []GameSession
	cur      *active
	missing  int
	lastLive func() (Live, bool)
	scan     func() *runningGame
	now      func() time.Time
}

const (
	minSession  = 30 * time.Second
	maxSessions = 200
)

func NewGameMonitor(dir string, lastLive func() (Live, bool)) *GameMonitor {
	m := &GameMonitor{path: filepath.Join(dir, "sessions.json"), lastLive: lastLive, scan: scanRunningGame, now: time.Now}
	if b, err := os.ReadFile(m.path); err == nil {
		json.Unmarshal(b, &m.sessions) // si está dañado se empieza vacío
	}
	return m
}

func (m *GameMonitor) save() {
	b, err := json.MarshalIndent(m.sessions, "", "  ")
	if err != nil {
		return
	}
	tmp := m.path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, m.path)
	}
}

// tick: un paso del detector (se llama cada ~3 s).
func (m *GameMonitor) tick() {
	g := m.scan()
	syncTimer(g != nil) // resolución de temporizador alta solo mientras hay un juego abierto
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if g != nil {
		m.missing = 0
		if m.cur == nil || m.cur.game.Exe != g.Exe {
			m.finish(now)
			m.cur = &active{game: *g, start: now}
		}
		if l, ok := m.lastLive(); ok {
			m.cur.add(l)
		}
		return
	}
	if m.cur != nil {
		m.missing++
		if m.missing >= 2 { // el proceso desapareció 2 lecturas seguidas
			m.finish(now)
		}
	}
}

func (m *GameMonitor) finish(now time.Time) {
	if m.cur == nil {
		return
	}
	s := m.cur.snapshot(now)
	m.cur, m.missing = nil, 0
	if time.Duration(s.DurationSec)*time.Second < minSession {
		return
	}
	m.sessions = append([]GameSession{s}, m.sessions...)
	if len(m.sessions) > maxSessions {
		m.sessions = m.sessions[:maxSessions]
	}
	m.save()
}

func (m *GameMonitor) Run(stop <-chan struct{}) {
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			m.tick()
		}
	}
}

func (m *GameMonitor) Current() *GameSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cur == nil {
		return nil
	}
	s := m.cur.snapshot(m.now())
	return &s
}

func (m *GameMonitor) Sessions() []GameSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]GameSession(nil), m.sessions...)
}

func (m *GameMonitor) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = nil
	m.save()
}
