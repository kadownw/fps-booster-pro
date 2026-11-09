package main

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
)

// Diagnóstico de cada ajuste: dice, con datos leídos del sistema en este momento, si el ajuste está
// aplicado, parcial o sin aplicar, o por qué no puede aplicarse en este equipo. Solo LEE: no cambia nada.

type Diagnosis struct {
	Status  string // applied | partial | not_applied | unavailable | once
	Detail  string
	Done    int
	Total   int
	Skipped int
}

// skippedChanges cuenta cambios sin comodines que se omiten porque el servicio o la clave no existe aquí.
func skippedChanges(t Tweak) int {
	if t.Vendor != "" {
		return 0
	}
	n := 0
	for _, c := range t.Changes {
		if strings.Contains(c.Path, nicPlaceholder) || strings.Contains(c.Path, dispPlaceholder) {
			continue
		}
		if c.MustExist && !keyExists(c) {
			n++
			continue
		}
		if c.IfExists {
			if st, err := readState(c); err != nil || !st.Exists {
				n++
			}
		}
	}
	return n
}

// whyUnsupported explica en lenguaje claro por qué un ajuste no aplica a este equipo.
func whyUnsupported(t Tweak) string {
	b := windowsBuild()
	if t.MinBuild > 0 && b < t.MinBuild {
		return fmt.Sprintf("Necesita Windows build %d o superior y este equipo tiene la %d.", t.MinBuild, b)
	}
	if t.MaxBuild > 0 && b > t.MaxBuild {
		return fmt.Sprintf("Solo existe hasta la build %d de Windows y este equipo tiene la %d.", t.MaxBuild, b)
	}
	switch {
	case len(t.Nvidia) > 0:
		return "No se encontró una GPU NVIDIA con el driver que ofrece la API de perfiles (nvapi64.dll)."
	case t.ID == "memory_compression_off":
		return "El servicio SysMain está deshabilitado y Windows no permite cambiar la compresión de memoria sin él (error 1058)."
	case t.Vendor != "":
		found := gpus()
		have := "no se detectó ninguna GPU"
		if len(found) > 0 {
			have = "se detectó: " + strings.Join(found, ", ")
		}
		return fmt.Sprintf("Es solo para GPU %s y en este equipo %s.", strings.ToUpper(t.Vendor), have)
	}
	return fmt.Sprintf("Este Windows no tiene ninguna de las claves, servicios o tareas que modifica este ajuste (se buscaron %d), así que no hay nada que cambiar.", len(t.Changes)+len(t.Tasks))
}

// diagnose calcula el estado detallado. active = resultado de isActive (ya calculado por el llamador).
func (e *Engine) diagnose(t Tweak, active bool) Diagnosis {
	if t.Once {
		return Diagnosis{Status: "once", Detail: "Acción única: se ejecuta cuando la pedís y el resultado se comprueba con Windows."}
	}
	if !supported(t) {
		return Diagnosis{Status: "unavailable", Detail: "No aplica a tu equipo. " + whyUnsupported(t)}
	}
	adm := ""
	if t.Admin && !isAdmin() && !e.sim {
		adm = " Para cambiarlo necesitás abrir South Tweaks como administrador."
	}
	reboot := ""
	if t.Reboot {
		reboot = " El cambio se nota después de reiniciar Windows."
	}
	yesno := func(on bool, yes, no string) Diagnosis {
		if on {
			return Diagnosis{Status: "applied", Detail: yes + reboot}
		}
		return Diagnosis{Status: "not_applied", Detail: no + adm}
	}
	switch {
	case e.sim:
		return yesno(active, "Aplicado en modo simulación (no se tocó tu Windows).", "No aplicado (modo simulación).")
	case t.Power:
		return yesno(active, "Aplicado: el plan de energía activo es el de este ajuste.", "No aplicado: el plan de energía activo es otro.")
	case len(t.Nvidia) > 0 && len(t.Changes) == 0:
		return yesno(active, fmt.Sprintf("Aplicado: los %d valores del perfil están guardados en el driver NVIDIA.", len(t.Nvidia)), fmt.Sprintf("No aplicado: el driver NVIDIA no tiene todavía los %d valores del perfil.", len(t.Nvidia)))
	case t.PS != nil, t.Cmd != nil:
		return yesno(active, "Aplicado: Windows informa el estado buscado.", "No aplicado: Windows informa un estado distinto al buscado.")
	}
	regs := expand(t)
	tasks := existingTasks(t)
	d := Diagnosis{Total: len(regs) + len(tasks), Skipped: skippedChanges(t)}
	var missing []string
	for _, c := range regs {
		st, err := readState(c)
		if err == nil && satisfied(c, st) {
			d.Done++
		} else {
			missing = append(missing, c.Name)
		}
	}
	for p, st := range tasks {
		if st == "Disabled" {
			d.Done++
		} else {
			missing = append(missing, "tarea "+p)
		}
	}
	if len(t.Nvidia) > 0 { // ajuste mixto: además de los valores de registro, el perfil del driver NVIDIA
		d.Total += len(t.Nvidia)
		if nvIsActive(t.Nvidia) {
			d.Done += len(t.Nvidia)
		} else {
			missing = append(missing, fmt.Sprintf("%d valores del perfil NVIDIA", len(t.Nvidia)))
		}
	}
	skip := ""
	if d.Skipped > 0 {
		if d.Skipped == 1 {
			skip = " (1 más no existe en tu Windows y se omite)"
		} else {
			skip = fmt.Sprintf(" (%d más no existen en tu Windows y se omiten)", d.Skipped)
		}
	}
	switch {
	case d.Total > 0 && d.Done == d.Total:
		d.Status, d.Detail = "applied", fmt.Sprintf("Aplicado y verificado: %s en su estado final%s.%s", ofValues(d.Done, d.Total), skip, reboot)
	case d.Done > 0:
		if len(missing) > 3 {
			missing = append(missing[:3], "…")
		}
		d.Status, d.Detail = "partial", fmt.Sprintf("Parcial: %s aplicados%s. Faltan: %s.%s", ofValues(d.Done, d.Total), skip, strings.Join(missing, ", "), adm)
	default:
		d.Status, d.Detail = "not_applied", fmt.Sprintf("No aplicado: %s está en su estado final%s.%s", noneOf(d.Total), skip, adm)
	}
	return d
}

