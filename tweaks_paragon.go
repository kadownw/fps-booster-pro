package main

// Ajustes agregados tras analizar qué funciones ofrece Paragon Tweaking Utility (nombres de sus
// módulos y su interfaz). Ninguno copia código ni datos de Paragon: cada uno usa configuración
// pública de Windows o la API oficial del driver NVIDIA, con backup y reversión del motor.
//
// No se implementan a propósito las funciones de Paragon que debilitan Windows Defender
// ("RT Scan Trim", "Security Quiet"): reducen la protección y contradicen las reglas del proyecto.

const (
	searchSettings = `Software\Microsoft\Windows\CurrentVersion\SearchSettings`
	appPrivacy     = `SOFTWARE\Policies\Microsoft\Windows\AppPrivacy`
)

func paragonCatalog() []Tweak {
	return []Tweak{
		{
			ID: "fps_power_plan", Title: "Plan de energía FPS Booster: rendimiento estable", Category: "Energía", Power: true, PlanID: "fps",
			Description: "Crea un plan propio: la CPU sube de frecuencia al instante y baja de a poco (menos oscilación entre frames), boost agresivo, " +
				"USB, PCIe y Wi-Fi sin ahorro de energía y ventiladores antes que bajar frecuencia. Conserva los estados de reposo del procesador " +
				"para no calentar en vano. En CPUs híbridas y Ryzen X3D de dos CCD respeta el estacionamiento de núcleos. Al revertir vuelve tu plan anterior y se borra.",
		},
		{
			ID: "nvidia_profile", Title: "Perfil NVIDIA: máximo rendimiento y baja latencia", Category: "GPU", Admin: true,
			Description: "Ajusta el perfil global del driver con la API oficial de NVIDIA: modo de energía «Preferir rendimiento máximo» (relojes estables), " +
				"modo de baja latencia activado (1 frame en cola) y caché de shaders activada. Sube el consumo en reposo. Se guarda y restaura cada valor previo.",
			Nvidia: []NvSetting{
				{ID: nvSettingPreferredPState, Value: nvPStatePreferMax, Name: "Modo de administración de energía"},
				{ID: nvSettingPrerenderLimit, Value: 1, Name: "Modo de baja latencia (cuadros pre-renderizados)"},
				{ID: nvSettingShaderCache, Value: nvShaderCacheOn, Name: "Caché de shaders"},
			},
		},
		{
			ID: "hibernate_off", Title: "Desactivar hibernación (borra hiberfil.sys)", Category: "Energía", Admin: true,
			Description: "Apaga la hibernación y libera el espacio de hiberfil.sys (similar a tu RAM). También desactiva el Inicio rápido. " +
				"Al revertir se vuelve a activar si estaba activada.",
			Cmd: &CmdSpec{
				Apply:  []string{"/hibernate", "off"},
				Revert: []string{"/hibernate", "on"},
				State:  dw("HKLM", `SYSTEM\CurrentControlSet\Control\Power`, "HibernateEnabled", 0),
			},
		},
		{
			ID: "trim_on", Title: "Asegurar TRIM en SSD", Category: "Sistema", Admin: true,
			Description: "Garantiza que Windows envíe TRIM a los SSD (DisableDeleteNotification = 0), lo que mantiene el rendimiento de escritura a largo plazo. Es el valor por defecto; corrige equipos donde se desactivó.",
			Changes:     []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\FileSystem`, "DisableDeleteNotification", 0)},
		},
		{
			ID: "search_local", Title: "Búsqueda de Inicio liviana y local", Category: "Privacidad",
			Description: "Desactiva el cuadro de búsqueda dinámico y la búsqueda de Bing en el menú Inicio. La búsqueda queda en tu equipo.",
			Changes: []RegChange{
				dw("HKCU", searchSettings, "IsDynamicSearchBoxEnabled", 0),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Search`, "BingSearchEnabled", 0),
			},
		},
		{
			ID: "search_cloud_off", Title: "Búsqueda sin contenido en la nube", Category: "Privacidad",
			Description: "La búsqueda de Windows deja de consultar tu cuenta Microsoft o de trabajo y el historial de búsqueda del dispositivo.",
			Changes: []RegChange{
				dw("HKCU", searchSettings, "IsAADCloudSearchEnabled", 0),
				dw("HKCU", searchSettings, "IsMSACloudSearchEnabled", 0),
				dw("HKCU", searchSettings, "IsDeviceSearchHistoryEnabled", 0),
			},
		},
		{
			ID: "voice_activation_off", Title: "Desactivar activación por voz", Category: "Privacidad", Admin: true,
			Description: "Las apps dejan de escuchar palabras de activación por el micrófono.",
			Changes: []RegChange{
				dw("HKCU", `Software\Microsoft\Speech_OneCore\Settings\VoiceActivation\UserPreferenceForAllApps`, "AgentActivationEnabled", 0),
				dw("HKLM", appPrivacy, "LetAppsActivateWithVoice", 2),
			},
		},
		{
			ID: "cloud_sync_off", Title: "Desactivar sincronización de configuración", Category: "Privacidad", Admin: true,
			Description: "Windows deja de sincronizar la configuración entre dispositivos (tema, contraseñas, idioma).",
			Changes: []RegChange{
				dw("HKLM", `SOFTWARE\Policies\Microsoft\Windows\SettingSync`, "DisableSettingSync", 2),
				dw("HKLM", `SOFTWARE\Policies\Microsoft\Windows\SettingSync`, "DisableSettingSyncUserOverride", 1),
			},
		},
		{
			ID: "experimentation_lock", Title: "Bloquear experimentos de Windows", Category: "Privacidad", Admin: true,
			Description: "Impide que Microsoft active funciones de prueba en tu equipo (política System/AllowExperimentation).",
			Changes:     []RegChange{dw("HKLM", `SOFTWARE\Microsoft\PolicyManager\current\device\System`, "AllowExperimentation", 0)},
		},
		{
			ID: "media_tracking_off", Title: "Sin seguimiento de uso del Reproductor", Category: "Privacidad",
			Description: "El Reproductor de Windows Media deja de registrar el uso.",
			Changes:     []RegChange{dw("HKCU", `Software\Microsoft\MediaPlayer\Preferences`, "UsageTracking", 0)},
		},
		{
			ID: "nudge_blocker", Title: "Sin avisos de cuenta ni sugerencias de bienvenida", Category: "Privacidad",
			Description: "Quita las notificaciones de cuenta del menú Inicio, la experiencia de bienvenida tras actualizar y sugerencias en Configuración.",
			Changes: []RegChange{
				dw("HKCU", explorerAdv, "Start_AccountNotifications", 0),
				dw("HKCU", contentDM, "SoftLandingEnabled", 0),
				dw("HKCU", contentDM, "SubscribedContent-310093Enabled", 0),
				dw("HKCU", contentDM, "SubscribedContent-338393Enabled", 0),
				dw("HKCU", contentDM, "SubscribedContent-353694Enabled", 0),
				dw("HKCU", contentDM, "SubscribedContent-353696Enabled", 0),
			},
		},
		{
			ID: "proximity_deny", Title: "Sin descubrimiento de dispositivos cercanos", Category: "Privacidad", Admin: true,
			Description: "Desactiva Compartir con dispositivos cercanos y que las apps se sincronicen con dispositivos sin emparejar.",
			Changes: []RegChange{
				dw("HKLM", appPrivacy, "LetAppsSyncWithDevices", 2),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\CDP`, "RomeSdkChannelUserAuthzPolicy", 0),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\CDP`, "CdpSessionUserAuthzPolicy", 0),
			},
		},
	}
}
