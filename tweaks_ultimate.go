package main

// Ajustes adaptados de "Ultimate" de FR33THY (https://github.com/FR33THYFR33THY/Ultimate, licencia MIT;
// ver THIRD-PARTY-NOTICES.md). Se tomaron solo los que son de registro, se pueden revertir, no
// debilitan la seguridad ni ejecutan código descargado. Cada uno se reescribió para el motor de
// South Tweaks (backup del valor original, verificación y rollback). Lo que NO se adaptó y por qué
// está en docs/ULTIMATE_ADAPTATION.md.

const (
	mouseKey2 = `Control Panel\Mouse`
	// Curvas de aceleración lineales (1:1) para escala de pantalla 100 %: 5 puntos de 8 bytes (X e Y).
	linearCurveX = "0000000000000000" + "c0cc0c0000000000" + "8099190000000000" + "4066260000000000" + "0033330000000000"
	linearCurveY = "0000000000000000" + "0000380000000000" + "0000700000000000" + "0000a80000000000" + "0000e00000000000"
	updUX        = `SOFTWARE\Microsoft\WindowsUpdate\UX\Settings`
)

func bin(root, path, name, hexv string) RegChange {
	return RegChange{Root: root, Path: path, Name: name, Kind: "bin", Hex: hexv}
}

func dated(name, spec string) RegChange {
	return RegChange{Root: "HKLM", Path: updUX, Name: name, Kind: "sz", Date: spec}
}

func ultimateCatalog() []Tweak {
	nic := `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}\` + nicPlaceholder
	nicStr := func(name string) RegChange {
		c := sz("HKLM", nic, name, "0")
		c.IfExists = true // solo si el driver del adaptador lo ofrece
		return c
	}
	return []Tweak{
		{
			ID: "windowed_flip", Title: "Optimizaciones para juegos en ventana (flip)", Category: "Juegos", MinBuild: buildWin11,
			Description: "Los juegos DX10/DX11 en ventana o sin bordes usan el modo de presentación moderno (flip): menos latencia y permite VRR. " +
				"Es la opción oficial de Windows 11 («Optimizaciones para juegos en ventana»). Algunos juegos pueden mostrar tearing: usá V-Sync o limitá los FPS. Conserva el resto de tus ajustes de gráficos.",
			Changes: []RegChange{{Root: "HKCU", Path: `Software\Microsoft\DirectX\UserGpuPreferences`, Name: "DirectXUserGlobalSettings", Kind: "sz", KV: "SwapEffectUpgradeEnable=1"}},
		},
		{
			ID: "raw_mouse_throttle", Title: "Mouse sin límite de sondeo en segundo plano", Category: "Entrada",
			Description: "Quita el tope de Windows al sondeo del mouse cuando la app no está al frente (RawMouseThrottleEnabled). Útil con mouse de 1000 Hz o más.",
			Changes:     []RegChange{dw("HKCU", mouseKey2, "RawMouseThrottleEnabled", 0)},
		},
		{
			ID: "mouse_raw_curve", Title: "Puntero 1:1 sin aceleración (raw aim)", Category: "Entrada",
			Description: "Sensibilidad 6/11, sin precisión del puntero y curvas de aceleración lineales: el movimiento del mouse en Windows es proporcional y predecible. " +
				"Las curvas son para escala de pantalla del 100 %. Cerrá sesión para aplicarlo del todo.",
			Changes: []RegChange{
				sz("HKCU", mouseKey2, "MouseSensitivity", "10"),
				sz("HKCU", mouseKey2, "MouseSpeed", "0"),
				sz("HKCU", mouseKey2, "MouseThreshold1", "0"),
				sz("HKCU", mouseKey2, "MouseThreshold2", "0"),
				bin("HKCU", mouseKey2, "SmoothMouseXCurve", linearCurveX),
				bin("HKCU", mouseKey2, "SmoothMouseYCurve", linearCurveY),
			},
		},
		{
			ID: "mpo_off", Title: "Desactivar Multiplane Overlay (MPO)", Category: "GPU", Admin: true, Reboot: true,
			Description: "Apaga MPO en el compositor (Dwm). Es una solución conocida para parpadeos, pantalla negra o tirones con algunos drivers de video; " +
				"en el resto de equipos no cambia nada o puede costar algo de eficiencia. Aplicalo solo si tenés ese síntoma.",
			Changes: []RegChange{dw("HKLM", `SOFTWARE\Microsoft\Windows\Dwm`, "OverlayTestMode", 5)},
		},
		{
			ID: "flip_immediate", Title: "Flip inmediato forzado (experimental)", Category: "GPU", Admin: true, Reboot: true,
			Description: "Activa ForceFlipTrueImmediateMode en el planificador de gráficos: presenta los frames sin esperar al próximo refresco. " +
				"Puede bajar la latencia en algunos juegos DX12 pero también causar tearing. No está documentado por Microsoft: probalo y medí.",
			Changes: []RegChange{dw("HKLM", `SYSTEM\CurrentControlSet\Control\GraphicsDrivers\Scheduler`, "ForceFlipTrueImmediateMode", 1)},
		},
		{
			ID: "nic_power_off", Title: "Adaptador de red sin ahorro de energía", Category: "Red", Admin: true, Reboot: true,
			Description: "En tus adaptadores Ethernet y Wi-Fi: Windows no puede apagarlos para ahorrar energía ni usarlos para despertar el equipo, y se desactiva " +
				"Energy-Efficient Ethernet y Green Ethernet (si el driver los ofrece). Evita microcortes y latencia al \"despertar\" el enlace. Solo adaptadores físicos.",
			Changes: []RegChange{
				dw("HKLM", nic, "PnPCapabilities", 24),
				nicStr("*EEE"), nicStr("AdvancedEEE"), nicStr("EEELinkAdvertisement"),
				nicStr("EnableGreenEthernet"), nicStr("GigaLite"), nicStr("ULPMode"), nicStr("SipsEnabled"),
			},
		},
		{
			ID: "bg_apps_policy", Title: "Prohibir apps en segundo plano (política)", Category: "Sistema", Admin: true,
			Description: "Política del sistema que impide que las apps de la Tienda corran en segundo plano para todos los usuarios. Es la versión de administrador del ajuste de usuario.",
			Changes:     []RegChange{dw("HKLM", `SOFTWARE\Policies\Microsoft\Windows\AppPrivacy`, "LetAppsRunInBackground", 2)},
		},
		{
			ID: "updates_pause", Title: "Pausar Windows Update 35 días", Category: "Sistema", Admin: true,
			Description: "Pausa las actualizaciones de calidad y de características por 35 días desde hoy, para que no se instalen en medio de tus sesiones. " +
				"Los parches de seguridad se retrasan: al revertir o al vencer la pausa se instalan con normalidad.",
			Changes: []RegChange{
				dated("PauseUpdatesStartTime", "now"), dated("PauseUpdatesExpiryTime", "+35d"),
				dated("PauseFeatureUpdatesStartTime", "now"), dated("PauseFeatureUpdatesEndTime", "+35d"),
				dated("PauseQualityUpdatesStartTime", "now"), dated("PauseQualityUpdatesEndTime", "+35d"),
			},
		},
	}
}