// humanError traduce el error técnico del motor a una explicación entendible (con el detalle al final).
func humanError(t Tweak, err error) string {
	if err == nil {
		return ""
	}
	raw := err.Error()
	switch {
	case errors.Is(err, errNeedAdmin):
		return "Necesita permisos de administrador: abrí South Tweaks como administrador (botón «Reiniciar como administrador») y volvé a intentar. No se cambió nada."
	case errors.Is(err, errTier):
		return raw
	case strings.Contains(raw, "no aplica a tu equipo"):
		return "No aplica a tu equipo. " + whyUnsupported(t)
	case errors.Is(err, syscall.ERROR_ACCESS_DENIED) || strings.Contains(raw, "Access is denied") || strings.Contains(raw, "Acceso denegado"):
		return "Windows denegó el acceso: esa clave o servicio está protegida por el sistema (TrustedInstaller) o por un antivirus. Lo que se alcanzó a cambiar se revirtió. Detalle: " + raw
	case strings.Contains(raw, "verificación fallida") || strings.Contains(raw, "no se pudo verificar"):
		return "Se escribió el valor pero al volver a leerlo Windows devolvió otro: algo (otra herramienta, una política de grupo o el propio sistema) lo está sobrescribiendo. Se revirtió. Detalle: " + raw
	}
	return "No se pudo aplicar: " + raw
}

// ApplyReport aplica un ajuste y devuelve un mensaje basado en una lectura posterior del sistema.
func (e *Engine) ApplyReport(id string) (string, error) {
	t, ok := findTweak(id)
	if !ok {
		return "", fmt.Errorf("ajuste desconocido: %s", id)
	}
	if err := e.Apply(id); err != nil {
		return "", errors.New(humanError(t, err))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	d := e.diagnose(t, e.isActive(t))
	if d.Status != "applied" {
		return "", fmt.Errorf("Se ejecutó el cambio pero la comprobación posterior no coincide: %s", d.Detail)
	}
	return d.Detail, nil
}

// RevertReport revierte un ajuste y confirma que cada valor volvió a su original.
func (e *Engine) RevertReport(id string) (string, error) {
	t, _ := findTweak(id)
	e.mu.Lock()
	n := 0
	if bk, ok := e.st.Items[id]; ok {
		n = len(bk.Items) + len(bk.Tasks)
		if bk.PrevPower != "" || bk.Nvidia != nil || bk.PSOrig != nil || bk.CmdOrig != nil {
			n++
		}
	}
	e.mu.Unlock()
	if err := e.Revert(id); err != nil {
		return "", errors.New(humanError(t, err))
	}
	if e.Simulation() {
		return "Revertido en modo simulación.", nil
	}
	return fmt.Sprintf("Revertido y verificado: %d valor(es) volvieron a como estaban antes de aplicar.", n), nil
}

func ofValues(done, total int) string {
	if total == 1 {
		return "el único valor está"
	}
	return fmt.Sprintf("%d de %d valores están", done, total)
}

func noneOf(total int) string {
	if total == 1 {
		return "el único valor todavía no"
	}
	return fmt.Sprintf("ninguno de los %d valores", total)
}
