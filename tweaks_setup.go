package main

// Ajustes adaptados de los scripts de "Setup", "Graphics", "Windows" y "Hardware" de Ultimate
// (FR33THY, MIT; ver THIRD-PARTY-NOTICES.md y docs/ULTIMATE_ADAPTATION.md). Los que se pueden
// deshacer llevan interruptor; los que no, son acciones únicas con botón «Aplicar». Lo que no se
// adaptó, con su motivo, está en docs/ULTIMATE_ADAPTATION.md.

const (
	nvTweakKey  = `SYSTEM\CurrentControlSet\Services\nvlddmkm\Parameters\Global\NVTweak`
	gpuClassKey = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\` + gpuPlaceholder
	explorerKey = `Software\Microsoft\Windows\CurrentVersion\Explorer`
	appEvents   = `AppEvents\Schemes\Apps\.Default\`
	msiKey      = `SYSTEM\CurrentControlSet\Enum\` + dispPlaceholder + `\Device Parameters\Interrupt Management\MessageSignaledInterruptProperties`

	// IDs y valores verificados contra NvApiDriverSettings.h del SDK oficial de NVIDIA (MIT).
	nvVSyncMode         = 0x00A879CF // Vertical Sync
	nvVSyncForceOff     = 0x08416747 // VSYNCMODE_FORCEOFF
	nvVSyncTearControl  = 0x005A375C // Vertical Sync Tear Control
	nvTearDisable       = 0x96861077 // VSYNCTEARCONTROL_DISABLE
	nvRefreshOverride   = 0x0064B541 // Preferred refresh rate
	nvRefreshHighest    = 0x00000001 // REFRESH_RATE_OVERRIDE_HIGHEST_AVAILABLE
	nvTexQuality        = 0x00CE2691 // Texture filtering - Quality
	nvTexHighPerf       = 0x00000014 // QUALITY_ENHANCEMENTS_HIGHPERFORMANCE
	nvAnisoOpt          = 0x0084CD70 // Anisotropic filter optimization
	nvAnisoSampleOpt    = 0x00E73211 // Anisotropic sample optimization
	nvThreadedOpt       = 0x20C1221E // Threaded optimization
	nvGDICompat         = 0x2072C5A3 // OpenGL GDI compatibility
	nvShaderCacheMax    = 0x00AC8497 // Shader disk cache maximum size
	nvOGLPresentDXMode  = 0x20D690F8 // Vulkan/OpenGL present method
	nvShaderCacheNoCap  = 0xFFFFFFFF // PS_SHADERDISKCACHE_MAX_SIZE_MAX
	nvOGLThreadsEnabled = 0x00000001 // OGL_THREAD_CONTROL_ENABLE
)

func setupCatalog() []Tweak {
	// Sonidos del sistema: el valor predeterminado de cada evento queda vacío (sin sonido).
	events := []string{"CriticalBatteryAlarm", "DeviceConnect", "DeviceDisconnect", "DeviceFail", "FaxBeep", "LowBatteryAlarm",
		"MailBeep", "MessageNudge", "Notification.Default", "Notification.IM", "Notification.Mail", "Notification.Proximity",
		"Notification.Reminder", "Notification.SMS", "ProximityConnection", "SystemAsterisk", "SystemExclamation", "SystemHand",
		"SystemNotification", "WindowsUAC"}
	sounds := []RegChange{{Root: "HKCU", Path: appEvents + `.Default\.Current`, Name: "", Kind: "sz", S: "", MustExist: true}}
	for _, e := range events {
		sounds = append(sounds, RegChange{Root: "HKCU", Path: appEvents + e + `\.Current`, Name: "", Kind: "sz", S: "", MustExist: true})
	}
	sounds = append(sounds, RegChange{Root: "HKCU", Path: `AppEvents\Schemes`, Name: "", Kind: "sz", S: ".None", MustExist: true})

	amdUMD := func(name, hexv string) RegChange {
		c := bin("HKLM", gpuClass+`\UMD`, name, hexv)
		c.MustExist = true
		return c
	}

	return []Tweak{
		{
			ID: "memory_compression_off", Title: "Desactivar la compresión de memoria", Category: "Sistema", Admin: true, Reboot: true,
			Description: "Windows deja de comprimir páginas de RAM poco usadas: ahorra algo de CPU a cambio de usar más memoria. Tiene sentido con 16 GB o más; " +
				"con poca RAM puede empeorar. Se aplica al reiniciar y se revierte con el mismo interruptor.",
			PS: &PSSpec{
				State: `(Get-MMAgent).MemoryCompression`, Want: "False",
				Apply: `Disable-MMAgent -MemoryCompression`, Revert: `Enable-MMAgent -MemoryCompression`,
			},
		},
		{
			ID: "store_autoupdate_off", Title: "Microsoft Store sin actualizaciones automáticas", Category: "Sistema", Admin: true,
			Description: "La Tienda deja de descargar actualizaciones de apps por su cuenta (AutoDownload = 2). Actualizás a mano cuando quieras.",
			Changes:     []RegChange{dw("HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsStore\WindowsUpdate`, "AutoDownload", 2)},
		},
		{
			ID: "timer_resolution", Title: "Resolución de temporizador alta mientras jugás", Category: "Sistema", Admin: true,
			Description: "Mientras South Tweaks detecta un juego abierto, pide al sistema la resolución de temporizador máxima y la libera al cerrarse. " +
				"Es la versión integrada del «servicio de timer resolution»: no instala ningún servicio ni compila código. Requiere tener South Tweaks abierto al jugar. " +
				"Sube un poco el consumo; el efecto en FPS no está medido.",
			Changes: []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\kernel`, "GlobalTimerResolutionRequests", 1)},
		},
		{
			ID: "gamebar_controller_off", Title: "Game Bar sin atajos del control", Category: "Juegos",
			Description: "Evita que el botón Xbox del control abra la Game Bar y que Ver + Menú actúen como botón de guía en las apps.",
			Changes: []RegChange{
				dw("HKCU", `Software\Microsoft\GameBar`, "UseNexusForGameBarEnabled", 0),
				dw("HKCU", `Software\Microsoft\GameBar`, "GamepadNexusChordEnabled", 0),
			},
		},
		{
			ID: "audio_ducking_off", Title: "Sin bajar el volumen del juego en llamadas", Category: "Entrada",
			Description: "Windows deja de bajar el volumen de otros sonidos cuando detecta una comunicación (por ejemplo, el chat de voz mientras jugás). Comunicaciones: no hacer nada.",
			Changes:     []RegChange{dw("HKCU", `Software\Microsoft\Multimedia\Audio`, "UserDuckingPreference", 3)},
		},
		{
			ID: "explorer_prefs", Title: "Explorador de archivos más claro y directo", Category: "Visual",
			Description: "Abre «Este equipo», muestra las extensiones de archivo y la ruta completa en la barra de título, y oculta carpetas frecuentes, archivos de la nube, avisos de sincronización y el asistente de uso compartido.",
			Changes: []RegChange{
				dw("HKCU", explorerAdv, "LaunchTo", 1), dw("HKCU", explorerAdv, "HideFileExt", 0),
				dw("HKCU", explorerAdv, "ShowSyncProviderNotifications", 0), dw("HKCU", explorerAdv, "SharingWizardOn", 0),
				dw("HKCU", explorerKey, "ShowFrequent", 0), dw("HKCU", explorerKey, "ShowCloudFilesInQuickAccess", 0),
				dw("HKCU", explorerKey+`\CabinetState`, "FullPath", 1),
			},
		},
		{
			ID: "startup_sound_off", Title: "Sin sonido de inicio de Windows", Category: "Visual", Admin: true,
			Description: "Silencia el sonido que se reproduce al iniciar Windows.",
			Changes: []RegChange{
				dw("HKLM", `Software\Microsoft\Windows\CurrentVersion\Authentication\LogonUI\BootAnimation`, "DisableStartupSound", 1),
				dw("HKLM", `Software\Microsoft\Windows\CurrentVersion\EditionOverrides`, "UserSetting_DisableStartupSound", 1),
			},
		},
		{
			ID: "sound_scheme_none", Title: "Esquema de sonidos: ninguno", Category: "Visual",
			Description: "Quita los sonidos del sistema (avisos, dispositivos, errores). Tus sonidos anteriores se guardan y vuelven al revertir.",
			Changes:     sounds,
		},
		{
			ID: "nv_driver_prefs", Title: "NVIDIA: PhysX en GPU, sin icono de bandeja y nitidez clásica", Category: "GPU", Admin: true, Reboot: true, Vendor: "nvidia",
			Description: "PhysX en la GPU (NvCplPhysxAuto = 0), herramientas de desarrollador visibles en el Panel de control, el ícono de la bandeja no arranca con Windows y se activa la nitidez clásica del driver (EnableGR535 = 0).",
			Changes: []RegChange{
				dw("HKLM", nvTweakKey, "NvCplPhysxAuto", 0), dw("HKLM", nvTweakKey, "NvDevToolsVisible", 1),
				dw("HKCU", `Software\NVIDIA Corporation\NvTray`, "StartOnLogin", 0),
				dw("HKLM", `SYSTEM\CurrentControlSet\Services\nvlddmkm\Parameters\FTS`, "EnableGR535", 0),
			},
		},
		{
			ID: "nv_hdcp_off", Title: "NVIDIA: desactivar HDCP", Category: "GPU", Admin: true, Reboot: true, Vendor: "nvidia",
			Description: "Apaga la protección de contenido HDCP del driver. Puede evitar parpadeos con algunos monitores; las apps de video protegido (streaming en alta calidad) pueden dejar de reproducir o bajar la resolución.",
			Changes:     []RegChange{dw("HKLM", gpuClass, "RMHdcpKeyglobZero", 1)},
		},
		{
			ID: "amd_software_prefs", Title: "AMD Software: sin overlay, avisos ni anuncios", Category: "GPU", Vendor: "amd",
			Description: "En AMD Software (Adrenalin): sin actualización automática, sin overlay en juego, sin atajos de teclado, sin ícono de bandeja, sin navegador web integrado, sin anuncios ni avisos y sin animaciones.",
			Changes: []RegChange{
				dw("HKCU", `Software\AMD\CN`, "AutoUpdate", 0), dw("HKCU", `Software\AMD\AIM`, "LaunchBugTool", 0), dw("HKCU", `Software\AMD\DVR`, "HotkeysDisabled", 1),
				sz("HKCU", `Software\AMD\CN`, "SystemTray", "false"), sz("HKCU", `Software\AMD\DVR`, "ShowRSOverlay", "false"),
				sz("HKCU", `Software\AMD\CN`, "RSXBrowserUnavailable", "true"), sz("HKCU", `Software\AMD\CN`, "AllowWebContent", "false"),
				sz("HKCU", `Software\AMD\CN`, "CN_Hide_Toast_Notification", "true"), sz("HKCU", `Software\AMD\CN`, "AnimationEffect", "false"),
			},
		},
		{
			ID: "amd_graphics_perf", Title: "AMD: gráficos en modo rendimiento", Category: "GPU", Admin: true, Reboot: true, Vendor: "amd",
			Description: "En el driver de AMD: espera de sincronía vertical siempre apagada, calidad de filtrado de texturas en rendimiento, teselación sobrescrita (nivel máximo apagado) y Vari-Bright al máximo. Baja algo de calidad de imagen.",
			Changes: []RegChange{
				amdUMD("VSyncControl", "3000"), amdUMD("TFQ", "3200"), amdUMD("Tessellation", "3100"), amdUMD("Tessellation_OPTION", "3200"),
				{Root: "HKLM", Path: gpuClass + `\power_v1`, Name: "abmlevel", Kind: "bin", Hex: "00000000", MustExist: true},
			},
		},
		{
			ID: "gpu_msi_mode", Title: "Modo MSI en la tarjeta de video", Category: "GPU", Admin: true, Reboot: true,
			Description: "Fuerza las interrupciones por mensaje (MSI) en tu(s) tarjeta(s) de video (MSISupported = 1). Los drivers modernos de NVIDIA y AMD ya lo usan; solo cambia algo si tu driver no lo traía activado. " +
				"Si tras reiniciar hay pantalla negra, entrá en modo seguro y revertilo. Solo afecta a dispositivos de video.",
			Changes: []RegChange{dw("HKLM", msiKey, "MSISupported", 1)},
		},
		{
			ID: "remove_copilot_app", Title: "Desinstalar la app Copilot", Category: "Acciones únicas", Admin: true, Once: true,
			Description: "Quita el paquete de Copilot de Windows. No se puede deshacer desde acá; si cambiás de idea, se reinstala desde la Microsoft Store. Para ocultarlo sin desinstalar, usá el ajuste «Desactivar Copilot».",
		},
		{
			ID: "remove_gamebar_app", Title: "Desinstalar Xbox Game Bar (solo la barra)", Category: "Acciones únicas", Admin: true, Once: true,
			Description: "Quita solo la superposición Xbox Game Bar. No toca Gaming Services ni la app de Xbox, que necesitan Game Pass y varios juegos de la Tienda. Se reinstala desde la Microsoft Store.",
		},
	}
}
