# FPS BOOSTER PRO — Análisis y propuesta de optimizaciones

Estado: propuesta para revisar (no implementada). Base: fase 1 (`docs/ARCHITECTURE.md`).

## 1. Regla número uno: medir antes de optimizar

Hoy toda recomendación dice "impacto no medido". Sin un sistema de medición, cualquier
optimización nueva es una apuesta. Por eso **la prioridad 0 es el benchmark**, no un ajuste.

### Métricas (por sesión de juego)

| Métrica | Definición usada | Fuente |
|---|---|---|
| FPS promedio | frames / tiempo total (no promedio de FPS instantáneos) | PresentMon `MsBetweenPresents` |
| 1% low | FPS equivalente al percentil 99 del frametime | ídem |
| 0.1% low | FPS equivalente al percentil 99,9 del frametime | ídem |
| Estabilidad | desvío estándar y p99 − p50 del frametime | ídem |
| Latencia (estimada) | `MsPCLatency` / `MsClickToPhotonLatency` si PresentMon la reporta; si no, "no disponible" | PresentMon 2.x |
| Limitado por | CPU busy vs GPU busy (`MsCPUBusy`, `MsGPUBusy`) | PresentMon 2.x |

Se documenta cuál definición de "1% low" se usa: hay dos en circulación y dan números distintos.

### Protocolo A/B

1. Detectar el juego en primer plano y su proceso.
2. Calentamiento de 20–30 s sin registrar (shaders, caché, temperatura).
3. Al menos 3 capturas de 60 s en la misma escena (o el benchmark integrado del juego).
4. Aplicar el cambio → repetir exactamente lo mismo.
5. Comparar **medianas** con intervalo de confianza (bootstrap). Si el intervalo cruza 0 →
   "sin diferencia medible". Solo entonces la app puede mostrar "+X FPS medido en tu PC".
6. Descartar la corrida si cambian la temperatura (throttling), la resolución, o el juego
   pasa a segundo plano.

## 2. Optimizaciones propuestas, ordenadas

Escala: Impacto esperado (A/M/B, **a confirmar con benchmark**), Riesgo (bajo/medio),
Reversión (trivial / con reinicio / manual guiada).

### Nivel 1 — Diagnóstico de alto impacto (solo lectura, sin riesgo)

Lo que más FPS "regala" suele ser una mala configuración, no un ajuste de registro.

