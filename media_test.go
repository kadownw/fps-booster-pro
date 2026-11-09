package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExeFromCommand(t *testing.T) {
	cases := map[string]string{
		`"C:\Program Files\Steam\steam.exe" -silent`: `C:\Program Files\Steam\steam.exe`,
		`C:\Tools\app.exe /min`:                      `C:\Tools\app.exe`,
		``:                                           ``,
	}
	for in, want := range cases {
		if got := exeFromCommand(in); got != want {
			t.Errorf("%q -> %q, esperado %q", in, got, want)
		}
	}
}

func TestSteamAppIDFromManifest(t *testing.T) {
	root := t.TempDir()
	steam := filepath.Join(root, "steamapps")
	os.MkdirAll(filepath.Join(steam, "common", "Counter-Strike Global Offensive", "game", "bin"), 0o755)
	os.WriteFile(filepath.Join(steam, "appmanifest_730.acf"), []byte("\"AppState\"\n{\n\t\"appid\"\t\t\"730\"\n\t\"installdir\"\t\t\"Counter-Strike Global Offensive\"\n}\n"), 0o644)
	exe := filepath.Join(steam, "common", "Counter-Strike Global Offensive", "game", "bin", "cs2.exe")
	if id := steamAppID(exe); id != "730" {
		t.Fatalf("appid = %q", id)
	}
	if id := steamAppID(`C:\Games\Foo\foo.exe`); id != "" {
		t.Fatalf("un juego fuera de Steam no debe tener appid: %q", id)
	}
}

func TestAppIconsExtractsRealIcon(t *testing.T) {
	p := filepath.Join(os.Getenv("WINDIR"), "System32", "notepad.exe")
	m := AppIcons([]string{p})
	if !strings.HasPrefix(m[p], "data:image/png;base64,") || len(m[p]) < 200 {
		t.Fatalf("no se obtuvo el icono de %s", p)
	}
}

func TestTierFor(t *testing.T) {
	l := tierList{Owners: []string{"111111111111111111"}, Pro: []string{"222222222222222222"}}
	if tierFor("111111111111111111", l) != "Propietario" || tierFor("222222222222222222", l) != "Pro Tier" || tierFor("333333333333333333", l) != "Free Tier" {
		t.Fatal("tiers incorrectos")
	}
	if DiscordSetClientID("abc") == "" || DiscordSetClientID("123") == "" {
		t.Fatal("un Application ID inválido debe rechazarse")
	}
}

func TestGameLibraryGroupsByExe(t *testing.T) {
	s := []GameSession{
		{Game: "A", Exe: "a.exe", Start: "2026-02-01T10:00:00Z", DurationSec: 100},
		{Game: "A", Exe: "A.exe", Start: "2026-01-01T10:00:00Z", DurationSec: 50, AvgFPS: 120, Low01FPS: 60, Low1FPS: 80},
		{Game: "B", Exe: "b.exe", Start: "2026-01-02T10:00:00Z", DurationSec: 10},
	}
	lib := gameLibrary(s)
	if len(lib) != 2 || lib[0].Sessions != 2 || lib[0].TotalSec != 150 || !lib[0].Measured || lib[0].AvgFPS != 120 || lib[1].Measured {
		t.Fatalf("biblioteca incorrecta: %+v", lib)
	}
}
