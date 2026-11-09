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
			ID: "south_power_plan", Title: "South's Core power plan", Category: "Energía", Power: true, PlanID: "south",
			Description: "Plan de energía de alto rendimiento para juegos: procesador siempre disponible con corriente, boost agresivo, núcleos sin estacionar (salvo CPUs híbridas y Ryzen X3D de dos CCD), y sin ahorro en discos, USB, PCIe ni Wi-Fi. Con batería conserva el ahorro. Al revertir vuelve tu plan anterior y este se borra.",
		}, {
			ID: "nvidia_profile", Title: "NVIDIA: perfil global de máximo rendimiento y baja latencia", Category: "GPU", Admin: true, Reboot: true, Vendor: "nvidia",
			Description: "Un solo perfil global del driver NVIDIA (API oficial) con todo lo de rendimiento: modo de energía «Preferir rendimiento máximo», sin cambios dinámicos de estado de energía de la GPU, " +
				"baja latencia (1 frame en cola), V-Sync apagado, control de tearing desactivado, frecuencia de actualización más alta disponible, filtrado de texturas en «alto rendimiento» " +
				"con optimizaciones anisotrópicas, hilos de OpenGL optimizados, compatibilidad GDI de OpenGL apagada, método de presentación DXGI para Vulkan/OpenGL apagado y caché de shaders activada y sin límite de tamaño. " +
				"Sube el consumo en reposo, baja algo de calidad de imagen y puede mostrar tearing. Cada valor previo se guarda y se restaura.",
			Nvidia: []NvSetting{
				{ID: nvSettingPreferredPState, Value: nvPStatePreferMax, Name: "Modo de administración de energía"},
				{ID: nvSettingPrerenderLimit, Value: 1, Name: "Modo de baja latencia (cuadros pre-renderizados)"},
				{ID: nvSettingShaderCache, Value: nvShaderCacheOn, Name: "Caché de shaders"},
				{ID: nvVSyncMode, Value: nvVSyncForceOff, Name: "V-Sync"},
				{ID: nvVSyncTearControl, Value: nvTearDisable, Name: "Control de tearing"},
				{ID: nvRefreshOverride, Value: nvRefreshHighest, Name: "Frecuencia de actualización preferida"},
				{ID: nvTexQuality, Value: nvTexHighPerf, Name: "Calidad de filtrado de texturas"},
				{ID: nvAnisoOpt, Value: 1, Name: "Optimización anisotrópica"},
				{ID: nvAnisoSampleOpt, Value: 1, Name: "Optimización de muestras anisotrópicas"},
				{ID: nvThreadedOpt, Value: nvOGLThreadsEnabled, Name: "Optimización de hilos (OpenGL)"},
				{ID: nvGDICompat, Value: 0, Name: "Compatibilidad GDI de OpenGL"},
				{ID: nvOGLPresentDXMode, Value: 0, Name: "Método de presentación Vulkan/OpenGL"},
				{ID: nvShaderCacheMax, Value: nvShaderCacheNoCap, Name: "Tamaño máximo de la caché de shaders"},
			},
			Changes: []RegChange{dw("HKLM", gpuClass, "DisableDynamicPstate", 1)},
		},
		{ID: "hibernate_off", Title: "Desactivar hibernación (borra hiberfil.sys)", Category: "Energía", Admin: true,
			Description: "Apaga la hibernación y libera el espacio de hiberfil.sys (similar a tu RAM). También desactiva el Inicio rápido. " +
				"Al revertir se vuelve a activar si estaba activada.",
			Cmd: &CmdSpec{
				Apply:  []string{"/hibernate", "off"},
				Revert: []string{"/hibernate", "on"},
				State:  dw("HKLM", `SYSTEM\CurrentControlSet\Control\Power`, "HibernateEnabled", 0),
			},
		},
		{
			ID: "retrim_ssd", Title: "Ejecutar TRIM ahora en los SSD", Category: "Acciones únicas", Admin: true, Once: true,
			Description: "Le avisa a cada SSD qué bloques están libres (defrag /L). No modifica tus datos ni se puede deshacer; los discos HDD se omiten.",
		},
		{
			ID: "flush_dns", Title: "Vaciar la caché de DNS", Category: "Acciones únicas", Admin: true, Once: true,
			Description: "Borra los nombres de sitios guardados; Windows los vuelve a resolver solo. Útil si un sitio o servidor no carga. No se puede deshacer, pero es inocuo.",
		},
		{
			ID: "trim_on", Title: "Asegurar TRIM en SSD", Category: "Sistema", Admin: true,
			Description: "Garantiza que Windows envíe TRIM a los SSD (DisableDeleteNotification = 0), lo que mantiene el rendimiento de escritura a largo plazo. Es el valor por defecto; corrige equipos donde se desactivó.",
			Changes:     []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\FileSystem`, "DisableDeleteNotification", 0)},
		},
		{
			ID: "search_local", Title: "Búsqueda de Inicio local y liviana (sin web ni nube)", Category: "Privacidad",
			Description: "La búsqueda del menú Inicio queda en tu equipo: sin resultados de internet ni sugerencias de Bing, sin cuadro de búsqueda dinámico y sin consultar tu cuenta Microsoft o de trabajo ni el historial de búsqueda del dispositivo. Responde más rápido y envía menos datos.",
			Changes: []RegChange{
				dw("HKCU", searchSettings, "IsDynamicSearchBoxEnabled", 0),
				dw("HKCU", `Software\Microsoft\Windows\CurrentVersion\Search`, "BingSearchEnabled", 0),
				dw("HKCU", `Software\Policies\Microsoft\Windows\Explorer`, "DisableSearchBoxSuggestions", 1),
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
