# 10 — Plan de implementación por fases

**Nada de este plan está implementado.** Todo cambio de código requiere autorización previa (encargo,
"Antes de modificar código, explicá el plan y solicitá autorización").

## 1. Decisiones que necesita tomar el equipo antes de programar

| ID | Decisión | Opciones | Recomendación técnica |
|---|---|---|---|
| **D0** | ¿Cuál es la base canónica? | (a) Go + Wails (`go-rewrite`): exe único, UI nueva, motor con servicios/tareas, sin benchmark. (b) Python + PySide6 (`main` + `feat/benchmark`): benchmark con PresentMon verificado, reglas del colaborador. | **(a) con port del benchmark de `feat/benchmark` a Go**, respetando sus fórmulas, umbrales y reglas (`CLAUDE.md`). Alternativa válida: (a) para la UI + llamar al módulo Python de benchmark como proceso externo durante la transición. Coordinar con el autor de `feat/benchmark` antes de mergear nada. |
| **D1** | ¿Qué hacer con los ajustes sin evidencia ya implementados? | Retirar / ocultar / dejar como "experimentales" | Sacarlos de los perfiles y moverlos a una sección **Experimental** (no se aplican sin experimento A/B). Ver F0. |
| **D2** | Modo por defecto | Real / Simulación / Diagnóstico | **Diagnóstico (solo lectura)** en la primera ejecución, como exige el `CLAUDE.md` de `feat/benchmark`. |
| **D3** | ¿Driver de kernel para temperatura de CPU? | No / PawnIO opcional | **No por ahora**; opcional más adelante con consentimiento explícito. |

## 2. Fases

### F0 — Correcciones de coherencia (sin funciones nuevas) · pequeño
1. Perfiles: quitar `net_throttle`, `mmcss_extra`, `foreground_boost`, `fullscreen_opt` (global), `usb_recovery`,
   `power_throttle`, `nv_pstate`, `amd_ulps`, `games_priority` del perfil Gaming (pasan a Experimental).
2. Limpieza: **deseleccionar la caché de shaders por defecto** y explicar cuándo tiene sentido.
3. Separar el catálogo en **Rendimiento / Preferencias de interfaz / Privacidad / Experimental**, con la
   etiqueta de nivel (N1–N5) y evidencia de la matriz 06.
4. Textos: "Optimizar con un clic" → "Aplicar perfil"; ninguna promesa de FPS.
5. Salud: documentar los pesos actuales o reemplazarlos por la lista de hallazgos sin puntaje.
6. Modo por defecto según D2.

### F1 — Medición (habilitador) · grande
1. Port de `feat/benchmark` (o integración como proceso): localizar PresentMon, verificar hash y firma,
   capturar, parsear 1.x/2.x, métricas de 07, almacenamiento JSON + CSV crudo.
2. Pantalla **Benchmarks**: capturar, listar, comparar A/B con veredicto y explicación.
3. Detector de juegos → disparo opcional de captura al iniciar sesión.
4. FrameType: separar FPS del motor y generados.

### F2 — Diagnóstico de alto valor (solo lectura) · mediano
Monitor Hz (DIAG-001), RAM velocidad/canales (DIAG-002/003), iGPU (DIAG-004), cuello CPU/GPU (DIAG-005),
throttling GPU (DIAG-006), frecuencia por núcleo (DIAG-007), estabilidad (DIAG-008), drivers (DIAG-009),
parche Zen (DIAG-010), X3D (DIAG-011), plan de terceros (DIAG-012), VBS informativo (DIAG-013), disco
(DIAG-014), overlays (DIAG-015), presión de memoria (DIAG-017), VRAM (DIAG-018), PCIe (DIAG-019).

### F3 — Motor de experimentos · grande
`experiment` (A/B guiado, orden, reinicios, veredicto, reversión automática ante regresión), utilidad con
pesos de 09, perfiles como objetivos, KB versionada v1, restauración por sesión, diario de transacciones.

### F4 — Latencia y red · mediano
LATENCY LAB (03) y NETWORK DIAGNOSTICS (04).

### F5 — Juegos y privilegios · grande
Fichas verificadas de Valorant, CS2 y Fortnite; presets de configuración con backup (juego cerrado);
helper elevado con lista blanca; firma del ejecutable.

## 3. Las 20 oportunidades más prometedoras

