# 00 — Estado del proyecto y auditoría inicial

Fecha: 2026-11-08. Alcance: inspección de solo lectura del repositorio y de la PC de desarrollo.
No se modificó el sistema ni el código durante esta investigación.

## 1. Estado de Git

| Rama | Último commit | Contenido |
|---|---|---|
| `main` (remoto) | `ae42af8` | Fase 1 en Python/PySide6: detección, monitoreo, perfiles reversibles, modo simulación, 70 pruebas. |
| `feat/benchmark` (remoto, otro colaborador) | `c5f3bd5` | **Integración real con PresentMon 2.6.0** (Python): captura, métricas, comparación A/B con bootstrap, informes, CLI, `CLAUDE.md` con reglas del proyecto. |
| `go-rewrite` (local y remoto) | `72759b5` local, `295e35f` remoto | Reescritura en Go + Wails. **1 commit sin subir** (`72759b5`). |

Hallazgo crítico: **hay dos líneas de desarrollo divergentes.** La rama `go-rewrite` reemplazó todo el
código Python y, con él, **no incorpora** el sistema de benchmark de `feat/benchmark`. Además,
`go-rewrite` implementó varios ajustes que `docs/PERFORMANCE_PLAN.md` (en `main`) clasifica
explícitamente como **descartados** (ver §4). Antes de seguir hay que decidir cuál es la base
canónica (ver `10_IMPLEMENTATION_ROADMAP.md`, decisión D0).

## 2. `docs/PERFORMANCE_PLAN.md` (rama `main`)

Existe y es coherente con este encargo: "la prioridad 0 es el benchmark, no un ajuste". Propone:

- Métricas por sesión desde PresentMon (FPS promedio como frames/tiempo, 1 % y 0,1 % low como
  percentiles 99 y 99,9 del frametime, desvío y p99−p50, latencia si PresentMon la reporta,
  CPU/GPU-bound con `CPUBusy`/`GPUBusy`).
- Protocolo A/B: calentamiento, ≥3 capturas de 60 s, medianas con IC bootstrap.
- Nivel 1 de diagnóstico (EXPO/XMP, single channel, Hz del monitor, iGPU vs dGPU, ReBAR, disco).
- Lista de **descartadas**: desactivar seguridad, servicios en masa, HPET/dynamic tick,
  `TcpAckFrequency`/`TCPNoDelay`, limpiadores de RAM, `Win32PrioritySeparation`,
  `SystemResponsiveness`, `NetworkThrottlingIndex`, pagefile off, core parking/timer resolution, OC/UV.

Este documento es la línea base de rigor del proyecto. La investigación actual lo confirma en lo
esencial y lo amplía.

## 3. Integración con PresentMon

| Rama | Estado |
|---|---|
| `feat/benchmark` | **Implementada y verificada** con PresentMon 2.6.0 (SHA-256 y firma Intel comprobadas por el colaborador). Parser 1.x/2.x, métricas, A/B con bootstrap (veredictos: insuficiente / sin diferencia / mejora / empeora con umbral 2 %), almacenamiento JSON + CSV crudo, CLI, script guiado para Valorant. Falta: baseline real de Valorant y pantalla en la UI. |
| `go-rewrite` | **No existe.** Las únicas menciones son textos ("los FPS no se miden: requiere PresentMon"). La app en Go no mide FPS, frametimes ni latencia. |

Consecuencia: hoy la app en Go **no puede demostrar** que ningún ajuste funciona. Todo el catálogo
actual está en estado "impacto no medido".

## 4. Auditoría de lo implementado en `go-rewrite` contra la evidencia

Resumen (el detalle por ajuste está en `06_OPTIMIZATION_MATRIX.csv`, columna `estado_catalogo_actual`):

| Área | Qué hay | Problema según este encargo |
|---|---|---|
| Motor | backup → aplicar → verificar → rollback; simulación; registro de cambios; servicios y tareas vía registro/PowerShell con validación | Sólido. Falta medición, consentimiento por sesión y helper elevado separado (la app entera corre elevada cuando se usa admin). |
| Catálogo (52 ajustes) | Registro, plan de energía, servicios, tareas, loggers | **Mezcla rendimiento con privacidad y cosmética** sin distinguirlo. Incluye ajustes que la evidencia no respalda: `net_throttle` (NetworkThrottlingIndex + SystemResponsiveness=0), `games_priority`, `mmcss_extra`, `foreground_boost`, `svchost_split`, `autologgers`, `fullscreen_opt` global, `nv_pstate`. |
| Perfiles | Listas fijas (Gaming, sin admin, Privacidad) | **Viola la Parte 13** ("los perfiles no deben ejecutar listas fijas"). El perfil Gaming aplica 19 ajustes sin evaluar hardware ni medir. |
| Limpieza | 6 categorías, **caché de shaders seleccionada por defecto** | **Viola la Parte 7** ("no borrar cachés de shaders de forma rutinaria"). |
| Salud | Puntaje con pesos fijos (−20, −25, −15…) | **Viola la Parte 12** ("no utilizar ponderaciones arbitrarias sin documentarlas"). |
| Textos | Botón "Optimizar con un clic" | Sugiere una mejora no demostrada. |
| Telemetría | CPU (PDH), GPU NVIDIA (`nvidia-smi`), discos | Correcta y honesta (no muestra temperatura de CPU falsa). Falta: razones de throttling, frecuencia efectiva por núcleo, AMD/Intel GPU. |
| Detector de juegos | Por carpeta de instalación; sesiones con promedios | Útil como base; sin FPS no sirve para evaluar optimizaciones. |
| Pruebas | 27 pruebas Go (round-trip real HKCU, admin opcional, catálogo, limpieza, inicio, telemetría, juegos) | Buenas para seguridad del motor; **ninguna valida impacto de rendimiento** (no hay benchmark). |

