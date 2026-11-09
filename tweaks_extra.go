package main

// Ajustes adicionales. Las claves y valores de registro son configuración pública de Windows
// (políticas, claves de usuario); se eligieron tras revisar qué ajustes aplican herramientas
// de optimización conocidas y se reimplementaron aquí con el motor propio (backup + rollback).

const (
	explorerAdv = `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`
	contentDM   = `Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`
	polWindows  = `SOFTWARE\Policies\Microsoft\Windows`
	polDataColl = polWindows + `\DataCollection`
	polCloud    = polWindows + `\CloudContent`
	polSystem   = polWindows + `\System`
	polAI       = polWindows + `\WindowsAI`
	polAI_CU    = `Software\Policies\Microsoft\Windows\WindowsAI`
	accessKey   = `Control Panel\Accessibility`
)

const (
	buildWin11 = 22000
	build24H2  = 26100
)

func extraCatalog() []Tweak {
	return []Tweak{
		// ---- Rendimiento / sistema ----
		{
			ID: "power_throttle", Title: "Desactivar Power Throttling", Category: "Sistema", Admin: true, Reboot: true,
			Description: "Evita que Windows limite la CPU de procesos en segundo plano para ahorrar energía. Puede subir el consumo en notebooks.",
			Changes:     []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\Power\PowerThrottling`, "PowerThrottlingOff", 1)},
		},
		{
			ID: "explorer_startup", Title: "Explorador sin demora de arranque", Category: "Visual",
			Description: "Quita la demora artificial con la que Windows carga el escritorio y las apps de inicio tras iniciar sesión.",
			Changes:     []RegChange{dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Explorer\Serialize`, "StartupDelayInMSec", 0)},
		},
		{
			ID: "visual_fx", Title: "Menos animaciones de la interfaz", Category: "Visual",
			Description: "Desactiva animaciones de la barra de tareas, sombras de listas y Aero Peek. Solo cambia la apariencia.",
			Changes: []RegChange{
				dw("HKCU", explorerAdv, "TaskbarAnimations", 0),
				dw("HKCU", explorerAdv, "ListviewShadow", 0),
				dw("HKCU", `Software\Microsoft\Windows\DWM`, "EnableAeroPeek", 0),
			},
		},
		{
			ID: "sticky_keys", Title: "Sin avisos de Teclas especiales", Category: "Entrada",
			Description: "Desactiva los atajos que abren los avisos de Teclas especiales, Filtro y Alternancia (por ejemplo al apretar Shift 5 veces en un juego).",
			Changes: []RegChange{
				sz("HKCU", accessKey+`\StickyKeys`, "Flags", "26"),
				sz("HKCU", accessKey+`\Keyboard Response`, "Flags", "2"),
				sz("HKCU", accessKey+`\ToggleKeys`, "Flags", "34"),
			},
		},
		{
			ID: "search_web", Title: "Búsqueda de Inicio sin resultados web", Category: "Sistema",
			Description: "El menú Inicio deja de buscar en internet cada vez que escribís: responde más rápido y envía menos datos.",
			Changes:     []RegChange{dw("HKCU", `Software\Policies\Microsoft\Windows\Explorer`, "DisableSearchBoxSuggestions", 1)},
		},
		{
			ID: "delivery_opt", Title: "Desactivar Optimización de distribución", Category: "Red", Admin: true,
			Description: "Windows deja de usar tu conexión para compartir actualizaciones con otros equipos. Libera ancho de banda mientras jugás.",
			Changes:     []RegChange{dw("HKLM", polWindows+`\DeliveryOptimization`, "DODownloadMode", 0)},
		},

		// ---- Privacidad (también reducen tareas en segundo plano) ----
		{
			ID: "content_delivery", Title: "Sin sugerencias ni contenido promocionado", Category: "Privacidad",
			Description: "Apaga las sugerencias de apps y contenido promocionado de Windows (Inicio, pantalla de bloqueo, notificaciones).",
			Changes: []RegChange{
				dw("HKCU", contentDM, "ContentDeliveryAllowed", 0),
				dw("HKCU", contentDM, "SubscribedContent-338387Enabled", 0),
				dw("HKCU", contentDM, "SubscribedContent-338388Enabled", 0),
				dw("HKCU", contentDM, "SubscribedContent-338389Enabled", 0),
				dw("HKCU", contentDM, "SubscribedContent-353698Enabled", 0),
				dw("HKCU", contentDM, "SystemPaneSuggestionsEnabled", 0),
			},
		},
		{
			ID: "preinstalled_apps", Title: "Bloquear reinstalación de apps preinstaladas", Category: "Privacidad",
			Description: "Evita que Windows instale automáticamente apps patrocinadas en tu cuenta. No desinstala nada.",
			Changes: []RegChange{
				dw("HKCU", contentDM, "PreInstalledAppsEnabled", 0),
				dw("HKCU", contentDM, "PreInstalledAppsEverEnabled", 0),
				dw("HKCU", contentDM, "OemPreInstalledAppsEnabled", 0),
				dw("HKCU", contentDM, "SilentInstalledAppsEnabled", 0),
			},
		},
		{
			ID: "ads", Title: "Desactivar ID de publicidad y experiencias personalizadas", Category: "Privacidad", Admin: true,
			Description: "Apaga el identificador de publicidad, las experiencias a medida con datos de diagnóstico y las sugerencias de terceros.",
			Changes: []RegChange{
				dw("HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\AdvertisingInfo`, "Enabled", 0),
				dw("HKLM", polWindows+`\AdvertisingInfo`, "DisabledByGroupPolicy", 1),
				dw("HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Privacy`, "TailoredExperiencesWithDiagnosticDataEnabled", 0),
				dw("HKLM", polCloud, "DisableTailoredExperiencesWithDiagnosticData", 1),
				dw("HKLM", polCloud, "DisableWindowsConsumerFeatures", 1),
				dw("HKLM", polCloud, "DisableSoftLanding", 1),
				dw("HKLM", polCloud, "DisableThirdPartySuggestions", 1),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\UserProfileEngagement`, "ScoobeSystemSettingEnabled", 0),
				dw("HKCU", `SOFTWARE\Microsoft\InputPersonalization`, "RestrictImplicitInkCollection", 1),
				dw("HKCU", `SOFTWARE\Microsoft\InputPersonalization`, "RestrictImplicitTextCollection", 1),
				dw("HKCU", `SOFTWARE\Microsoft\InputPersonalization\TrainedDataStore`, "HarvestContacts", 0),
				dw("HKCU", `Control Panel\International\User Profile`, "HttpAcceptLanguageOptOut", 1),
			},
		},
		{
			ID: "activity_history", Title: "Desactivar historial de actividades", Category: "Privacidad", Admin: true,
			Description: "Windows deja de registrar y sincronizar tu historial de actividades (Línea de tiempo).",
			Changes: []RegChange{
				dw("HKLM", polSystem, "PublishUserActivities", 0),
				dw("HKLM", polSystem, "EnableActivityFeed", 0),
				dw("HKLM", polSystem, "PublishUserActivitiesOnUserConsent", 0),
				dw("HKLM", polSystem, "UploadUserActivities", 0),
			},
		},
		{
			ID: "news_interests", Title: "Desactivar Noticias e intereses", Category: "Privacidad", Admin: true,
			Description: "Quita el widget de noticias de la barra de tareas, que consume memoria y red en segundo plano.",
			Changes: []RegChange{
				dw("HKLM", `SOFTWARE\Policies\Microsoft\Dsh`, "AllowNewsAndInterests", 0),
				dw("HKLM", `SOFTWARE\Microsoft\PolicyManager\default\NewsAndInterests\AllowNewsAndInterests`, "value", 0),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Feeds`, "ShellFeedsTaskbarViewMode", 0),
			},
		},
		{
			ID: "telemetry", Title: "Reducir telemetría de Windows", Category: "Privacidad", Admin: true,
			Description: "Baja la recolección de datos de diagnóstico al mínimo permitido y apaga el inventario de aplicaciones y los avisos de opinión. Algunas ediciones de Windows mantienen un nivel básico.",
			Changes: []RegChange{
				dw("HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\DataCollection`, "AllowTelemetry", 0),
				dw("HKLM", polDataColl, "AllowTelemetry", 0),
				dw("HKLM", polDataColl, "DoNotShowFeedbackNotifications", 1),
				dw("HKLM", polDataColl, "AllowCommercialDataPipeline", 0),
				dw("HKLM", polDataColl, "AllowDeviceNameInTelemetry", 0),
				dw("HKLM", polWindows+`\AppCompat`, "DisableInventory", 1),
				dw("HKLM", polWindows+`\AppCompat`, "AITEnable", 0),
				dw("HKCU", `SOFTWARE\Microsoft\Siuf\Rules`, "NumberOfSIUFInPeriod", 0),
				dw("HKCU", explorerAdv, "Start_TrackProgs", 0),
				dw("HKCU", `Software\Policies\Microsoft\Windows\EdgeUI`, "DisableMFUTracking", 1),
				dw("HKCU", `SOFTWARE\Policies\Microsoft\Assistance\Client\1.0`, "NoExplicitFeedback", 1),
				dw("HKLM", `SOFTWARE\Policies\Microsoft\Assistance\Client\1.0`, "NoActiveHelp", 1),
				dw("HKLM", polDataColl, "MicrosoftEdgeDataOptIn", 0),
				dw("HKCU", `SOFTWARE\Microsoft\Speech_OneCore\Settings\OnlineSpeechPrivacy`, "HasAccepted", 0),
			},
		},
		{
			ID: "error_reporting", Title: "Desactivar informe de errores de Windows", Category: "Privacidad", Admin: true,
			Description: "Windows deja de recopilar y enviar informes cuando una aplicación se cierra con error.",
			Changes: []RegChange{
				dw("HKLM", `SOFTWARE\Microsoft\Windows\Windows Error Reporting`, "Disabled", 1),
				dw("HKLM", polWindows+`\HandwritingErrorReports`, "PreventHandwritingErrorReports", 1),
			},
		},
		{
			ID: "copilot", Title: "Desactivar Copilot", Category: "Privacidad", Admin: true, MinBuild: buildWin11,
			Description: "Oculta Copilot y su botón en la barra de tareas mediante política. Solo Windows 11.",
			Changes: []RegChange{
				dw("HKLM", polWindows+`\WindowsCopilot`, "TurnOffWindowsCopilot", 1),
				dw("HKCU", `Software\Policies\Microsoft\Windows\WindowsCopilot`, "TurnOffWindowsCopilot", 1),
				dw("HKCU", explorerAdv, "ShowCopilotButton", 0),
				sz("HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Shell Extensions\Blocked`, "{CB3B0003-8088-4EDE-8769-8B354AB2FF8C}", ""),
			},
		},
		{
			ID: "telemetry_tasks", Title: "Desactivar tareas programadas de telemetría", Category: "Privacidad", Admin: true,
			Description: "Deshabilita las tareas que recopilan datos de compatibilidad y experiencia de usuario (Application Experience, CEIP, Siuf, Autochk Proxy). Solo las que existen en tu Windows.",
			Tasks: []string{
				`\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser`,
				`\Microsoft\Windows\Application Experience\ProgramDataUpdater`,
				`\Microsoft\Windows\Application Experience\MareBackup`,
				`\Microsoft\Windows\Application Experience\StartupAppTask`,
				`\Microsoft\Windows\Application Experience\PcaPatchDbTask`,
				`\Microsoft\Windows\Autochk\Proxy`,
				`\Microsoft\Windows\Customer Experience Improvement Program\Consolidator`,
				`\Microsoft\Windows\Customer Experience Improvement Program\UsbCeip`,
				`\Microsoft\Windows\Feedback\Siuf\DmClient`,
				`\Microsoft\Windows\Feedback\Siuf\DmClientOnScenarioDownload`,
			},
		},
		{
			ID: "error_tasks", Title: "Desactivar tareas de informes de errores y diagnóstico de disco", Category: "Privacidad", Admin: true,
			Description: "Deshabilita las tareas programadas que envían informes de errores y datos de diagnóstico de disco.",
			Tasks: []string{
				`\Microsoft\Windows\Windows Error Reporting\QueueReporting`,
				`\Microsoft\Windows\DiskDiagnostic\Microsoft-Windows-DiskDiagnosticDataCollector`,
				`\Microsoft\Windows\DiskDiagnostic\Microsoft-Windows-DiskDiagnosticResolver`,
				`\Microsoft\Windows\Diagnosis\Scheduled`,
			},
		},
		{
			ID: "location_off", Title: "Desactivar ubicación y sensores", Category: "Privacidad", Admin: true,
			Description: "Apaga los servicios de ubicación y sensores, las actualizaciones de mapas y los informes de puntos Wi-Fi. Las apps que usan tu ubicación (mapas, clima) dejan de saber dónde estás.",
			Tasks:       []string{`\Microsoft\Windows\Maps\MapsUpdateTask`},
			Changes: []RegChange{
				dw("HKLM", polWindows+`\LocationAndSensors`, "DisableLocation", 1),
				dw("HKLM", polWindows+`\LocationAndSensors`, "DisableSensors", 1),
				dw("HKLM", `SYSTEM\Maps`, "AutoUpdateEnabled", 0),
				dw("HKCU", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Sensor\Permissions\{BFA794E4-F964-4FDB-90F6-51056BFE4B44}`, "SensorPermissionState", 0),
				sz("HKCU", `Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location`, "Value", "Deny"),
				sz("HKLM", `Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location`, "Value", "Deny"),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Geolocation`, "Status", 0),
				dw("HKLM", `SYSTEM\CurrentControlSet\Services\lfsvc\Service\Configuration`, "Status", 0),
				dw("HKLM", `Software\Microsoft\PolicyManager\default\WiFi\AllowWiFiHotSpotReporting`, "Value", 0),
				dw("HKLM", `Software\Microsoft\PolicyManager\default\WiFi\AllowAutoConnectToWiFiSenseHotspots`, "Value", 0),
			},
		},
		{
			ID: "settings_ads", Title: "Sin anuncios en Configuración (cuenta Microsoft 365)", Category: "Privacidad", Admin: true,
			Description: "Quita las ofertas y avisos de Microsoft 365 que aparecen en la página de inicio de Configuración.",
			Changes:     []RegChange{dw("HKLM", polCloud, "DisableConsumerAccountStateContent", 1)},
		},
		{
			ID: "find_my_device", Title: "Desactivar «Encontrar mi dispositivo»", Category: "Privacidad", Admin: true,
			Description: "Windows deja de enviar la ubicación del equipo a tu cuenta Microsoft.",
			Changes:     []RegChange{dw("HKLM", `SOFTWARE\Policies\Microsoft\FindMyDevice`, "AllowFindMyDevice", 0)},
		},
		{
			ID: "start_recommended", Title: "Ocultar «Recomendado» del menú Inicio", Category: "Privacidad", MinBuild: buildWin11,
			Description: "Quita la sección de archivos y apps recomendados del menú Inicio. Solo Windows 11.",
			Changes:     []RegChange{dw("HKCU", `SOFTWARE\Policies\Microsoft\Windows\Explorer`, "HideRecommendedSection", 1)},
		},
		{
			ID: "meet_now", Title: "Ocultar el botón «Reunirse ahora»", Category: "Privacidad", MaxBuild: buildWin11 - 1,
			Description: "Quita el icono de Reunirse ahora de la barra de tareas. Solo Windows 10.",
			Changes:     []RegChange{dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Policies\Explorer`, "HideSCAMeetNow", 1)},
		},
		{
			ID: "cortana", Title: "Desactivar Cortana y búsqueda en la nube", Category: "Privacidad", Admin: true, MaxBuild: buildWin11 - 1,
			Description: "Apaga Cortana, la búsqueda en la nube y el uso de tu ubicación en la búsqueda. Solo Windows 10.",
			Changes: []RegChange{
				dw("HKLM", polWindows+`\Windows Search`, "AllowCortana", 0),
				dw("HKLM", polWindows+`\Windows Search`, "AllowCloudSearch", 0),
				dw("HKLM", polWindows+`\Windows Search`, "AllowCortanaAboveLock", 0),
				dw("HKLM", polWindows+`\Windows Search`, "AllowSearchToUseLocation", 0),
				dw("HKLM", polWindows+`\Windows Search`, "ConnectedSearchUseWeb", 0),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Search`, "CortanaConsent", 0),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Search`, "CortanaConsent2", 0),
			},
		},
		{
			ID: "ai_recall", Title: "Desactivar Recall y Click to Do", Category: "Privacidad", Admin: true, MinBuild: build24H2,
			Description: "Impide que Windows guarde capturas periódicas de tu pantalla (Recall) y desactiva Click to Do. Solo Windows 11 24H2 o superior.",
			Changes: []RegChange{
				dw("HKLM", polAI, "DisableAIDataAnalysis", 1),
				dw("HKLM", polAI, "AllowRecallEnablement", 0),
				dw("HKLM", polAI, "TurnOffSavingSnapshots", 1),
				dw("HKLM", polAI, "DisableClickToDo", 1),
				dw("HKCU", polAI_CU, "DisableAIDataAnalysis", 1),
				dw("HKCU", polAI_CU, "TurnOffSavingSnapshots", 1),
				dw("HKCU", polAI_CU, "DisableClickToDo", 1),
			},
		},
	}
}