Orden por (impacto esperado × probabilidad de aplicar × evidencia) ÷ (riesgo × costo). Los primeros puestos
son diagnósticos porque **en equipos reales los mayores problemas suelen ser de configuración**, como
muestra la PC de desarrollo.

| # | ID | Oportunidad | Por qué | Nivel |
|---|---|---|---|---|
| 1 | BENCH-001 | Benchmark con PresentMon | Sin medición no hay optimización demostrable | 1 |
| 2 | DIAG-005 | Clasificador CPU/GPU-bound | Decide qué recomendaciones tienen sentido | 1 |
| 3 | DIAG-001 | Monitor por debajo de su Hz máximo | Mejora percibida inmediata; confirmado en la PC de desarrollo (60 de 75 Hz) | 1 |
| 4 | DIAG-002/003 | RAM: velocidad y canales | Confirmado en la PC de desarrollo (2133, asimétrica); alto impacto en CPU-bound | 4 (manual) |
| 5 | GPU-001/002 | Reflex / Anti-Lag 2 en el juego | Latencia: efecto grande y documentado | 1 |
| 6 | GPU-005 | Límite de FPS (con VRR) | Mejora 1 % low y latencia; sin riesgo | 2 |
| 7 | GAME-001 | Configuración del juego según cuello y hardware | La mayor palanca de FPS está dentro del juego | 2 |
| 8 | DIAG-004 / WIN-005 | Juego en iGPU / preferencia de GPU | Enorme cuando ocurre (portátiles) | 2 |
| 9 | DIAG-006 | Throttling térmico/potencia de GPU | Explica caídas en sesiones largas | 1 |
| 10 | DIAG-008 | Estabilidad (WHEA, TDR, cierres) | Rendimiento sostenible (Parte 22) | 1 |
| 11 | DIAG-011 | X3D dos CCD: driver/estacionamiento | Pérdidas grandes cuando falla | 2 |
| 12 | DIAG-010 | Parche Zen (KB5041587/24H2) | Hasta ~11 % reportado en Zen 5 | 2 |
| 13 | DIAG-009 | Drivers de GPU/chipset | Correcciones de rendimiento y estabilidad | 2 |
| 14 | WIN-003 | Optimizaciones para juegos en ventana | Menor latencia en sin bordes (oficial) | 1 |
| 15 | DIAG-012 / WIN-006 | Plan de energía (incluye planes de terceros) | Confirmado plan de terceros en la PC de desarrollo; impacto a medir | 2 |
| 16 | GPU-004 | Reporte honesto de FPS generados | Evita engañar al usuario | 1 |
| 17 | DIAG-018 | Saturación de VRAM | Causa típica de tirones en AAA | 2 |
| 18 | DIAG-015 | Overlays/capturas activas | Costo oculto frecuente | 2 |
| 19 | NET-001 | Diagnóstico de red | Separa "tirones" de red de problemas gráficos | 1 |
| 20 | LAT-001 | LATENCY LAB | Configuración de menor latencia estable por equipo | 2 |

## 4. Primeras cinco investigaciones experimentales (PC de desarrollo: i3-9100F, RTX 2060 12 GB, 16 GB DDR4-2133 asimétrica, monitor 75 Hz a 60 Hz)

Todas requieren F1 (benchmark) y siguen el protocolo de 07. Juego sugerido: **Valorant** (script guiado ya
existe en `feat/benchmark`) y **CS2**. Ningún experimento se ejecuta sin autorización explícita.

### E1 — Refresco del monitor: 60 Hz → 75 Hz
- **Hipótesis:** aumenta la fluidez percibida y reduce la latencia de presentación cuando el juego supera 60 FPS.
- **Mecanismo:** más escaneos por segundo; menos tiempo hasta mostrar.
- **Cambio:** modo de pantalla (reversible, la app guarda el modo original).
- **Métricas:** frames mostrados/s, Present→mostrado (PresentMon), desvío de frametime; con V-Sync, FPS limitados a 60 vs 75.
- **Éxito:** frames mostrados ↑ ~25 % (si FPS > 75) y latencia de presentación ↓, sin regresión de 1 % low.
- **Riesgo:** bajo (algunos monitores en 75 Hz por HDMI pueden fallar → revertir automático a los 15 s sin confirmación).