Nada de lo anterior se modificó: el encargo pide no tocar lo ya hecho sin autorización. Las
correcciones propuestas están en `10_IMPLEMENTATION_ROADMAP.md`, fase F0.

## 5. Diagnóstico de solo lectura de la PC de desarrollo

Datos obtenidos con WMI, registro y `nvidia-smi` (solo lectura). Sirven como **caso real** de por qué
el diagnóstico rinde más que los tweaks:

| Componente | Observado | Interpretación | Nivel |
|---|---|---|---|
| CPU | Intel Core i3-9100F, 4 núcleos / 4 hilos, base 3,6 GHz, ~4,0 GHz efectivos | Sin SMT. Muy probable **CPU-bound** con RTX 2060 en juegos competitivos a baja resolución. A confirmar con `CPUBusy`/`GPUBusy`. | — |
| RAM | **3 módulos: 4 GB (canal A) + 4 GB y 8 GB (canal B)**, marcas distintas, todos a **2133 MT/s** | Configuración asimétrica (modo "flex": solo una parte en doble canal). Los módulos Crucial `CT4G4DFS824A` son de 2400 y el ADATA de 3200, y la plataforma (i3 9.ª gen + B365) admite hasta 2400 según la especificación de Intel (**verificar en Intel ARK**). Oportunidad: 2400 MT/s en BIOS y, a futuro, 2×8 GB iguales. | 4 (manual) |
| Monitor | **60 Hz activos, 75 Hz máximos reportados**, 1360×768 | El monitor puede ir a 75 Hz y está a 60. Es probablemente la mejora de fluidez percibida más barata de esta PC. | 1 |
| GPU | RTX 2060 12 GB, driver 596.49, PCIe 3.0 x16 (máximo de la placa), BAR1 = 256 MiB | ReBAR **no aplica**: la RTX 2060 (Turing) no lo soporta. Ejemplo de filtro de compatibilidad. | — |
| Seguridad | VBS corriendo y HVCI (integridad de memoria) activa | Se mantiene activado (regla de seguridad). Solo se documenta el costo posible. | — |
| HAGS | Activado (`HwSchMode=2`) | Valor por defecto razonable; impacto a medir. | 2 |
| Modo Juego | Activado | Por defecto de Windows. | 1 |
| Plan de energía | **"Paragon Core Power Plan"** (plan personalizado instalado por un tercero) | Plan de energía desconocido y no documentado en el sistema. Recomendación: auditarlo o compararlo contra "Equilibrado"/"Alto rendimiento" con benchmark. | 2 |
| Estabilidad | 0 eventos WHEA y 0 eventos 4101 (reinicio de driver de video) en 30 días | Sin señales de inestabilidad de hardware o driver. | — |
| BIOS / placa | ASUS PRIME B365M-A, BIOS 1.3.8 (2021) | Verificar si hay versión posterior (solo recomendación manual). | 4 |
| Red | Ethernet Realtek 1 Gbps | Base buena; diagnóstico de jitter/pérdida pendiente. | — |

Lo más valioso que FPS BOOSTER PRO podría hacer **en esta PC concreta** no es un ajuste de registro:
es (1) detectar los 60 Hz de un monitor de 75 Hz, (2) detectar la RAM asimétrica a 2133 y (3) medir
si el juego está limitado por CPU para orientar la configuración del juego (resolución, límite de FPS,
Reflex). Esto confirma la tesis del encargo y de `PERFORMANCE_PLAN.md`.

## 6. Límites de esta investigación (honestidad metodológica)

- La verificación de fuentes se hizo con un buscador web. En la mayoría de los casos se leyeron
  **extractos** de las páginas oficiales, no el documento completo. Cada afirmación lleva un nivel de
  verificación (ver `06_OPTIMIZATION_MATRIX.csv`, columna `fuente`, y la tabla de fuentes en cada documento).
- Las afirmaciones marcadas **NV** provienen de conocimiento técnico previo y **no** fueron verificadas
  en esta sesión. No deben usarse para implementar sin verificación.
- No se ejecutó ningún benchmark: todas las cifras de impacto son **hipótesis** o resultados de terceros
  citados como tales.
- No se descargó ni ejecutó software externo.
