package main

import "strings"

// Servicios en Â«mÃ¡ximo rendimientoÂ»: deshabilita casi todos los servicios opcionales de Windows.
// Se parte de la lista del script Â«17 ServicesÂ» de Ultimate (230 servicios con Start=4) y se
// QUITAN los que, apagados, rompen cosas que un jugador usa a diario. El cambio es reversible:
// cada valor Start se guarda antes de tocarlo y Â«DesactivarÂ» lo devuelve tal cual estaba.
//
// Se dejan SIEMPRE encendidos (no estÃ¡n en la lista):
//   - Seguridad y base del sistema: BFE y mpssvc (firewall), EventLog, EventSystem, PlugPlay, SamSs,
//     DPS, Themes, ShellHWDetection, StorSvc, TrkWks, vds, smphost, W32Time, CertPropSvc, EFS.
//   - Red y conexiÃ³n: NlaSvc, iphlpsvc, LanmanWorkstation, Netlogon, WlanSvc, WwanSvc, RasMan, RasAuto,
//     DoSvc, lmhosts, WinHttpAutoProxySvc, BITS.
//   - Cuenta y Tienda (Game Pass, Xbox, Battle.net, etc.): TokenBroker, ClipSVC, wlidsvc, InstallService,
//     LicenseManager, WpnService, UserDataSvc, XblAuthManager, XblGameSave, XboxGipSvc, XboxNetApiSvc,
//     GameInputSvc, GraphicsPerfSvc, DisplayEnhancementService.
//   - Entrada y perifÃ©ricos: hidserv, Bluetooth (bthserv, BthAvctpSvc, BTAGService, BluetoothUserService),
//     TabletInputService, WbioSrvc, PenService, FontCache.
//   - RestauraciÃ³n y actualizaciones: VSS, swprv, wuauserv, UsoSvc, WaaSMedicSvc, SDRSVC.
//
// Tampoco se incluyen WdiServiceHost ni WdiSystemHost: su clave pertenece a TrustedInstaller y ni\n// un administrador puede escribirla.\n// Por eso este ajuste es Â«ModeradoÂ» y avisa que el cambio se nota tras reiniciar.

