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

func servicesCatalog() []Tweak {
	autologgers := []string{"AppModel", "Cellcore", "CloudExperienceHostOobe", "DataMarket", "DiagLog",
		"Diagtrack-Listener", "LwtNetLog", "SQMLogger", "WdiContextLog", "WiFiSession"}
	var loggers []RegChange
	for _, n := range autologgers {
		loggers = append(loggers, loggerOff(n))
	}

	return []Tweak{
		{
			ID: "svc_unneeded", Title: "Deshabilitar servicios que casi nadie usa", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Pone en «Deshabilitado» servicios como Fax, Registro remoto, Acceso remoto, Router AllJoyn, App-V, Demo de tienda, Teclado de acceso asignado y Puerto compartido de red TCP. Solo se tocan los que existen en tu Windows.",
			Changes: svcs(startDisabled, "AJRouter", "AppVClient", "AssignedAccessManagerSvc", "DialogBlockingService",
				"Fax", "NetTcpPortSharing", "RemoteAccess", "RemoteRegistry", "RetailDemo", "UevAgentService", "shpamsvc"),
		},
		{
			ID: "svc_background_manual", Title: "Servicios de fondo: solo cuando hacen falta", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Pasa a inicio manual el servicio de dispositivos conectados (CDP), las actualizaciones de Edge, los mapas sin conexión y la ubicación. Se inician solos si alguna app los pide.",
			Changes:     svcs(startManual, "CDPSvc", "edgeupdate", "edgeupdatem", "MapsBroker", "lfsvc"),
		},
		{
			ID: "svc_xbox_manual", Title: "Servicios de Xbox: inicio manual", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Los servicios de autenticación, partidas guardadas y accesorios de Xbox dejan de arrancar con Windows y se inician cuando un juego o la app de Xbox los necesita.",
			Changes:     svcs(startManual, "XblAuthManager", "XblGameSave", "XboxGipSvc", "XboxNetApiSvc"),
		},
		{
			ID: "svc_search_manual", Title: "Indexador de búsqueda: inicio manual", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Windows Search deja de indexar archivos en segundo plano al arrancar. La búsqueda de archivos puede ser más lenta.",
			Changes:     svcs(startManual, "WSearch"),
		},
		{
			ID: "svc_print_manual", Title: "Cola de impresión y uso compartido: inicio manual", Category: "Servicios", Admin: true, Reboot: true,
			Description: "El servicio de impresión y el de carpetas compartidas (Server) arrancan solo cuando se usan. Si compartís carpetas o imprimís en red, dejalo apagado.",
			Changes:     svcs(startManual, "Spooler", "LanmanServer"),
		},
		{
			ID: "svc_telemetry", Title: "Servicios de telemetría y diagnóstico", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Deshabilita la telemetría conectada (DiagTrack), el enrutador de mensajes push, el servicio de colecta de datos y el asistente de compatibilidad de programas.",
			Changes:     svcs(startDisabled, "DiagTrack", "dmwappushservice", "DcpSvc", "PcaSvc"),
		},
		{
			ID: "svc_error_reporting", Title: "Servicio de informe de errores", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Deshabilita el servicio que envía los informes de error de Windows (WerSvc).",
			Changes:     svcs(startDisabled, "WerSvc"),
		},
		{
			ID: "autologgers", Title: "Desactivar registros de diagnóstico de arranque", Category: "Servicios", Admin: true, Reboot: true,
			Description: "Apaga los registradores de eventos que Windows inicia en cada arranque (DiagLog, SQMLogger, WiFiSession y otros). Reduce escritura a disco y trabajo al iniciar.",
			Changes:     loggers,
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
			ID: "update_pause", Title: "Permitir pausar Windows Update por más tiempo", Category: "Sistema", Admin: true,
			Description: "Quita el límite de días al pausar actualizaciones, para que no se instalen en medio de una sesión de juego. No desactiva las actualizaciones.",
			Changes:     []RegChange{dw("HKLM", `SOFTWARE\Microsoft\WindowsUpdate\UX\Settings`, "FlightSettingsMaxPauseDays", 0xffffffff)},
		},
		{
			ID: "webdav_limit", Title: "Quitar límite de tamaño de archivos WebDAV", Category: "Red", Admin: true,
			Description: "Elimina el límite de 50 MB al copiar archivos desde unidades de red WebDAV.",
			Changes: []RegChange{{Root: "HKLM", Path: `SYSTEM\CurrentControlSet\Services\WebClient\Parameters`, Name: "FileSizeLimitInBytes",
				Kind: "dword", D: 0xffffffff, MustExist: true}},
		},
	}
}
