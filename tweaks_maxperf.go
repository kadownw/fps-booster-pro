package main

import "strings"

// Servicios en «máximo rendimiento»: deshabilita casi todos los servicios opcionales de Windows.
// Se parte de la lista del script «17 Services» de Ultimate (230 servicios con Start=4) y se
// QUITAN los que, apagados, rompen cosas que un jugador usa a diario. El cambio es reversible:
// cada valor Start se guarda antes de tocarlo y «Desactivar» lo devuelve tal cual estaba.
//
// Se dejan SIEMPRE encendidos (no están en la lista):
//   - Seguridad y base del sistema: BFE y mpssvc (firewall), EventLog, EventSystem, PlugPlay, SamSs,
//     DPS, Themes, ShellHWDetection, StorSvc, TrkWks, vds, smphost, W32Time, CertPropSvc, EFS.
//   - Red y conexión: NlaSvc, iphlpsvc, LanmanWorkstation, Netlogon, WlanSvc, WwanSvc, RasMan, RasAuto,
//     DoSvc, lmhosts, WinHttpAutoProxySvc, BITS.
//   - Cuenta y Tienda (Game Pass, Xbox, Battle.net, etc.): TokenBroker, ClipSVC, wlidsvc, InstallService,
//     LicenseManager, WpnService, UserDataSvc, XblAuthManager, XblGameSave, XboxGipSvc, XboxNetApiSvc,
//     GameInputSvc, GraphicsPerfSvc, DisplayEnhancementService.
//   - Entrada y periféricos: hidserv, Bluetooth (bthserv, BthAvctpSvc, BTAGService, BluetoothUserService),
//     TabletInputService, WbioSrvc, PenService, FontCache.
//   - Restauración y actualizaciones: VSS, swprv, wuauserv, UsoSvc, WaaSMedicSvc, SDRSVC.
//   - Servicios que otras funciones de South Tweaks necesitan: defragsvc (TRIM de SSD), BDESVC (BitLocker),
//     FrameServer (cámaras), VaultSvc (Administrador de credenciales), DsmSvc (instalar dispositivos) y
//     SgrmBroker (System Guard / protecciones que usan los anticheats).
//
// Tampoco se incluyen WdiServiceHost ni WdiSystemHost: su clave pertenece a TrustedInstaller y ni\n// un administrador puede escribirla.\n// Por eso este ajuste es «Moderado» y avisa que el cambio se nota tras reiniciar.

var maxPerfServices = []string{
	"ADPSvc", "AarSvc", "AJRouter", "ALG", "AppIDSvc", "AppMgmt", "AppReadiness", "AppVClient", "ApxSvc",
	"AssignedAccessManagerSvc", "autotimesvc", "AxInstSV", "BcastDVRUserService", "Browser",
	"CaptureService", "cbdhsvc", "CDPSvc", "CDPUserSvc", "CloudBackupRestoreSvc", "cloudidsvc", "COMSysApp",
	"ConsentUxUserSvc", "CredentialEnrollmentManagerUserSvc", "CscService", "dcsvc", "DeviceAssociationBrokerSvc", "DevicePickerUserSvc", "DevicesFlowUserSvc", "DevQueryBroker",
	"diagnosticshub.standardcollector.service", "diagsvc", "DiagTrack", "DialogBlockingService",
	"DcpSvc", "dmwappushservice", "dot3svc", "DsSvc", "DusmSvc", "EapHost", "embeddedmode", "EntAppSvc",
	"Fax", "fdPHost", "FDResPub", "fhsvc", "hpatchmon", "HvHost",
	"icssvc", "IKEEXT", "InventorySvc", "IpxlatCfgSvc", "KtmRm", "LanmanServer", "lfsvc", "lltdsvc",
	"LocalKdc", "LxpSvc", "MapsBroker", "McmSvc", "McpManagementService", "MessagingService", "midisrv",
	"MixedRealityOpenXRSvc", "MSDTC", "MSiSCSI", "MsKeyboardFilter", "NaturalAuthentication", "NcaSvc",
	"NcbService", "NcdAutoSetup", "NetTcpPortSharing", "NPSMSvc", "OneSyncSvc", "p2pimsvc", "p2psvc",
	"P9RdrService", "PcaSvc", "PeerDistSvc", "perceptionsimulation", "PerfHost", "PhoneSvc",
	"PimIndexMaintenanceSvc", "pla", "PNRPAutoReg", "PNRPsvc", "PolicyAgent", "PrintDeviceConfigurationService",
	"PrintNotify", "PrintScanBrokerService", "PrintWorkflowUserSvc", "PushToInstall", "QWAVE", "refsdedupsvc",
	"RemoteAccess", "RemoteRegistry", "RetailDemo", "RmSvc", "RpcLocator", "SCardSvr", "ScDeviceEnum",
	"SCPolicySvc", "SEMgrSvc", "SensorDataService", "SensorService", "SensrSvc", "SessionEnv", "SharedAccess", "SharedRealitySvc", "shpamsvc", "SmsRouter", "SNMPTrap", "spectrum", "Spooler", "SSDPSRV",
	"ssh-agent", "SstpSvc", "StiSvc", "svsvc", "SysMain", "TapiSrv", "TieringEngineService",
	"TroubleshootingSvc", "tzautoupdate", "UdkUserSvc", "UevAgentService", "uhssvc", "UmRdpService",
	"UnistoreSvc", "upnphost", "VacSvc", "vmicguestinterface", "vmicheartbeat",
	"vmickvpexchange", "vmicrdv", "vmicshutdown", "vmictimesync", "vmicvmsession", "vmicvss", "WalletService",
	"WarpJITSvc", "wbengine", "wcncsvc", "WebClient", "Wecsvc",
	"WEPHOSTSVC", "wercplsupport", "WerSvc", "WFDSConMgrSvc", "whesvc", "WiaRpc", "WinRM", "wisvc",
	"wmiApSrv", "WMPNetworkSvc", "workfolderssvc", "WpcMonSvc", "WPDBusEnum", "WpnUserService",
	"WSAIFabricSvc", "WSearch", "wuqisvc",
}

