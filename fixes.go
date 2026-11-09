package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
	winsvc "golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Fixes: arreglos para problemas comunes de Windows. Cada uno hace tres cosas, todas con datos leídos
// del sistema: (1) COMPRUEBA si el problema existe, (2) lo corrige y (3) vuelve a comprobar. Si el
// problema no existe, no toca nada. Lo que es configuración (servicios, políticas) pasa por el motor
// de ajustes, así que guarda el valor original y se puede deshacer; los pasos que no son
// configuración (arrancar un servicio, registrar una app) se informan por separado.

// ---------- utilidades de servicios (nativas, sin PowerShell) ----------

func regServiceStart(name string) (uint32, bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\`+name, registry.QUERY_VALUE)
	if err != nil {
		return 0, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("Start")
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

func serviceRunning(name string) (running, known bool) {
	m, err := mgr.Connect()
	if err != nil {
		return false, false
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return false, false
	}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return false, false
	}
	return st.State == winsvc.Running, true
}

// startServiceNow arranca un servicio y espera hasta 10 s a que quede corriendo.
func startServiceNow(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	if st, err := s.Query(); err == nil && st.State == winsvc.Running {
		return nil
	}
	if err := s.Start(); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		if st, err := s.Query(); err == nil && st.State == winsvc.Running {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("el servicio %s no llegó a estado «en ejecución» en 10 s", name)
}

func disabledServices(names ...string) []string {
	var out []string
	for _, n := range names {
		if st, ok := regServiceStart(n); ok && st == startDisabled {
			out = append(out, fmt.Sprintf("El servicio %s está deshabilitado.", n))
		}
	}
	return out
}

func presentPolicies(changes []RegChange) []string {
	var out []string
	for _, c := range changes {
		if !c.Delete {
			continue
		}
		if st, err := readState(c); err == nil && st.Exists {
			out = append(out, fmt.Sprintf("La política %s\\%s está activa.", lastPath(c.Path), c.Name))
		}
	}
	return out
}

func lastPath(p string) string {
	if i := strings.LastIndex(p, `\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// svcFix: pone un servicio en el inicio indicado SOLO si hoy está deshabilitado.
func svcFix(name string, to uint32) RegChange {
	c := svc(name, to)
	d := uint32(startDisabled)
	c.IfCurrent = &d
	return c
}

func svcFixes(to uint32, names ...string) []RegChange {
	out := make([]RegChange, 0, len(names))
	for _, n := range names {
		out = append(out, svcFix(n, to))
	}
	return out
}

// policyGone: la política debe NO existir (se borra solo si está).
func policyGone(path, name string) RegChange {
	c := dw("HKLM", path, name, 0)
	c.Delete = true
	c.IfExists = true
	return c
}

// ---------- PowerShell de solo lectura / comandos fijos ----------

func fixPS(script string) (string, error) { return runPSEnv(script) }

// ---------- catálogo de arreglos ----------

const (
	wuPol   = `SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate`
	storePo = `SOFTWARE\Policies\Microsoft\WindowsStore`
	defPol  = `SOFTWARE\Policies\Microsoft\Windows Defender`
	hvciKey = `SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity`
	dgKey   = `SYSTEM\CurrentControlSet\Control\DeviceGuard`
)

var storeServices = []string{"wuauserv", "UsoSvc", "BITS", "WaaSMedicSvc", "ClipSVC", "InstallService", "DoSvc", "AppXSvc",
	"TokenBroker", "wlidsvc", "LicenseManager", "XblAuthManager", "XblGameSave", "XboxGipSvc", "XboxNetApiSvc", "GamingServices", "GamingServicesNet"}

func storePolicies() []RegChange {
	return []RegChange{
		policyGone(wuPol+`\AU`, "NoAutoUpdate"), policyGone(wuPol, "DisableWindowsUpdateAccess"),
		policyGone(wuPol, "DoNotConnectToWindowsUpdateInternetLocations"), policyGone(wuPol, "SetDisableUXWUAccess"),
		policyGone(storePo, "RemoveWindowsStore"), policyGone(storePo, "DisableStoreApps"),
	}
}

func defenderPolicies() []RegChange {
	rt := defPol + `\Real-Time Protection`
	return []RegChange{
		policyGone(defPol, "DisableAntiSpyware"), policyGone(defPol, "DisableAntiVirus"), policyGone(defPol, "DisableRoutinelyTakingAction"),
		policyGone(rt, "DisableRealtimeMonitoring"), policyGone(rt, "DisableBehaviorMonitoring"), policyGone(rt, "DisableOnAccessProtection"),
		policyGone(rt, "DisableScanOnRealtimeEnable"), policyGone(rt, "DisableIOAVProtection"),
		policyGone(`SOFTWARE\Microsoft\Windows Defender`, "DisableAntiSpyware"),
	}
}

func fixesCatalog() []Tweak {
	storeChanges := append(svcFixes(startManual, storeServices...), storePolicies()...)
	defChanges := append(append(svcFixes(startAuto, "WinDefend"), svcFixes(startManual, "WdNisSvc")...), defenderPolicies()...)
	return []Tweak{
		{
			ID: "fix_valorant_cfg", Title: "Valorant: arreglar Control Flow Guard (CFG)", Category: "Fixes", Icon: "valorant", Admin: true, Fix: true,
			Description: "Activa Control Flow Guard y sus mitigaciones relacionadas en todo el sistema, que Vanguard (el anticheat de Valorant) exige. Comprueba el estado real antes y después.",
		},
		{
			ID: "fix_valorant_hvci", Title: "Valorant: activar Integridad de memoria (HVCI)", Category: "Fixes", Icon: "valorant", Admin: true, Reboot: true, Fix: true,
			Description: "Activa la Integridad de memoria (HVCI), que Vanguard y FACEIT exigen. Necesita reiniciar y Windows la deja apagada si detecta drivers incompatibles. Tiene un costo de rendimiento chico y variable (las pruebas publicadas van de ~3 % a ~8 % en juegos limitados por CPU).",
			Changes:     []RegChange{dw("HKLM", hvciKey, "Enabled", 1), dw("HKLM", dgKey, "EnableVirtualizationBasedSecurity", 1), dw("HKLM", dgKey, "RequirePlatformSecurityFeatures", 1)},
		},
		{
			ID: "fix_nvidia_panel", Title: "Panel de control de NVIDIA (sin Microsoft Store)", Category: "Fixes", Icon: "nvidia", Fix: true,
			Description: "Abre el Panel de control de NVIDIA. Si no está instalado, busca el paquete dentro del driver y lo registra sin pasar por la Tienda; si tu driver no lo trae, te lo dice (en ese caso se instala desde la Tienda o con el instalador completo del driver).",
		},
		{
			ID: "fix_wifi", Title: "Wi-Fi: servicios de red inalámbrica", Category: "Fixes", Icon: "wifi", Admin: true, Reboot: false, Fix: true,
			Description: "Vuelve a poner en inicio automático WLAN AutoConfig y WinHTTP Web Proxy Auto-Discovery si estaban deshabilitados, y los arranca. Sin esos servicios el Wi-Fi no conecta.",
			Changes:     svcFixes(startAuto, "WlanSvc", "WinHttpAutoProxySvc"),
		},
		{
			ID: "fix_bluetooth", Title: "Bluetooth: servicios de emparejamiento", Category: "Fixes", Icon: "bluetooth", Admin: true, Fix: true,
			Description: "Pone en su inicio normal (manual) los servicios de Bluetooth que estén deshabilitados y arranca el principal. Arregla emparejamientos y conexiones que no aparecen.",
			Changes:     svcFixes(startManual, "bthserv", "BTAGService", "BthAvctpSvc", "BluetoothUserService", "DeviceAssociationService"),
		},
		{
			ID: "fix_installer_2502", Title: "Error 2502/2503 del instalador de Windows", Category: "Fixes", Icon: "installer", Admin: true, Fix: true,
			Description: "Crea la carpeta Temp si falta y devuelve los permisos que necesita el instalador (SYSTEM y Administradores con control total, Usuarios con permiso para crear archivos) en C:\\Windows\\Temp.",
		},
		{
			ID: "fix_store_update", Title: "Tienda, Windows Update y Xbox", Category: "Fixes", Icon: "store", Admin: true, Fix: true,
			Description: "Vuelve a poner en inicio manual los servicios de la Tienda, Windows Update y Xbox que estén deshabilitados y quita las políticas que bloquean Windows Update o la Tienda. Arregla errores como 0x80070422 o descargas que no empiezan. Solo toca lo que está mal.",
			Changes:     storeChanges,
		},
		{
			ID: "fix_power_sleep", Title: "Energía y suspensión (Configuración no abre)", Category: "Fixes", Icon: "moon", Admin: true, Fix: true,
			Description: "Arregla que la configuración de energía y suspensión no se abra cuando el servicio de sensores está deshabilitado: lo vuelve a poner en manual.",
			Changes:     svcFixes(startManual, "SensorService", "SensrSvc", "SensorDataService"),
		},
		{
			ID: "fix_defender", Title: "Seguridad de Windows en blanco / Defender", Category: "Fixes", Icon: "shield", Admin: true, Fix: true,
			Description: "Repara la pantalla de Seguridad de Windows que sale en blanco o no funciona: vuelve a poner en su inicio los servicios de Defender, quita políticas que lo desactivan y vuelve a registrar la app de Seguridad de Windows. Con la Protección contra alteraciones activada, Windows puede rechazar algunos cambios y la app te lo dice.",
			Changes:     defChanges,
		},
	}
}

// ---------- comprobaciones y pasos posteriores de cada arreglo ----------

type fixSpec struct {
	Check   func() []string // problemas detectados; vacío = no hay nada que arreglar
	Post    func() []string // pasos que no son configuración (líneas con ✓ / ✗)
	Always  bool            // se ejecuta aunque no haya «problema» (p. ej. abrir el panel de NVIDIA)
	Healthy string          // texto cuando no hay problema
}

func okLine(s string) string  { return "✓ " + s }
func badLine(s string) string { return "✗ " + s }

var fixSpecs = map[string]fixSpec{
	"fix_valorant_cfg": {
		Healthy: "Control Flow Guard ya está activo en el sistema.",
		Check: func() []string {
			out, err := fixPS(`$ErrorActionPreference='Stop'; try { (Get-ProcessMitigation -System).CFG.Enable } catch { 'ERROR|' + $_.Exception.Message; exit 3 }`)
			switch {
			case err != nil:
				return []string{"No se pudo leer la mitigación CFG: " + strings.TrimPrefix(strings.TrimSpace(out), "ERROR|")}
			case !strings.EqualFold(strings.TrimSpace(out), "ON"):
				return []string{fmt.Sprintf("Control Flow Guard está en «%s» y Vanguard lo necesita en ON.", strings.TrimSpace(out))}
			}
			return nil
		},
		Post: func() []string {
			if out, err := fixPS(`Set-ProcessMitigation -System -Enable CFG`); err != nil {
				return []string{badLine("Set-ProcessMitigation falló: " + strings.TrimSpace(out))}
			}
			return []string{okLine("Se pidió activar CFG en todo el sistema.")}
		},
	},
	"fix_valorant_hvci": {
		Healthy: "La Integridad de memoria ya está configurada y activa.",
		Check: func() []string {
			k, err := registry.OpenKey(registry.LOCAL_MACHINE, hvciKey, registry.QUERY_VALUE)
			if err == nil {
				defer k.Close()
				if v, _, err := k.GetIntegerValue("Enabled"); err == nil && v == 1 {
					return nil
				}
			}
			return []string{"La Integridad de memoria (HVCI) no está activada en la configuración."}
		},
		Post: func() []string {
			out, _ := fixPS(`(Get-CimInstance -ClassName Win32_DeviceGuard -Namespace root\Microsoft\Windows\DeviceGuard).SecurityServicesRunning -join ','`)
			for _, s := range strings.Split(strings.TrimSpace(out), ",") {
				if strings.TrimSpace(s) == "2" {
					return []string{okLine("Windows informa HVCI en ejecución.")}
				}
			}
			return []string{okLine("Quedó configurada, pero todavía no está en ejecución: reiniciá Windows. Si después sigue apagada, Windows detectó drivers incompatibles (Seguridad de Windows → Aislamiento del núcleo te dice cuáles).")}
		},
	},
	"fix_nvidia_panel": {
		Always: true,
		Check:  func() []string { return nil },
		Post: func() []string {
			out, _ := fixPS(`(Get-AppxPackage -AllUsers -Name 'NVIDIACorp.NVIDIAControlPanel' | Select-Object -First 1).PackageFamilyName`)
			fam := strings.TrimSpace(out)
			if fam == "" {
				// Sin la app registrada: se busca el paquete dentro del driver (solo en la carpeta de drivers de Windows).
				manifest := findNvidiaPanelManifest()
				if manifest == "" {
					return []string{badLine("El Panel de control de NVIDIA no está instalado y tu driver no incluye el paquete para registrarlo. Instalalo desde la Microsoft Store (Panel de control de NVIDIA) o con el instalador completo del driver.")}
				}
				if out, err := runPSEnv(`Add-AppxPackage -DisableDevelopmentMode -Register $env:ST_MANIFEST`, "ST_MANIFEST="+manifest); err != nil {
					return []string{badLine("No se pudo registrar el paquete del driver: " + strings.TrimSpace(out))}
				}
				out, _ = fixPS(`(Get-AppxPackage -AllUsers -Name 'NVIDIACorp.NVIDIAControlPanel' | Select-Object -First 1).PackageFamilyName`)
				if fam = strings.TrimSpace(out); fam == "" {
					return []string{badLine("Se intentó registrar el paquete pero Windows no lo muestra como instalado.")}
				}
			}
			cmd := exec.Command("explorer.exe", `shell:AppsFolder\`+fam+`!NVIDIACorp.NVIDIAControlPanel`)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if err := cmd.Start(); err != nil {
				return []string{badLine("No se pudo abrir el Panel de control de NVIDIA: " + err.Error())}
			}
			for i := 0; i < 16; i++ {
				time.Sleep(500 * time.Millisecond)
				if o, _ := fixPS(`(Get-Process -Name nvcplui -ErrorAction SilentlyContinue | Measure-Object).Count`); strings.TrimSpace(o) != "0" && strings.TrimSpace(o) != "" {
					return []string{okLine("El Panel de control de NVIDIA se abrió (proceso nvcplui en ejecución).")}
				}
			}
			return []string{badLine("Se pidió abrir el Panel de control de NVIDIA pero el proceso no apareció en 8 s.")}
		},
	},
	"fix_wifi": {
		Healthy: "WLAN AutoConfig y WinHTTP Auto-Proxy están habilitados y en ejecución.",
		Check: func() []string {
			out := disabledServices("WlanSvc", "WinHttpAutoProxySvc")
			for _, n := range []string{"WlanSvc", "WinHttpAutoProxySvc"} {
				if run, known := serviceRunning(n); known && !run && len(disabledServices(n)) == 0 {
					out = append(out, fmt.Sprintf("El servicio %s no está en ejecución.", n))
				}
			}
			return out
		},
		Post: func() []string {
			var lines []string
			for _, n := range []string{"WlanSvc", "WinHttpAutoProxySvc"} {
				if st, ok := regServiceStart(n); ok && st == startDisabled {
					lines = append(lines, badLine(n+" sigue deshabilitado."))
					continue
				}
				if err := startServiceNow(n); err != nil {
					hint := ""
					if n == "WlanSvc" {
						hint = " (si tu equipo no tiene adaptador Wi-Fi, Windows no lo deja arrancar)"
					}
					lines = append(lines, badLine(fmt.Sprintf("%s no pudo arrancar: %v%s", n, err, hint)))
				} else {
					lines = append(lines, okLine(n+" en ejecución."))
				}
			}
			return lines
		},
	},
	"fix_bluetooth": {
		Healthy: "Los servicios de Bluetooth no están deshabilitados.",
		Check: func() []string {
			return disabledServices("bthserv", "BTAGService", "BthAvctpSvc", "BluetoothUserService", "DeviceAssociationService")
		},
		Post: func() []string {
			if err := startServiceNow("bthserv"); err != nil {
				return []string{badLine(fmt.Sprintf("El servicio de soporte Bluetooth no pudo arrancar: %v (si no hay adaptador Bluetooth, es normal)", err))}
			}
			return []string{okLine("Servicio de soporte Bluetooth en ejecución.")}
		},
	},
	"fix_installer_2502": {
		Healthy: "La carpeta Temp existe y tiene los permisos que necesita el instalador.",
		Check:   checkTempAcl,
		Post: func() []string {
			win := os.Getenv("SystemRoot")
			if win == "" {
				return []string{badLine("No se pudo leer la carpeta de Windows (SystemRoot).")}
			}
			temp := filepath.Join(win, "Temp")
			var lines []string
			if err := os.MkdirAll(temp, 0o777); err != nil {
				return []string{badLine("No se pudo crear " + temp + ": " + err.Error())}
			}
			// SID fijos (independientes del idioma): SYSTEM, Administradores, Usuarios.
			args := []string{temp, "/grant", "*S-1-5-18:(OI)(CI)F", "*S-1-5-32-544:(OI)(CI)F", "*S-1-5-32-545:(CI)(S,WD,AD,X)"}
			if out, err := runExe("icacls.exe", args...); err != nil {
				lines = append(lines, badLine("icacls falló: "+out))
			} else {
				lines = append(lines, okLine("Permisos de "+temp+" corregidos con icacls."))
			}
			if ut := os.TempDir(); ut != "" {
				if err := os.MkdirAll(ut, 0o700); err != nil {
					lines = append(lines, badLine("No se pudo crear la carpeta Temp del usuario: "+err.Error()))
				}
			}
			return lines
		},
	},
	"fix_store_update": {
		Healthy: "No hay servicios de la Tienda/Windows Update/Xbox deshabilitados ni políticas que los bloqueen.",
		Check: func() []string {
			return append(disabledServices(storeServices...), presentPolicies(storePolicies())...)
		},
	},
	"fix_power_sleep": {
		Healthy: "Los servicios de sensores no están deshabilitados.",
		Check:   func() []string { return disabledServices("SensorService", "SensrSvc", "SensorDataService") },
	},
	"fix_defender": {
		Healthy: "Los servicios y políticas de Defender están bien y la app de Seguridad de Windows está registrada.",
		Check: func() []string {
			out := append(disabledServices("WinDefend", "WdNisSvc"), presentPolicies(defenderPolicies())...)
			if o, err := fixPS(`(Get-AppxPackage -AllUsers -Name 'Microsoft.SecHealthUI' | Measure-Object).Count`); err == nil && strings.TrimSpace(o) == "0" {
				out = append(out, "La app Seguridad de Windows (SecHealthUI) no está registrada: por eso la pantalla sale en blanco.")
			}
			return out
		},
		Post: func() []string {
			var lines []string
			manifest := filepath.Join(os.Getenv("SystemRoot"), "SystemApps", "Microsoft.Windows.SecHealthUI_cw5n1h2txyewy", "AppXManifest.xml")
			if _, err := os.Stat(manifest); err == nil {
				if out, err := runPSEnv(`Add-AppxPackage -DisableDevelopmentMode -Register $env:ST_MANIFEST`, "ST_MANIFEST="+manifest); err != nil {
					lines = append(lines, badLine("No se pudo registrar la app Seguridad de Windows: "+strings.TrimSpace(out)))
				} else if o, _ := fixPS(`(Get-AppxPackage -AllUsers -Name 'Microsoft.SecHealthUI' | Measure-Object).Count`); strings.TrimSpace(o) != "0" {
					lines = append(lines, okLine("La app Seguridad de Windows quedó registrada."))
				} else {
					lines = append(lines, badLine("Se registró la app pero Windows no la muestra como instalada."))
				}
			} else {
				lines = append(lines, badLine("No está el archivo de la app Seguridad de Windows en SystemApps; no se pudo registrar."))
			}
			if err := startServiceNow("WinDefend"); err != nil {
				lines = append(lines, badLine("El servicio de Defender no arrancó: "+err.Error()+". Puede ser que otro antivirus esté instalado o que la Protección contra alteraciones lo bloquee."))
			} else {
				lines = append(lines, okLine("Servicio de Defender en ejecución."))
			}
			return lines
		},
	},
}

func checkTempAcl() []string {
	out, err := fixPS(`$p = Join-Path $env:SystemRoot 'Temp'
if (-not (Test-Path $p)) { 'MISSING_DIR'; exit 0 }
$sids = (Get-Acl $p).Access | ForEach-Object { try { $_.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value } catch {} }
foreach ($s in 'S-1-5-18','S-1-5-32-544','S-1-5-32-545') { if ($sids -notcontains $s) { 'NOACL|' + $s } }`)
	if err != nil {
		return []string{"No se pudo leer los permisos de C:\\Windows\\Temp: " + strings.TrimSpace(out)}
	}
	names := map[string]string{"S-1-5-18": "SYSTEM", "S-1-5-32-544": "Administradores", "S-1-5-32-545": "Usuarios"}
	var probs []string
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		switch {
		case l == "MISSING_DIR":
			probs = append(probs, "La carpeta C:\\Windows\\Temp no existe.")
		case strings.HasPrefix(l, "NOACL|"):
			probs = append(probs, fmt.Sprintf("Falta el permiso de %s en C:\\Windows\\Temp.", names[strings.TrimPrefix(l, "NOACL|")]))
		}
	}
	return probs
}

// findNvidiaPanelManifest busca el paquete del Panel de control de NVIDIA dentro del almacén de drivers.
func findNvidiaPanelManifest() string {
	root := filepath.Join(os.Getenv("SystemRoot"), "System32", "DriverStore", "FileRepository")
	dirs, _ := filepath.Glob(filepath.Join(root, "nv_disp*"))
	for _, d := range dirs {
		for _, pat := range []string{"NVCPL", "NvCpl", "NVIDIA Control Panel"} {
			if m, _ := filepath.Glob(filepath.Join(d, pat, "AppxManifest.xml")); len(m) > 0 {
				return m[0]
			}
			if m, _ := filepath.Glob(filepath.Join(d, pat, "*", "AppxManifest.xml")); len(m) > 0 {
				return m[0]
			}
		}
	}
	return ""
}

// ---------- API para la interfaz ----------

type FixView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Admin       bool   `json:"admin"`
	Reboot      bool   `json:"reboot"`
	Risk        string `json:"risk"`
	LastRun     string `json:"lastRun,omitempty"`
	CanUndo     bool   `json:"canUndo"`
}

func (e *Engine) ListFixes() []FixView {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []FixView
	for _, t := range catalog() {
		if !t.Fix {
			continue
		}
		_, has := e.st.Items[t.ID]
		out = append(out, FixView{ID: t.ID, Title: t.Title, Description: t.Description, Icon: t.Icon, Admin: t.Admin, Reboot: t.Reboot,
			Risk: t.Risk, LastRun: e.runs[t.ID], CanUndo: has})
	}
	return out
}

type FixCheckResult struct {
	Problems []string `json:"problems"`
	Healthy  string   `json:"healthy"`
	Always   bool     `json:"always"`
	Error    string   `json:"error"`
}

// FixCheck lee el sistema y dice qué problemas encontró (sin cambiar nada).
func (e *Engine) FixCheck(id string) FixCheckResult {
	t, ok := findTweak(id)
	spec, has := fixSpecs[id]
	if !ok || !t.Fix || !has {
		return FixCheckResult{Error: "arreglo desconocido: " + id}
	}
	if spec.Check == nil {
		return FixCheckResult{Healthy: spec.Healthy, Always: spec.Always}
	}
	return FixCheckResult{Problems: spec.Check(), Healthy: spec.Healthy, Always: spec.Always}
}

type FixOutcome struct {
	Message string   `json:"message"`
	Error   string   `json:"error"`
	Lines   []string `json:"lines"`
	Noop    bool     `json:"noop"`
	Reboot  bool     `json:"reboot"`
}

// RunFix comprueba, corrige y vuelve a comprobar. Informa solo lo que Windows confirma.
func (e *Engine) RunFix(id string) (out FixOutcome) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var err error
	defer func() {
		if out.Error != "" {
			err = fmt.Errorf("%s", out.Error)
		}
		e.log("fix", id, err)
	}()
	t, ok := findTweak(id)
	spec, has := fixSpecs[id]
	if !ok || !t.Fix || !has {
		return FixOutcome{Error: "arreglo desconocido: " + id}
	}
	if e.sim {
		return FixOutcome{Message: "Simulación: no se ejecutó nada.", Noop: true}
	}
	if t.Admin && !isAdmin() {
		return FixOutcome{Error: humanError(t, errNeedAdmin)}
	}
	var problems []string
	if spec.Check != nil {
		problems = spec.Check()
	}
	if len(problems) == 0 && !spec.Always {
		return FixOutcome{Message: "No se encontró el problema. " + spec.Healthy + " No se cambió nada.", Noop: true}
	}
	var lines []string
	for _, p := range problems {
		lines = append(lines, "Encontrado: "+p)
	}
	if regs := expand(t); len(regs) > 0 {
		if err := e.applyCore(t); err != nil {
			return FixOutcome{Error: humanError(t, err), Lines: lines}
		}
		lines = append(lines, okLine(fmt.Sprintf("%d valor(es) de configuración corregidos y verificados (se pueden deshacer).", len(regs))))
	}
	if spec.Post != nil {
		lines = append(lines, spec.Post()...)
	}
	var after []string
	if spec.Check != nil && !spec.Always {
		after = spec.Check()
	}
	e.runs[id] = time.Now().Format(time.RFC3339)
	e.saveRuns()
	failed := false
	for _, l := range lines {
		if strings.HasPrefix(l, "✗") {
			failed = true
		}
	}
	if len(after) > 0 || failed {
		msg := "No se pudo arreglar todo."
		if len(after) > 0 {
			msg += " Después del arreglo Windows sigue informando: " + strings.Join(after, " ")
		}
		return FixOutcome{Error: msg, Lines: lines, Reboot: t.Reboot}
	}
	msg := "Arreglado y verificado con Windows."
	if spec.Always {
		msg = "Listo."
	}
	return FixOutcome{Message: msg, Lines: lines, Reboot: t.Reboot}
}
