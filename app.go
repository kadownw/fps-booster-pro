package main

import (
	"context"
	"os"
	"syscall"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

// App expone al frontend (Wails) solo lo necesario.
type App struct {
	ctx    context.Context
	engine *Engine
	initEr string
}

func NewApp() *App {
	e, err := NewEngine()
	a := &App{engine: e}
	go taskStates() // precalienta la consulta de tareas programadas (tarda ~2 s)
	if err != nil {
		a.initEr = err.Error()
	}
	return a
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) InitError() string { return a.initEr }

func (a *App) GetSystemInfo() SysInfo { return systemInfo() }

func (a *App) GetLive() Live { return liveUsage() }

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