// dedupeMaxPerf quita del ajuste de máximo rendimiento los servicios que ya controla otro ajuste.
// Si dos ajustes guardaran el «valor original» de la misma clave, revertirlos en distinto orden
// dejaría un valor equivocado; así cada clave tiene un solo dueño.
func dedupeMaxPerf(list []Tweak) {
	owned := map[string]bool{}
	mi := -1
	for i, t := range list {
		if t.ID == "svc_max_performance" {
			mi = i
			continue
		}
		for _, c := range t.Changes {
			owned[strings.ToLower(c.Root+`\`+c.Path+`\`+c.Name)] = true
		}
	}
	if mi < 0 {
		return
	}
	kept := make([]RegChange, 0, len(list[mi].Changes))
	for _, c := range list[mi].Changes {
		if !owned[strings.ToLower(c.Root+`\`+c.Path+`\`+c.Name)] {
			kept = append(kept, c)
		}
	}
	list[mi].Changes = kept
}

func maxPerfCatalog() []Tweak {
	names := maxPerfServices
	return []Tweak{
		{
			ID: "svc_max_performance", Title: "Deshabilitar todos los servicios opcionales (máximo rendimiento)", Category: "Servicios",
			Risk: "moderate", Admin: true, Reboot: true,
			Description: "Pone en «Deshabilitado» más de 150 servicios opcionales de Windows (impresión, Escritorio remoto, búsqueda, telemetría, mapas, Hyper-V invitado, etc.) para liberar RAM y CPU. Se dejan encendidos firewall, red, Wi-Fi, audio, Bluetooth, Windows Update, Tienda y Xbox para que nada de lo que usás jugando se rompa. Solo se tocan los que existen en tu Windows; al desactivarlo cada uno vuelve a su valor original. Los servicios que ya manejan los otros ajustes de esta categoría (telemetría, Xbox, búsqueda, impresión, informe de errores…) se activan desde esos ajustes: activalos también para cubrir todo. Necesita reiniciar.",
			Changes:     svcs(startDisabled, names...),
		},
		{
			ID: "bitlocker_off", Title: "Quitar BitLocker (descifrar los discos)", Category: "Acciones únicas", Admin: true, Once: true,
			Description: "Empieza a descifrar todas las unidades con BitLocker activo. Puede tardar un buen rato en segundo plano (podés seguir usando la PC) y deja tus archivos sin cifrar: si te roban la PC, quien la tenga puede leer el disco. Para volver a cifrar tenés que activarlo a mano desde Windows.",
		},
	}
}
