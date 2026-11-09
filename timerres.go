package main

import (
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// Resolución de temporizador alta mientras hay un juego abierto.
//
// Es la versión nativa de lo que hace el "servicio de resolución de temporizador" popular, pero sin
// compilar ni instalar ningún servicio: South Tweaks pide la resolución máxima solo mientras el
// detector ve un juego abierto y la libera al cerrarse. Con GlobalTimerResolutionRequests=1
// (ajuste "timer_resolution") Windows 10 2004+ y 11 aplican el pedido a todo el sistema; sin ese
// valor solo afectaría a este proceso. Mayor resolución = más despertares del procesador y algo más de
// consumo. Impacto en FPS: no medido.

var (
	ntdll               = syscall.NewLazyDLL("ntdll.dll")
	procNtSetTimerRes   = ntdll.NewProc("NtSetTimerResolution")
	procNtQueryTimerRes = ntdll.NewProc("NtQueryTimerResolution")
)

// timerLimits devuelve (peor, mejor) resolución en unidades de 100 ns; "mejor" es el valor numérico menor.
func timerLimits() (coarsest, finest uint32, ok bool) {
	var actual uint32
	r, _, _ := procNtQueryTimerRes.Call(uintptr(unsafe.Pointer(&coarsest)), uintptr(unsafe.Pointer(&finest)), uintptr(unsafe.Pointer(&actual)))
	return coarsest, finest, r == 0 && finest > 0
}

// setTimerBoost pide (true) o libera (false) la resolución máxima. Devuelve la resolución actual.
func setTimerBoost(on bool) (current uint32, ok bool) {
	_, finest, ok := timerLimits()
	if !ok {
		return 0, false
	}
	var set uintptr
	if on {
		set = 1
	}
	r, _, _ := procNtSetTimerRes.Call(uintptr(finest), set, uintptr(unsafe.Pointer(&current)))
	return current, r == 0
}

var (
	timerMu     sync.Mutex
	timerOn     bool
	timerFlagAt time.Time
	timerFlag   bool
)

// timerFeatureEnabled: el ajuste "timer_resolution" está aplicado (valor de registro), con caché corta.
func timerFeatureEnabled() bool {
	timerMu.Lock()
	defer timerMu.Unlock()
	if time.Since(timerFlagAt) < 5*time.Second {
		return timerFlag
	}
	timerFlagAt = time.Now()
	timerFlag = false
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\kernel`, registry.QUERY_VALUE); err == nil {
		if v, _, err := k.GetIntegerValue("GlobalTimerResolutionRequests"); err == nil && v == 1 {
			timerFlag = true
		}
		k.Close()
	}
	return timerFlag
}

// syncTimer se llama en cada paso del detector de juegos.
func syncTimer(gameRunning bool) {
	want := gameRunning && timerFeatureEnabled()
	timerMu.Lock()
	defer timerMu.Unlock()
	if want == timerOn {
		return
	}
	if _, ok := setTimerBoost(want); ok {
		timerOn = want
	}
}

func releaseTimer() {
	timerMu.Lock()
	defer timerMu.Unlock()
	if timerOn {
		setTimerBoost(false)
		timerOn = false
	}
}
