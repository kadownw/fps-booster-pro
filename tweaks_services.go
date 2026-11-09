package main

// Servicios: el tipo de inicio vive en HKLM\SYSTEM\CurrentControlSet\Services\<nombre>\Start
// (2 = automático, 3 = manual, 4 = deshabilitado). Se cambia ahí, con backup y rollback como
// cualquier valor de registro, y solo si el servicio existe en este equipo (MustExist).
// El cambio se nota al reiniciar.

const (
	startAuto     = 2
	startManual   = 3
	startDisabled = 4
)

func svc(name string, start uint32) RegChange {
	c := dw("HKLM", `SYSTEM\CurrentControlSet\Services\`+name, "Start", start)
	c.MustExist = true
	return c
}

func svcs(start uint32, names ...string) []RegChange {
	out := make([]RegChange, 0, len(names))
	for _, n := range names {
		out = append(out, svc(n, start))
	}
	return out
}

// loggerOff desactiva un logger de diagnóstico de arranque (solo si existe).
func loggerOff(name string) RegChange {
	c := dw("HKLM", `SYSTEM\CurrentControlSet\Control\WMI\Autologger\`+name, "Start", 0)
	c.MustExist = true
	return c
}

// bootLoggers: registradores de diagnóstico de arranque (se apagan junto con la telemetría).
func bootLoggers() []RegChange {
	var out []RegChange
	for _, n := range []string{"AppModel", "Cellcore", "CloudExperienceHostOobe", "DataMarket", "DiagLog",
		"Diagtrack-Listener", "LwtNetLog", "SQMLogger", "WdiContextLog", "WiFiSession"} {
		out = append(out, loggerOff(n))
	}
	return out
}

func servicesCatalog() []Tweak {
	return []Tweak{
		{
			ID: "svc_xbox_manual", Title: "Servicios de Xbox: inicio manual", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Los servicios de autenticación, partidas guardadas y accesorios de Xbox dejan de arrancar con Windows y se inician cuando un juego o la app de Xbox los necesita.",
			Changes:     svcs(startManual, "XblAuthManager", "XblGameSave", "XboxGipSvc", "XboxNetApiSvc"),
		},
		{
			ID: "fast_startup", Title: "Desactivar Inicio rápido", Category: "Energía", Admin: true, Reboot: true,
			Description: "Apaga el Inicio rápido: el apagado es completo y los cambios de drivers y configuración se aplican de verdad al reiniciar. El arranque puede tardar algo más.",
			Changes:     []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\Power`, "HiberbootEnabled", 0)},
		},
		{
			ID: "usb_recovery", Title: "USB: sin recuperación por caída de energía", Category: "Energía", Admin: true,
			Description: "Evita que Windows desconecte y vuelva a conectar dispositivos USB cuando detecta una caída de energía. Útil si tu mouse o teclado se corta de golpe.",
			Changes:     []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\USB\AutomaticSurpriseRemoval`, "AttemptRecoveryFromUsbPowerDrain", 0)},
		},
		{
			ID: "svchost_split", Title: "Consolidar procesos de servicios (svchost)", Category: "Sistema", Admin: true, Reboot: true,
			Description: "Agrupa los servicios en menos procesos svchost.exe, lo que reduce procesos y memoria cuando tenés bastante RAM. El umbral se calcula con tu RAM instalada.",
			Changes:     []RegChange{{Root: "HKLM", Path: `SYSTEM\CurrentControlSet\Control`, Name: "SvcHostSplitThresholdInKB", Kind: "dword", RAMKB: true}},
		},
		{
			ID: "webdav_limit", Title: "Quitar límite de tamaño de archivos WebDAV", Category: "Red", Admin: true,
			Description: "Elimina el límite de 50 MB al copiar archivos desde unidades de red WebDAV.",
			Changes: []RegChange{{Root: "HKLM", Path: `SYSTEM\CurrentControlSet\Services\WebClient\Parameters`, Name: "FileSizeLimitInBytes",
				Kind: "dword", D: 0xffffffff, MustExist: true}},
		},
	}
}
