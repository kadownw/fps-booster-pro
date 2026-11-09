package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Tareas programadas: se consultan y cambian con PowerShell (nombres de estado independientes
// del idioma: Ready, Running, Disabled...). Los nombres se validan antes de entrar a un comando.

var taskPathRe = regexp.MustCompile(`^\\[A-Za-z0-9 ._()\-\\]+$`)

var (
	taskMu     sync.Mutex
	taskCache  map[string]string
	taskCached time.Time
)

func runPS(script string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func invalidateTasks() {
	taskMu.Lock()
	taskCache = nil
	taskMu.Unlock()
}

// taskStates devuelve ruta(en minúsculas) -> estado, con caché corta.
func taskStates() map[string]string {
	taskMu.Lock()
	defer taskMu.Unlock()
	if taskCache != nil && time.Since(taskCached) < 60*time.Second {
		return taskCache
	}
	m := map[string]string{}
	out, err := runPS(`Get-ScheduledTask | ForEach-Object { $_.TaskPath + $_.TaskName + '|' + $_.State }`)
	if err == nil {
		for _, ln := range strings.Split(out, "\n") {
			ln = strings.TrimSpace(ln)
			if i := strings.LastIndex(ln, "|"); i > 0 {
				m[strings.ToLower(ln[:i])] = ln[i+1:]
			}
		}
	}
	taskCache, taskCached = m, time.Now()
	return m
}

func existingTasks(t Tweak) map[string]string {
	if len(t.Tasks) == 0 {
		return nil
	}
	all := taskStates()
	out := map[string]string{}
	for _, p := range t.Tasks {
		if st, ok := all[strings.ToLower(p)]; ok {
			out[p] = st
		}
	}
	return out
}

func setTask(path string, enable bool) error {
	if !taskPathRe.MatchString(path) || strings.Contains(path, `\\`) {
		return fmt.Errorf("ruta de tarea inválida: %q", path)
	}
	i := strings.LastIndex(path, `\`)
	dir, name := path[:i+1], path[i+1:]
	verb := "Disable"
	if enable {
		verb = "Enable"
	}
	out, err := runPS(fmt.Sprintf(`%s-ScheduledTask -TaskPath '%s' -TaskName '%s' -ErrorAction Stop | Out-Null`, verb, dir, name))
	invalidateTasks()
	if err != nil {
		return fmt.Errorf("tarea %s: %s", path, strings.TrimSpace(out))
	}
	return nil
}
