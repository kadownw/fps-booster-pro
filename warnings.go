package main

// Avisos de peligro: ajustes que desactivan algo que el usuario puede necesitar. La interfaz muestra un
// triángulo de advertencia en la tarjeta y, al pasar el mouse, este texto: QUÉ se apaga y QUÉ se pierde.
// No son errores ni ajustes «malos»: son cambios que conviene entender antes de aplicar.

var dangerNotes = map[string]string{
	"svc_max_performance":       "Deshabilita más de 140 servicios de Windows: impresión, Escritorio remoto, búsqueda e indexado, sensores, fax, telefonía, Hyper-V invitado, compartir archivos en red y más. Si usás alguna de esas funciones deja de andar hasta que lo revertís. El efecto se nota después de reiniciar.",
	"bitlocker_off":             "Quita el cifrado de tus discos. Si te roban la PC o el disco, cualquiera puede leer tus archivos. No se deshace desde la app: para volver a cifrar hay que activarlo a mano en Windows.",
	"prevent_device_encryption": "Windows deja de cifrar el disco automáticamente. Tus archivos quedan legibles si alguien se lleva el disco.",
	"updates_pause":             "Retrasa hasta 35 días los parches de seguridad de Windows. Mientras dure la pausa, las vulnerabilidades nuevas quedan abiertas.",
	"exclude_wu_drivers":        "Windows Update deja de instalar drivers, incluidos los que corrigen fallas de seguridad o estabilidad. Tenés que actualizarlos vos desde cada fabricante.",
	"maintenance_off":           "Windows deja de hacer solo el TRIM de los SSD, los diagnósticos y las actualizaciones de apps programadas. Con el tiempo el disco puede rendir menos si no lo hacés a mano.",
	"hibernate_off":             "Apaga la hibernación y el Inicio rápido. En una notebook, si la batería se agota con la tapa cerrada, se pierde lo que tenías abierto.",
	"memory_compression_off":    "Con poca RAM libre, Windows pasa a usar más el disco (archivo de paginación) y todo puede ir más lento. Conviene solo con 16 GB o más.",
	"nv_hdcp_off":               "Netflix, Disney+, Prime Video y otros servicios con protección de contenido pueden no reproducir, mostrar pantalla negra o bajar a baja resolución.",
	"location_off":              "Mapas, clima, zona horaria automática y «Encontrar mi dispositivo» dejan de saber dónde estás.",
	"find_my_device":            "Si perdés o te roban la PC no vas a poder ubicarla ni bloquearla desde tu cuenta Microsoft.",
	"bg_apps":                   "Correo, Calendario, Teams, Mensajes y otras apps de la Tienda dejan de recibir notificaciones cuando están cerradas.",
	"store_autoupdate_off":      "Las apps de la Tienda, incluidos los juegos de Game Pass y Gaming Services, no se actualizan solas. Algunos juegos online no abren hasta que los actualizás.",
	"ntfs_lastaccess_off":       "Algunos programas de backup, antivirus o limpieza usan la fecha de último acceso para decidir qué copiar o borrar, y pueden equivocarse.",
	"gpu_msi_mode":              "Con drivers de video que no lo soportan bien puede haber pantalla negra al arrancar. Se revierte desde la app o desde el Modo seguro de Windows.",
	"remove_bloat_apps":         "Desinstala apps de todos los usuarios. No se deshace desde la app: para volver a tenerlas hay que instalarlas desde la Microsoft Store.",
	"remove_gamebar_app":        "Quita la Xbox Game Bar: se pierden la grabación de clips, las capturas con Win+G y los widgets de rendimiento. Se reinstala desde la Tienda.",
	"remove_copilot_app":        "Desinstala Copilot. No se deshace desde la app: se reinstala desde la Microsoft Store.",
	"svc_xbox_manual":           "Si un juego de Game Pass no arranca o no sincroniza partidas, revertí este ajuste: los servicios de Xbox quedan en inicio manual.",
	"fix_valorant_hvci":         "La Integridad de memoria puede impedir que arranquen drivers viejos (de periféricos, RGB o anticheats antiguos) y tiene un costo de rendimiento chico.",
}

// applyWarnings copia cada aviso a su ajuste del catálogo.
func applyWarnings(list []Tweak) {
	for i := range list {
		if w, ok := dangerNotes[list[i].ID]; ok {
			list[i].Warn = w
		}
	}
}
