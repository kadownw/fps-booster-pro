package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestIsGamePath(t *testing.T) {
	yes := []string{
		`C:\Program Files (x86)\Steam\steamapps\common\Counter-Strike Global Offensive\game\bin\win64\cs2.exe`,
		`D:\Games\Epic Games\Fortnite\FortniteGame\Binaries\Win64\FortniteClient-Win64-Shipping.exe`,
		`C:\Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`,
		`C:\Users\Traw\AppData\Local\Roblox\Versions\version-5a21b87bbc2641cf\RobloxPlayerBeta.exe`,
	}
	no := []string{
		`C:\Program Files (x86)\Steam\steam.exe`,
		`C:\Program Files (x86)\Steam\steamapps\common\Elden Ring\Game\crashpad_handler.exe`,
		`C:\Program Files (x86)\Steam\steamapps\common\Elden Ring\Game\EasyAntiCheat_EOS.exe`,
		`C:\Program Files\Epic Games\Launcher\Portal\Binaries\Win64\EpicGamesLauncher.exe`,
		`C:\Riot Games\Riot Client\RiotClientServices.exe`,
		`C:\Windows\System32\notepad.exe`,
		`C:\Users\Traw\AppData\Local\Discord\app-1.0\Discord.exe`,
	}
	for _, p := range yes {
		if !isGamePath(p) {
			t.Errorf("debía ser juego: %s", p)
		}
	}
	for _, p := range no {
		if isGamePath(p) {
			t.Errorf("no debía ser juego: %s", p)
		}
	}
	if n := gameName(`C:\x\steamapps\common\Counter-Strike Global Offensive\game\cs2.exe`); n != "Counter-Strike Global Offensive" {
		t.Errorf("nombre de Steam: %q", n)
	}
	if n := gameName(`C:\Riot Games\VALORANT\live\VALORANT-Win64-Shipping.exe`); n != "VALORANT-Win64-Shipping" {
		t.Errorf("nombre por exe: %q", n)
	}
}

func TestSessionLifecycle(t *testing.T) {
	now := time.Date(2026, 1, 1, 20, 0, 0, 0, time.UTC)
	var game *runningGame
	live := Live{CPU: 50, RAM: 60, GPU: &GPULive{Util: 90, Temp: 70}}
	m := &GameMonitor{path: filepath.Join(t.TempDir(), "s.json"), now: func() time.Time { return now },
		scan: func() *runningGame { return game }, lastLive: func() (Live, bool) { return live, true }}

	game = &runningGame{Name: "Juego", Exe: "juego.exe"}
	m.tick() // empieza
	if m.Current() == nil {
		t.Fatal("debía detectar el juego")
	}
	now = now.Add(10 * time.Minute)
	live = Live{CPU: 70, RAM: 62, GPU: &GPULive{Util: 98, Temp: 80}}
	m.tick()
	game = nil
	m.tick() // 1.ª lectura sin el proceso: todavía no cierra
	if m.Current() == nil {
		t.Fatal("cerró la sesión demasiado pronto")
	}
	m.tick() // 2.ª: cierra
	if m.Current() != nil {
		t.Fatal("debía cerrar la sesión")
	}
	s := m.Sessions()
	if len(s) != 1 {
		t.Fatalf("sesiones: %d", len(s))
	}
	if s[0].DurationSec != 600 || s[0].AvgCPU != 60 || s[0].MaxGPUTemp != 80 || s[0].AvgGPU != 94 || s[0].Samples != 2 {
		t.Fatalf("promedios incorrectos: %+v", s[0])
	}

	// una sesión de menos de 30 s no se guarda
	game = &runningGame{Name: "Corto", Exe: "corto.exe"}
	m.tick()
	now = now.Add(10 * time.Second)
	game = nil
	m.tick()
	m.tick()
	if len(m.Sessions()) != 1 {
		t.Fatal("guardó una sesión demasiado corta")
	}

	// persistencia
	m2 := NewGameMonitor(filepath.Dir(m.path), func() (Live, bool) { return Live{}, false })
	m2.path = m.path
	m2.sessions = nil
	m2.sessions = append(m2.sessions, s...)
	if len(m2.Sessions()) != 1 {
		t.Fatal("no se leyó la sesión guardada")
	}
}
