package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/windows/registry"
)

// Programas de inicio. Windows guarda en "StartupApproved" si una entrada de Run está activa o
// no (es lo que usa el Administrador de tareas): el primer byte es 02 (activa) o 03 (desactivada).
// Cambiar ese valor es totalmente reversible y no borra la entrada original.

type startupSource struct {
	Scope    string // "user" | "machine" | "machine32"
	Root     registry.Key
	RunPath  string
	Approved string
	Admin    bool
	Label    string
}

const runBase = `Software\Microsoft\Windows\CurrentVersion`

var startupSources = []startupSource{
	{"user", registry.CURRENT_USER, runBase + `\Run`, runBase + `\Explorer\StartupApproved\Run`, false, "Usuario"},
	{"machine", registry.LOCAL_MACHINE, runBase + `\Run`, runBase + `\Explorer\StartupApproved\Run`, true, "Todos los usuarios"},
	{"machine32", registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, runBase + `\Explorer\StartupApproved\Run32`, true, "Todos (32 bits)"},
}

type StartupEntry struct {
	ID      string `json:"id"` // scope|nombre
	Name    string `json:"name"`
	Command string `json:"command"`
	Scope   string `json:"scope"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
	Admin   bool   `json:"admin"`
}

func findSource(scope string) (startupSource, bool) {
	for _, s := range startupSources {
		if s.Scope == scope {
			return s, true
		}
	}
	return startupSource{}, false
}

func approvedEnabled(s startupSource, name string) bool {
	k, err := registry.OpenKey(s.Root, s.Approved, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer k.Close()
	b, _, err := k.GetBinaryValue(name)
	if err != nil || len(b) == 0 {
		return true
	}
	return b[0]&1 == 0 // 02, 06 = activa; 03, 07 = desactivada
}

func ListStartup() []StartupEntry {
	var out []StartupEntry
	for _, s := range startupSources {
		k, err := registry.OpenKey(s.Root, s.RunPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(-1)
		for _, n := range names {
			cmd, _, err := k.GetStringValue(n)
			if err != nil || !utf8.ValidString(n) {
				continue
			}
			out = append(out, StartupEntry{ID: s.Scope + "|" + n, Name: n, Command: cmd, Scope: s.Scope,
				Label: s.Label, Enabled: approvedEnabled(s, n), Admin: s.Admin})
		}
		k.Close()
	}
	return out
}

func SetStartup(id string, enabled bool) error {
	scope, name, ok := strings.Cut(id, "|")
	if !ok || name == "" {
		return errors.New("entrada de inicio inválida")
	}
	s, ok := findSource(scope)
	if !ok {
		return fmt.Errorf("origen desconocido: %s", scope)
	}
	if s.Admin && !isAdmin() {
		return errNeedAdmin
	}
	// Solo se puede cambiar algo que realmente existe en Run.
	rk, err := registry.OpenKey(s.Root, s.RunPath, registry.QUERY_VALUE)
	if err != nil {
		return errors.New("no existe esa entrada de inicio")
	}
	_, _, err = rk.GetStringValue(name)
	rk.Close()
	if err != nil {
		return errors.New("no existe esa entrada de inicio")
	}
	k, _, err := registry.CreateKey(s.Root, s.Approved, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	val := make([]byte, 12)
	if enabled {
		val[0] = 0x02
	} else {
		val[0] = 0x03
		ft := uint64(time.Now().UnixNano()/100) + 116444736000000000 // Unix -> FILETIME
		for i := 0; i < 8; i++ {
			val[4+i] = byte(ft >> (8 * i))
		}
	}
	if err := k.SetBinaryValue(name, val); err != nil {
		return err
	}
	if approvedEnabled(s, name) != enabled {
		return errors.New("no se pudo verificar el cambio")
	}
	return nil
}
