package main

import (
	"context"
	"os"
	"sync"
	"syscall"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

// App expone al frontend (Wails) solo lo necesario.
type App struct {
	ctx      context.Context
	engine   *Engine
	initEr   string
	games    *GameMonitor
	stopMon  chan struct{}
	liveMu   sync.Mutex
	lastLive Live
	hasLive  bool
}

func NewApp() *App {
	e, err := NewEngine()
	a := &App{engine: e}
	go taskStates() // precalienta la consulta de tareas programadas (tarda ~2 s)
	if err != nil {
		a.initEr = err.Error()
	} else {
		a.games = NewGameMonitor(e.dir, a.latestLive)
	}
	return a
}

func (a *App) latestLive() (Live, bool) {
	a.liveMu.Lock()
	defer a.liveMu.Unlock()
	return a.lastLive, a.hasLive
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.games != nil {
		a.stopMon = make(chan struct{})
		go a.games.Run(a.stopMon)
	}
}

func (a *App) shutdown(ctx context.Context) {
	releaseTimer()
	fpsCap.Stop()
	if a.stopMon != nil {
		close(a.stopMon)
	}
}

func (a *App) InitError() string { return a.initEr }

func (a *App) GetSystemInfo() SysInfo { return systemInfo() }

func (a *App) GetLive() Live {
	l := liveUsage()
	enrichLive(&l)
	a.liveMu.Lock()
	a.lastLive, a.hasLive = l, true
	a.liveMu.Unlock()
	return l
}

// Detector de juegos
func (a *App) CurrentGame() *GameSession {
	if a.games == nil {
		return nil
	}
	return a.games.Current()
}

func (a *App) ListSessions() []GameSession {
	if a.games == nil {
		return nil
	}
	return a.games.Sessions()
}

func (a *App) ClearSessions() {
	if a.games != nil {
		a.games.Clear()
	}
}

func (a *App) ListTweaks() []TweakView {
	if a.engine == nil {
		return nil
	}
	return a.engine.List()
}

// Los métodos devuelven "" si salió bien o el mensaje de error.
func (a *App) ApplyTweak(id string) string {
	if a.engine == nil {
		return a.initEr
	}
	if err := a.engine.Apply(id); err != nil {
		return err.Error()
	}
	return ""
}

type OnceOutcome struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

// RunOnce ejecuta una acción única (botón Aplicar).
func (a *App) RunOnce(id string) OnceOutcome {
	if a.engine == nil {
		return OnceOutcome{Error: a.initEr}
	}
	msg, err := a.engine.RunOnce(id)
	if err != nil {
		return OnceOutcome{Error: err.Error()}
	}
	return OnceOutcome{Message: msg}
}

func (a *App) RevertTweak(id string) string {
	if a.engine == nil {
		return a.initEr
	}
	if err := a.engine.Revert(id); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) RevertAll() string {
	if a.engine == nil {
		return a.initEr
	}
	if err := a.engine.RevertAll(); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) ListProfiles() []Profile { return profiles() }

type ProfileOutcome struct {
	ProfileResult
	Error string `json:"error"`
}

func (a *App) ApplyProfile(id string) ProfileOutcome {
	if a.engine == nil {
		return ProfileOutcome{Error: a.initEr}
	}
	res, err := a.engine.ApplyProfile(id)
	out := ProfileOutcome{ProfileResult: res}
	if err != nil {
		out.Error = err.Error()
	}
	return out
}

func (a *App) CreateRestorePoint() string {
	if err := CreateRestorePoint(); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) ScanCleanup() []CleanItem { return ScanCleanup() }

type CleanOutcome struct {
	CleanResult
	Error string `json:"error"`
}

func (a *App) RunCleanup(ids []string) CleanOutcome {
	res, err := RunCleanup(ids)
	out := CleanOutcome{CleanResult: res}
	if err != nil {
		out.Error = err.Error()
	}
	return out
}

func (a *App) ListStartup() []StartupEntry { return ListStartup() }

func (a *App) DiscordState() DiscordUser           { return DiscordState() }
func (a *App) DiscordLogin() DiscordUser           { return DiscordLogin() }
func (a *App) DiscordLogout()                      { DiscordLogout() }
func (a *App) DiscordSetClientID(id string) string { return DiscordSetClientID(id) }

// AppIcons: ruta del programa -> icono (data URL PNG).
func (a *App) AppIcons(paths []string) map[string]string { return AppIcons(paths) }

// GameCover: portada del juego (data URL) o "" si no hay.
func (a *App) GameCover(exePath string) string { return GameCover(exePath) }

// GameLibrary: un juego por ejecutable, con los datos de su última sesión.
func (a *App) GameLibrary() []GameEntry {
	if a.games == nil {
		return nil
	}
	return gameLibrary(a.games.Sessions())
}

func (a *App) SetStartup(id string, enabled bool) string {
	if err := SetStartup(id, enabled); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) SetSimulation(v bool) {
	if a.engine != nil {
		a.engine.SetSimulation(v)
	}
}

func (a *App) GetSimulation() bool { return a.engine != nil && a.engine.Simulation() }

// RelaunchAsAdmin reabre la app con UAC y cierra esta instancia.
func (a *App) RelaunchAsAdmin() string {
	exe, err := os.Executable()
	if err != nil {
		return err.Error()
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	if err := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_NORMAL); err != nil {
		return "No se pudo pedir permisos de administrador: " + err.Error()
	}
	runtime.Quit(a.ctx)
	return ""
}
