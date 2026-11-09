package main

// Ajustes nuevos para dejar la PC más limpia y con menos trabajo en segundo plano. Ninguno promete FPS:
// cada descripción dice qué cambia y qué se pierde.

func cleanCatalog() []Tweak {
	return []Tweak{
		{
			ID: "exclude_wu_drivers", Title: "Windows Update sin drivers", Category: "Sistema", Admin: true,
			Description: "Windows Update deja de instalar o reemplazar drivers (por ejemplo el de tu placa de video) con las actualizaciones de calidad. Los drivers los actualizás vos desde el fabricante.",
			Changes:     []RegChange{dw("HKLM", wuPol, "ExcludeWUDriversInQualityUpdate", 1)},
		},
		{
			ID: "edge_background_off", Title: "Edge sin arranque ni procesos en segundo plano", Category: "Sistema", Admin: true,
			Description: "Edge deja de precargarse al iniciar Windows y de seguir corriendo en segundo plano al cerrarlo. Libera RAM. Edge va a mostrar «administrado por tu organización» porque se usa una política del sistema.",
			Changes: []RegChange{
				dw("HKLM", `SOFTWARE\Policies\Microsoft\Edge`, "StartupBoostEnabled", 0),
				dw("HKLM", `SOFTWARE\Policies\Microsoft\Edge`, "BackgroundModeEnabled", 0),
			},
		},
		{
			ID: "maintenance_off", Title: "Sin mantenimiento automático de Windows", Category: "Sistema", Admin: true,
			Description: "Windows no ejecuta por su cuenta el mantenimiento programado (optimizar unidades, diagnóstico, actualizaciones de apps) mientras jugás. Tampoco lo hace en reposo: tenés que correr TRIM o los diagnósticos vos de vez en cuando.",
			Changes:     []RegChange{dw("HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Schedule\Maintenance`, "MaintenanceDisabled", 1)},
		},
		{
			ID: "ntfs_lastaccess_off", Title: "Disco: sin hora de último acceso a archivos", Category: "Sistema", Admin: true, Reboot: true,
			Description: "NTFS deja de escribir la hora de último acceso cada vez que se lee un archivo: menos escrituras en el disco. Algunos programas de backup o de limpieza usan ese dato.",
			Changes:     []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\FileSystem`, "NtfsDisableLastAccessUpdate", 0x80000001)},
		},
		{
			ID: "prevent_device_encryption", Title: "Impedir el cifrado automático de BitLocker", Category: "Sistema", Admin: true,
			Description: "Windows no activa solo el cifrado del dispositivo al iniciar sesión con una cuenta Microsoft. No descifra lo que ya está cifrado (para eso está «Quitar BitLocker»). Tus archivos quedan sin cifrar si no lo activás vos.",
			Changes:     []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\BitLocker`, "PreventDeviceEncryption", 1)},
		},
		{
			ID: "remove_bloat_apps", Title: "Quitar apps preinstaladas que casi nadie usa", Category: "Acciones únicas", Admin: true, Once: true,
			Description: "Desinstala de todos los usuarios una lista fija: Noticias, Tiempo, Obtener ayuda, Sugerencias, Solitaire, Hub de Office, Contactos, Centro de opiniones, Clipchamp, Teams personal, Mixed Reality, Mapas, Skype, Cortana y Power Automate. No toca la Tienda, Xbox, Fotos, Calculadora ni Terminal. Se reinstalan desde la Microsoft Store. Cada paquete se verifica después.",
		},
	}
}

// bloatApps: paquetes AppX fijos que quita «remove_bloat_apps» (patrón de nombre, nombre legible).
var bloatApps = [][2]string{
	{"Microsoft.BingNews", "Noticias"}, {"Microsoft.BingWeather", "Tiempo"}, {"Microsoft.GetHelp", "Obtener ayuda"},
	{"Microsoft.Getstarted", "Sugerencias"}, {"Microsoft.MicrosoftSolitaireCollection", "Solitaire"},
	{"Microsoft.MicrosoftOfficeHub", "Hub de Office"}, {"Microsoft.People", "Contactos"},
	{"Microsoft.WindowsFeedbackHub", "Centro de opiniones"}, {"Clipchamp.Clipchamp", "Clipchamp"},
	{"MicrosoftTeams", "Teams (personal)"}, {"MSTeams", "Teams (nuevo)"}, {"Microsoft.MixedReality.Portal", "Mixed Reality"},
	{"Microsoft.WindowsMaps", "Mapas"}, {"Microsoft.SkypeApp", "Skype"}, {"Microsoft.549981C3F5F10", "Cortana"},
	{"Microsoft.PowerAutomateDesktop", "Power Automate"},
}
