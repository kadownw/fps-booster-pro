# 04 — Investigación de red para juegos y módulo NETWORK DIAGNOSTICS

Convenciones: [OF] oficial consultado · [SEC] secundario consultado · [NV] no verificado.

## 1. Principios

1. **La red no cambia los FPS.** Los problemas de red se perciben como "tirones" (rubber-banding,
   teletransportes, disparos que no registran), pero el frametime puede estar perfecto. La app debe
   separar ambos diagnósticos: si PresentMon muestra frametimes estables y el usuario reporta tirones,
   la hipótesis pasa a la red.
2. **Medir antes de tocar.** Ningún parámetro de red se modifica sin una medición previa y posterior.
3. **No prometer ping menor por DNS.** El DNS solo resuelve nombres; no cambia la ruta hacia el servidor
   del juego una vez conectado [NV — conocimiento general de redes].

## 2. Métricas y cómo medirlas

| Métrica | Definición | Medición desde la app | Notas |
|---|---|---|---|
| Ping / RTT | Ida y vuelta | ICMP a servidores de referencia; muchos servidores de juego no responden ICMP → usar la cifra del propio juego cuando la expone | Solo referencia |
| Jitter | Variación de RTT entre muestras | Desvío/diferencias sucesivas sobre ≥100 muestras | Más dañino que un ping alto estable |
| Pérdida | % de paquetes perdidos | Conteo de respuestas faltantes | Distinguir pérdida local (Wi-Fi) de remota |
| Bufferbloat | Aumento de latencia **bajo carga** (subida/bajada saturadas) | Ping en reposo vs ping durante una prueba de velocidad | Waveform califica A+ con < 5 ms y A con < 30 ms adicionales [SEC]; las pruebas web agregan ruido del navegador [SEC] |
| Calidad Wi-Fi | RSSI, tasa de enlace, canal, banda | `netsh wlan show interfaces` (solo lectura) | Wi-Fi 6/6E/7 reducen congestión, pero la interferencia domina [NV] |
| Ruta | Saltos y dónde aparece la pérdida | `tracert`/`pathping` (solo lectura) | Pérdida en saltos intermedios que no se propaga no es un problema |

## 3. Factores investigados (Parte 8)

| # | Tema | Hecho | Acción de la app | Nivel |
|---|---|---|---|---|
| 1–3 | Ping, jitter, pérdida | Ver §2 | Medir y explicar | Diagnóstico |
| 4 | Bufferbloat | Colas grandes en el router/módem bajo carga | Prueba guiada y recomendación de **SQM en el router** (fq_codel o CAKE) configurado un poco por debajo de la velocidad contratada [SEC]. CoDel tolera hasta 5 ms de cola sobre un intervalo de 100 ms (RFC 8289) [SEC] | N4 (router) |
| 5 | Ethernet vs Wi-Fi | Ethernet elimina interferencia y roaming | Detectar tipo de enlace (PC de desarrollo: Ethernet 1 Gbps) | N1 (recomendación) |
| 6–8 | Wi-Fi 5/6/6E/7, interferencias, señal | Canal congestionado y señal baja → jitter/pérdida | Diagnóstico de RSSI y canal | N2 |
| 9–10 | Drivers y adaptadores | Drivers viejos o con errores | Mostrar versión; recomendar actualizar desde el fabricante | N2 |
| 11–12 | QoS / SQM del router | Prioriza o controla colas | Guía manual | N4 |
| 13–14 | Saturación de subida/bajada | Descargas (Steam, Windows Update, P2P) durante el juego | Detectar tráfico alto de otros procesos durante sesiones (contadores por proceso [NV]) | N2 |
| 15–16 | Selección de servidor y rutas | Región incorrecta = RTT alto | Recomendar región en el juego | Manual |
| 17 | DNS | Afecta resolución de nombres, no el RTT de juego | Nunca prometer mejora de ping | N5 (como mejora de ping) |
| 18 | MTU | Valores incorrectos causan fragmentación | Solo diagnóstico (prueba de MTU de ruta); no cambiar por defecto | N3 |
| 19 | TCP (Nagle, ACK) | Solo TCP; ver `02 §4.9` | No modificar | N5 |
| 20 | UDP | Transporte habitual del tráfico de juego | — | — |
| 21 | Ahorro de energía del adaptador (EEE, suspensión) | Puede introducir latencia/desconexiones en algunos adaptadores | Diagnóstico; cambio por propiedad avanzada **N3** con medición | N3 |
| 22 | Roaming Wi-Fi | Cambios de punto de acceso causan cortes | Diagnóstico | N2 |
| 23 | Congestión de canales | — | Diagnóstico | N2 |
| 24 | Desconexiones | Eventos de NIC/Wi-Fi en el registro de eventos | Leer eventos de red (solo lectura) | Diagnóstico |

## 4. Módulo NETWORK DIAGNOSTICS (diseño)

1. **Inventario:** adaptadores activos, tipo (Ethernet/Wi-Fi), velocidad de enlace, driver, banda y RSSI.
2. **Prueba en reposo (60 s):** RTT/jitter/pérdida a 2–3 destinos estables (gateway local + destinos públicos).
3. **Prueba bajo carga (opcional, con consentimiento: consume datos):** repetir mientras se satura la
   conexión (o pedir al usuario que corra una prueba de velocidad) → bufferbloat.
4. **Localización:** si la pérdida aparece ya en el gateway → problema local (Wi-Fi/cable/adaptador);
   si aparece después → proveedor/ruta.
5. **Correlación con sesiones de juego:** si el juego expone ping/pérdida en archivos o API oficial, se
   importa; nunca se lee la memoria del juego.
6. **Recomendaciones:** Ethernet, canal/banda Wi-Fi, SQM en router, pausar descargas, región del servidor.
   Ningún cambio de registro TCP.

**Datos y privacidad:** las pruebas contra servidores externos revelan la IP pública del usuario a esos
servidores; deben ser explícitas y opcionales.

## Fuentes consultadas en esta sesión

- KB 328890 (TcpAckFrequency): https://support.microsoft.com/kb/328890
- Bufferbloat / Waveform / SQM (secundarias): https://forum.openwrt.org/t/2025-current-recommendations-for-bufferbloat-testing/241303 · https://ipfyi.com/scenario/bufferbloat-under-load/
- Pendiente de verificar en fuente primaria: RFC 8289 (CoDel) y RFC 8290 (FQ-CoDel) en rfc-editor.org.
