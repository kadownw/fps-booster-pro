package main

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// RegState es el estado de un valor de registro (para backup y verificación).
type RegState struct {
	Exists bool   `json:"exists"`
	Kind   string `json:"kind,omitempty"`
	D      uint32 `json:"d,omitempty"`
	S      string `json:"s,omitempty"`
}

func rootKey(r string) (registry.Key, error) {
	switch r {
	case "HKCU":
		return registry.CURRENT_USER, nil
	case "HKLM":
		return registry.LOCAL_MACHINE, nil
	}
	return 0, fmt.Errorf("raíz de registro no permitida: %q", r)
}

func readState(c RegChange) (RegState, error) {
	root, err := rootKey(c.Root)
	if err != nil {
		return RegState{}, err
	}
	k, err := registry.OpenKey(root, c.Path, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return RegState{}, nil
		}
		return RegState{}, err
	}
	defer k.Close()
	_, typ, err := k.GetValue(c.Name, nil)
	if err != nil && !errors.Is(err, registry.ErrShortBuffer) {
		if errors.Is(err, registry.ErrNotExist) {
			return RegState{}, nil
		}
		return RegState{}, err
	}
	switch typ {
	case registry.DWORD:
		v, _, err := k.GetIntegerValue(c.Name)
		if err != nil {
			return RegState{}, err
		}
		return RegState{Exists: true, Kind: "dword", D: uint32(v)}, nil
	case registry.SZ, registry.EXPAND_SZ:
		v, _, err := k.GetStringValue(c.Name)
		if err != nil {
			return RegState{}, err
		}
		return RegState{Exists: true, Kind: "sz", S: v}, nil
	}
	return RegState{}, fmt.Errorf("%s\\%s\\%s: tipo de valor no soportado (%d); no se toca", c.Root, c.Path, c.Name, typ)
}

func writeState(c RegChange, st RegState) error {
	root, err := rootKey(c.Root)
	if err != nil {
		return err
	}
	if !st.Exists {
		k, err := registry.OpenKey(root, c.Path, registry.SET_VALUE)
		if err != nil {
			if errors.Is(err, registry.ErrNotExist) {
				return nil
			}
			return err
		}
		defer k.Close()
		if err := k.DeleteValue(c.Name); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	k, _, err := registry.CreateKey(root, c.Path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if st.Kind == "dword" {
		return k.SetDWordValue(c.Name, st.D)
	}
	return k.SetStringValue(c.Name, st.S)
}

func desired(c RegChange) RegState {
	return RegState{Exists: true, Kind: c.Kind, D: c.D, S: c.S}
}

func sameState(a, b RegState) bool {
	return a.Exists == b.Exists && a.Kind == b.Kind && a.D == b.D && a.S == b.S
}

// ---- powercfg ----

var guidRe = regexp.MustCompile(`[0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}`)

func powercfg(args ...string) (string, error) {
	cmd := exec.Command("powercfg", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func activeScheme() (string, error) {
	out, err := powercfg("/getactivescheme")
	if err != nil {
		return "", fmt.Errorf("powercfg: %w", err)
	}
	g := guidRe.FindString(out)
	if g == "" {
		return "", errors.New("no se pudo leer el plan de energía activo")
	}
	return strings.ToLower(g), nil
}

func setScheme(guid string) error {
	if !guidRe.MatchString(guid) || len(guid) != 36 { // evita inyección de argumentos
		return fmt.Errorf("GUID inválido: %q", guid)
	}
	if out, err := powercfg("/setactive", guid); err != nil {
		return fmt.Errorf("powercfg /setactive: %v %s", err, strings.TrimSpace(out))
	}
	return nil
}

func isAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