### E2 — Plan de energía: "Paragon Core Power Plan" vs "Equilibrado" vs "Alto rendimiento"
- **Hipótesis:** en un i3 de 4 hilos CPU-bound, el plan puede cambiar la frecuencia efectiva y el 1 % low; el plan de terceros podría no aportar nada.
- **Mecanismo:** mínimos de estado del procesador, estacionamiento, boost.
- **Métricas:** frecuencia efectiva por núcleo, FPS, 1 % low, CPUBusy, potencia.
- **Diseño:** tres brazos; A-B-C repetido dos veces; sin reinicio entre brazos.
- **Riesgo:** bajo; reversión trivial.

### E3 — HAGS activado vs desactivado (RTX 2060, CPU débil)
- **Hipótesis:** HAGS reduce algo de carga de CPU del scheduling de GPU; en CPU-bound podría mejorar levemente el 1 % low. Resultado previo de terceros: variable.
- **Métricas:** CPUBusy, FPS, 1 % low, latencia de presentación.
- **Diseño:** AAA-BBB con reinicio en ambos lados.
- **Riesgo:** medio-bajo (reinicio; drivers).
- **Uso del resultado:** decide si `hags` queda N2 "recomendado en este tipo de equipo" o "sin efecto".

### E4 — Validar o descartar el ajuste global `fullscreen_opt` del catálogo actual
- **Hipótesis nula esperada:** desactivar FSO globalmente no mejora 1 % low ni latencia en juegos modernos y puede romper VRR/overlays.
- **Métricas:** modo de presentación reportado por PresentMon, latencia de presentación, 1 % low.
- **Uso del resultado:** si no hay mejora demostrada → se retira definitivamente (N5). Sirve como plantilla para auditar el resto de los ajustes "experimentales" del catálogo.

### E5 — RAM a 2400 MT/s (cambio manual en BIOS por el usuario)
- **Hipótesis:** en un CPU-bound con RAM asimétrica a 2133, subir a 2400 (velocidad nominal de los módulos Crucial y máxima de la plataforma según Intel — **verificar en ARK**) mejora 1 % low.
- **Cambio:** manual (Nivel 4) con guía; la app solo detecta y mide.
- **Estabilidad:** checklist de 07 §6 antes de aceptar.
- **Extensión:** documentar el efecto esperado de pasar a 2×8 GB iguales (doble canal completo) como recomendación de hardware, sin promesas numéricas.

## 5. Clasificación final (Parte 25) — resumen por nivel

| Nivel | IDs de la matriz |
|---|---|
| **N1 — Seguras y comprobadas** | BENCH-001, DIAG-001, 005, 006, 007, 008, 012, 013, 016, 017, WIN-001, 002, 003, GPU-001, 002, NET-001, 002, STO-002 (como espacio), preferencias de entrada/UI (LAT-002/003, UX-001/002), privacidad (PRIV-001) como categoría separada |
| **N2 — Dependientes del hardware** | DIAG-004, 009, 010, 011, 014, 015, 018, 019, WIN-004, 005, 006, 008, 009, 010, 011, CPU-006, GPU-003, 004, 005, STO-001, LAT-001, 004, UX-003, SYS-001, GAME-001 |
| **N3 — Experimentales** | WIN-007, REG-002, REG-003, CPU-001, 004, 005, GPU-007, 008, NET-005, SVC-001, 002, 005, 006, SYS-002 |
| **N4 — Manuales avanzadas** | DIAG-002, 003, RAM-003, CPU-007, SEC-001 (decisión informada), NET-003 |
| **N5 — Descartadas** | SEC-002, REG-001, 004, 005, 006, 007, 008, 009, 010, 011, 012, CPU-002, 003, GPU-006, GPU-009 (rutinario), RAM-001, 002, NET-004, SVC-003, 004, 007, SYS-003 |

## 6. Cómo retomar en otra sesión

1. Leer `00_ESTADO_DEL_PROYECTO.md` (estado y decisiones pendientes D0–D3).
2. Elegir fase; cada fase tiene criterios de "hecho": pruebas en verde + invariantes de 08 + documentación actualizada.
3. Verificar fuentes marcadas [NV] antes de implementar el ítem que las use; actualizar la matriz 06.
4. No mezclar en un mismo commit correcciones de F0 con funciones nuevas.