var maxPerfServices = []string{
	"ADPSvc", "AarSvc", "AJRouter", "ALG", "AppIDSvc", "AppMgmt", "AppReadiness", "AppVClient", "ApxSvc",
	"AssignedAccessManagerSvc", "autotimesvc", "AxInstSV", "BcastDVRUserService", "BDESVC", "Browser",
	"CaptureService", "cbdhsvc", "CDPSvc", "CDPUserSvc", "CloudBackupRestoreSvc", "cloudidsvc", "COMSysApp",
	"ConsentUxUserSvc", "CredentialEnrollmentManagerUserSvc", "CscService", "dcsvc", "defragsvc",
	"DeviceAssociationBrokerSvc", "DevicePickerUserSvc", "DevicesFlowUserSvc", "DevQueryBroker",
	"diagnosticshub.standardcollector.service", "diagsvc", "DiagTrack", "DialogBlockingService",
	"dmwappushservice", "dot3svc", "DsmSvc", "DsSvc", "DusmSvc", "EapHost", "embeddedmode", "EntAppSvc",
	"Fax", "fdPHost", "FDResPub", "fhsvc", "FrameServer", "FrameServerMonitor", "hpatchmon", "HvHost",
	"icssvc", "IKEEXT", "InventorySvc", "IpxlatCfgSvc", "KtmRm", "LanmanServer", "lfsvc", "lltdsvc",
	"LocalKdc", "LxpSvc", "MapsBroker", "McmSvc", "McpManagementService", "MessagingService", "midisrv",
	"MixedRealityOpenXRSvc", "MSDTC", "MSiSCSI", "MsKeyboardFilter", "NaturalAuthentication", "NcaSvc",
	"NcbService", "NcdAutoSetup", "NetTcpPortSharing", "NPSMSvc", "OneSyncSvc", "p2pimsvc", "p2psvc",
	"P9RdrService", "PcaSvc", "PeerDistSvc", "perceptionsimulation", "PerfHost", "PhoneSvc",
	"PimIndexMaintenanceSvc", "pla", "PNRPAutoReg", "PNRPsvc", "PolicyAgent", "PrintDeviceConfigurationService",
	"PrintNotify", "PrintScanBrokerService", "PrintWorkflowUserSvc", "PushToInstall", "QWAVE", "refsdedupsvc",
	"RemoteAccess", "RemoteRegistry", "RetailDemo", "RmSvc", "RpcLocator", "SCardSvr", "ScDeviceEnum",
	"SCPolicySvc", "SEMgrSvc", "SensorDataService", "SensorService", "SensrSvc", "SessionEnv", "SgrmBroker",
	"SharedAccess", "SharedRealitySvc", "shpamsvc", "SmsRouter", "SNMPTrap", "spectrum", "Spooler", "SSDPSRV",
	"ssh-agent", "SstpSvc", "stisvc", "StiSvc", "svsvc", "SysMain", "TapiSrv", "TieringEngineService",
	"TroubleshootingSvc", "tzautoupdate", "UdkUserSvc", "UevAgentService", "uhssvc", "UmRdpService",
	"UnistoreSvc", "upnphost", "VacSvc", "VaultSvc", "vmicguestinterface", "vmicheartbeat",
	"vmickvpexchange", "vmicrdv", "vmicshutdown", "vmictimesync", "vmicvmsession", "vmicvss", "WalletService",
	"WarpJITSvc", "wbengine", "wcncsvc", "WebClient", "Wecsvc",
	"WEPHOSTSVC", "wercplsupport", "WerSvc", "WFDSConMgrSvc", "whesvc", "WiaRpc", "WinRM", "wisvc",
	"wmiApSrv", "WMPNetworkSvc", "workfolderssvc", "WpcMonSvc", "WPDBusEnum", "WpnUserService",
	"WSAIFabricSvc", "WSearch", "wuqisvc",
}

// dedupeMaxPerf quita del ajuste de mÃ¡ximo rendimiento los servicios que ya controla otro ajuste.
// Si dos ajustes guardaran el Â«valor originalÂ» de la misma clave, revertirlos en distinto orden
// dejarÃ­a un valor equivocado; asÃ­ cada clave tiene un solo dueÃ±o.
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
			ID: "svc_max_performance", Title: "Deshabilitar todos los servicios opcionales (mÃ¡ximo rendimiento)", Category: "Servicios",
			Risk: "moderate", Admin: true, Reboot: true,
			Description: "Pone en Â«DeshabilitadoÂ» mÃ¡s de 150 servicios opcionales de Windows (impresiÃ³n, Escritorio remoto, bÃºsqueda, telemetrÃ­a, mapas, Hyper-V invitado, etc.) para liberar RAM y CPU. Se dejan encendidos firewall, red, Wi-Fi, audio, Bluetooth, Windows Update, Tienda y Xbox para que nada de lo que usÃ¡s jugando se rompa. Solo se tocan los que existen en tu Windows; al desactivarlo cada uno vuelve a su valor original. Los servicios que ya manejan los otros ajustes de esta categorÃ­a (telemetrÃ­a, Xbox, bÃºsqueda, impresiÃ³n, informe de erroresâ€¦) se activan desde esos ajustes: activalos tambiÃ©n para cubrir todo. Necesita reiniciar.",
			Changes:     svcs(startDisabled, names...),
		},
		{
			ID: "bitlocker_off", Title: "Quitar BitLocker (descifrar los discos)", Category: "Acciones Ãºnicas", Admin: true, Once: true,
			Description: "Empieza a descifrar todas las unidades con BitLocker activo. Puede tardar un buen rato en segundo plano (podÃ©s seguir usando la PC) y deja tus archivos sin cifrar: si te roban la PC, quien la tenga puede leer el disco. Para volver a cifrar tenÃ©s que activarlo a mano desde Windows.",
		},
	}
}
