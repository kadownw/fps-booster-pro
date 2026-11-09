package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// RegState es el estado de un valor de registro (para backup y verificación).
type RegState struct {
	Exists bool   `json:"exists"`
	Kind   string `json:"kind,omitempty"`
	D      uint32 `json:"d,omitempty"`
	S      string `json:"s,omitempty"`
	Hex    string `json:"hex,omitempty"` // valores binarios (REG_BINARY), en hexadecimal
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

func keyExists(c RegChange) bool {
	root, err := rootKey(c.Root)
	if err != nil {
		return false
	}
	k, err := registry.OpenKey(root, c.Path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
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
	case registry.BINARY:
		v, _, err := k.GetBinaryValue(c.Name)
		if err != nil {
			return RegState{}, err
		}
		return RegState{Exists: true, Kind: "bin", Hex: hex.EncodeToString(v)}, nil
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
	switch st.Kind {
	case "dword":
		return k.SetDWordValue(c.Name, st.D)
	case "bin":
		b, err := hex.DecodeString(st.Hex)
		if err != nil {
			return fmt.Errorf("valor binario inválido: %w", err)
		}
		return k.SetBinaryValue(c.Name, b)
	}
	return k.SetStringValue(c.Name, st.S)
}

func desired(c RegChange) RegState {
	return RegState{Exists: true, Kind: c.Kind, D: c.D, S: c.S, Hex: strings.ToLower(c.Hex)}
}

func sameState(a, b RegState) bool {
	return a.Exists == b.Exists && a.Kind == b.Kind && a.D == b.D && a.S == b.S && strings.EqualFold(a.Hex, b.Hex)
}

// ---- valores compuestos (lista clave=valor;) y fechas ----

const isoUTC = "2006-01-02T15:04:05Z"

// kvParse separa "A=1;B=2;" en pares ordenados.
func kvParse(s string) (keys []string, vals map[string]string) {
	vals = map[string]string{}
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		k = strings.TrimSpace(k)
		if _, seen := vals[k]; !seen {
			keys = append(keys, k)
		}
		vals[k] = strings.TrimSpace(v)
	}
	return
}

func kvHas(cur, want string) bool {
	_, have := kvParse(cur)
	wk, wv := kvParse(want)
	for _, k := range wk {
		if have[k] != wv[k] {
			return false
		}
	}
	return len(wk) > 0
}

// mergeKV agrega o reemplaza las claves de want dentro de cur, conservando las demás.
func mergeKV(cur, want string) string {
	keys, vals := kvParse(cur)
	wk, wv := kvParse(want)
	for _, k := range wk {
		if _, ok := vals[k]; !ok {
			keys = append(keys, k)
		}
		vals[k] = wv[k]
	}
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k + "=" + vals[k] + ";")
	}
	return b.String()
}

// dateFor calcula la fecha de un valor dinámico: "now" o "+35d".
func dateFor(spec string, now time.Time) string {
	if strings.HasPrefix(spec, "+") && strings.HasSuffix(spec, "d") {
		var n int
		if _, err := fmt.Sscanf(spec, "+%dd", &n); err == nil {
			return now.UTC().AddDate(0, 0, n).Format(isoUTC)
		}
	}
	return now.UTC().Format(isoUTC)
}

// satisfied indica si el estado actual cumple lo que el cambio busca.
func satisfied(c RegChange, st RegState) bool {
	switch {
	case c.KV != "":
		return st.Exists && st.Kind == "sz" && kvHas(st.S, c.KV)
	case c.Date != "":
		t, err := time.Parse(isoUTC, st.S)
		if !st.Exists || err != nil {
			return false
		}
		if strings.HasPrefix(c.Date, "+") { // una pausa solo cuenta mientras no haya vencido
			return t.After(time.Now().UTC())
		}
		return true
	}
	return sameState(st, desired(c))
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