| # | Detección | Impacto | Riesgo | Acción |
|---|---|---|---|---|
| 1 | RAM sin EXPO/XMP (`ConfiguredClockSpeed` < `Speed`) | A en juegos CPU-bound | ninguno | Guía para activarlo en BIOS (la app no toca BIOS) |
| 2 | RAM en single channel (1 módulo o slots mal usados) | A | ninguno | Aviso + guía |
| 3 | Monitor a menos Hz que su máximo (ej. 60 de 144) | A en fluidez percibida | bajo | Ofrecer cambiar Hz (reversible, ver #8) |
| 4 | Juego corriendo en la GPU integrada en PCs con iGPU + dGPU | A | ninguno | Ver #5 |
| 5 | Resizable BAR / SAM desactivado, driver de GPU viejo | M | ninguno | Aviso + guía |
| 6 | Disco del juego casi lleno o HDD | B (tiempos de carga, stutter) | ninguno | Aviso |

### Nivel 2 — Ajustes reversibles sin admin (HKCU / API de usuario)

| # | Ajuste | Mecanismo | Impacto | Riesgo | Reversión |
|---|---|---|---|---|---|
| 5 | GPU preferida por juego = alto rendimiento | `HKCU\Software\Microsoft\DirectX\UserGpuPreferences` (valor por exe) | A si había iGPU, nulo si no | bajo | trivial |
| 7 | Optimizaciones para juegos en ventana (flip model) | `HKCU\Software\Microsoft\DirectX\UserGpuPreferences\DirectXUserGlobalSettings` → `SwapEffectUpgradeEnable=1;` | M en latencia (borderless DX10/11) | bajo | trivial |
| 8 | Frecuencia de refresco al máximo soportado | `ChangeDisplaySettingsEx` (guardar modo original) | A (fluidez) | bajo | trivial |
| 9 | Prioridad "Alta" del proceso del juego mientras corre | `SetPriorityClass`, **nunca** Realtime | B | bajo | automática al cerrar el juego |
| 10 | Overlays (Game Bar ya está; detectar otros) | solo detección de procesos conocidos | B–M | ninguno | — (no se cierran sin permiso) |

### Nivel 3 — Ajustes por juego (archivos de configuración del juego)

En la práctica la mayor diferencia de FPS está **dentro del juego**: resolución, escalado,
sombras, límite de FPS. La app puede proponer un preset según el hardware y escribirlo
en el archivo de config, con backup completo del archivo.

| Juego | Archivo | Qué se puede ajustar | Notas |
|---|---|---|---|
| Fortnite | `%LOCALAPPDATA%\FortniteGame\Saved\Config\WindowsClient\GameUserSettings.ini` | modo de renderizado (DX11/DX12/Performance), calidad, límite de FPS, escalado | El juego debe estar cerrado al escribir |
| CS2 | `cs2_video.txt` en la carpeta del usuario de Steam | calidad, MSAA, escalado (FSR) | Opciones de lanzamiento: solo sugerir, no editar Steam |
| Valorant | `%LOCALAPPDATA%\VALORANT\Saved\Config\…\GameUserSettings.ini` | calidad, límite de FPS | Parte de la config se sincroniza con la cuenta; Vanguard: **no tocar binarios ni procesos**, solo el ini |
| Otros | datos en `games/*.json` | según el juego | Se agregan sin código nuevo |

Reglas: escribir solo con el juego cerrado; backup del archivo entero; verificar que el juego
arranca (preguntar al usuario) antes de dar el cambio por bueno; nunca inyectar ni leer memoria
del juego (anti-cheat).

Límite de FPS: con VRR (FreeSync/G-Sync) el tope unos FPS por debajo del refresco suele
**mejorar frametimes y latencia** más que cualquier tweak. Se propone con el limitador del juego.

### Nivel 4 — Requieren admin (fase 3, con helper elevado)

| # | Ajuste | Mecanismo | Impacto | Riesgo | Reversión |
|---|---|---|---|---|---|
| 11 | HAGS (programación de GPU acelerada por hardware) | `HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers\HwSchMode` | variable (±), necesario para frame generation | medio | con reinicio |
| 12 | Plan "Máximo rendimiento" (Ultimate) | `powercfg -duplicatescheme` + borrar al revertir | B; en X3D/portátiles puede empeorar temperaturas | bajo | trivial |

### Descartadas (mitos o peligrosas) — no se implementan

| Idea | Motivo |
|---|---|
| Desactivar Defender, firewall, UAC, mitigaciones Spectre/Meltdown, VBS/HVCI | Baja la seguridad; prohibido por el proyecto |
| Desactivar servicios en masa (SysMain, Search, Update, etc.) | Riesgo de estabilidad; ganancia no demostrada |
| `bcdedit useplatformclock` / HPET / `disabledynamictick` | Resultados inconsistentes, puede empeorar frametimes |
| `TcpAckFrequency` / `TCPNoDelay` (Nagle) | Los juegos competitivos usan UDP: sin efecto |
| "Limpiadores de RAM" / vaciar standby list | Sin beneficio en la mayoría de los casos; puede causar stutter |
| `Win32PrioritySeparation`, `SystemResponsiveness`, `NetworkThrottlingIndex` | No hay mediciones serias; se reevaluarán solo si el benchmark lo demuestra |
| Desactivar archivo de paginación | Riesgo de crashes |
| Hacks de core parking / timer resolution global | Ya lo gestiona Windows; efecto no demostrado |
| Overclock / undervolt | Fuera de alcance |

## 3. Cambios de arquitectura necesarios

```
src/fps_booster/
  benchmark/            # NUEVO
    presentmon.py       # ejecuta PresentMon (CSV), parsea frames
    metrics.py          # avg, 1%/0.1% low, p99, stdev, cpu/gpu bound
    session.py          # protocolo A/B, warmup, repeticiones
    compare.py          # medianas + IC bootstrap → "medido" / "sin diferencia"
  games/                # NUEVO
    library.py          # detecta Steam (libraryfolders.vdf), Epic (manifests), Riot
    catalog/*.json      # perfil por juego: exe, config, presets por nivel de hardware
  optimizations/
    file_config.py      # NUEVO: FileConfigOptimization (backup del archivo entero)
    per_app.py          # NUEVO: valor de registro por exe (GPU preferida)
    display.py          # NUEVO: refresco del monitor
    session.py          # NUEVO: cambios que viven mientras corre el juego (prioridad)
  diagnostics/          # NUEVO: EXPO/XMP, canales de RAM, ReBAR, driver, Hz
  elevation/            # FASE 3: helper elevado con lista blanca de operaciones
```

Cambios en el motor:
- `Optimization.scope`: `persistent` (backup + revertir) o `session` (se revierte solo al
  cerrar el juego, también si la app crashea → guardia al arrancar).
- `requires_reboot`: el cambio queda "pendiente de verificar" hasta el próximo inicio.
- Recomendaciones con campo `measured_impact` que solo se llena con datos del benchmark local.
- Admin: la UI **nunca** corre elevada; un proceso helper aparte ejecuta solo operaciones de
  una lista blanca y crea un punto de restauración antes (Windows limita a 1 cada 24 h).

## 4. Orden de trabajo propuesto

1. **Benchmark** (`benchmark/`) + pantalla de resultados A/B. Sin esto no se mide nada.
2. **Diagnósticos** de nivel 1 (solo lectura, alto valor, cero riesgo).
3. GPU preferida por juego + flip model + refresco del monitor (nivel 2).
4. Detección de juegos + perfiles de Fortnite, CS2 y Valorant (nivel 3).
5. Prioridad de proceso en sesión.
6. Helper elevado + HAGS (nivel 4).

Cada punto: rama propia, pruebas con `SimulatedSystemBackend`, invariantes de `docs/AUDIT.md`
actualizados, y benchmark real antes de mostrar cualquier número de FPS.

## 5. Coordinación con Git

- `main` estable; una rama por punto (`feat/benchmark`, `feat/diagnostics`, …).
- Antes de tocar un módulo, el dueño de la rama lo anota en el PR; no se editan los mismos
  archivos en dos ramas a la vez.
- Revisión cruzada en cada PR: pruebas + `ruff` + invariantes de seguridad.
