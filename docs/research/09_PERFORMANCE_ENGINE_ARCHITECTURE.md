# 09 — Arquitectura del PERFORMANCE INTELLIGENCE ENGINE

## 1. Visión

El motor no "aplica tweaks": **formula hipótesis a partir de mediciones**, las filtra con una base de
conocimiento versionada, pide autorización, prueba **un cambio por vez** con benchmark A/B, conserva lo
que demuestra beneficio y revierte lo demás. Cada recomendación muestra su evidencia.

## 2. Flujo (Parte 12, pasos 1–20)

```
┌──────────── DETECCIÓN (solo lectura) ────────────┐
│ 1 hardware  2 SO/build  3 drivers  4 config actual│  → Huella del equipo
└──────────────────────────────────────────────────┘
                 │
┌──────────── MEDICIÓN ────────────┐
│ 5 benchmarks (PresentMon + PDH + NVML) │  → Corridas con métricas (07)
│ 6 cuellos de botella: síntoma → causa probable → causa confirmada
└──────────────────────────────────┘
                 │
┌──────────── RAZONAMIENTO ────────────┐
│ 7 KB versionada  8 filtro de compatibilidad
│ 9 riesgo  10 confianza  11 prioridad (utilidad esperada)
└──────────────────────────────────────┘
                 │
┌──────────── EXPERIMENTO ────────────┐
│ 12 autorización  13 backup  14 un cambio  15 medir
│ 16 comparar con baseline  17 regresiones  18 revertir si corresponde
└─────────────────────────────────────┘
                 │
┌──────────── APRENDIZAJE ────────────┐
│ 19 evidencia guardada  20 recomendaciones futuras (KB local del equipo)
└─────────────────────────────────────┘
```

## 3. Componentes

| Componente | Responsabilidad | Estado en `go-rewrite` | Estado en `feat/benchmark` |
|---|---|---|---|
| `detect` | Huella: CPU (topología, caché), GPU(s), RAM (módulos, canales, velocidades), monitor (Hz, VRR), discos, BIOS, build, drivers, VBS, plan de energía | Parcial (`sysinfo.go`, `telemetry.go`) | Parcial (`hardware/detector.py`, RAM por módulos) |
| `measure` | PresentMon (frames), PDH (CPU por núcleo, memoria, disco), NVML (GPU y razones de límite) | PDH + `nvidia-smi` (sin frames) | **PresentMon completo** |
| `diagnose` | Reglas síntoma → causa con niveles de certeza | ✗ | Parcial (CPU/GPU-bound, tope de FPS) |
| `kb` | Hechos y recomendaciones versionados (JSON + esquema), con fuentes y fecha de revisión | ✗ (catálogo fijo en código) | ✗ |
| `planner` | Filtra, puntúa y ordena candidatos | ✗ (perfiles = listas fijas) | ✗ |
| `engine` | Transacciones: backup → aplicar → verificar → rollback | ✅ (`engine.go`) | ✅ (Python) |
| `experiment` | Orquesta A/B (series, orden, reinicios, veredicto) | ✗ | Parcial (`compare.py`, CLI) |
| `evidence` | Almacén de corridas y veredictos | ◐ (sesiones de juego sin FPS) | ✅ (`store.py`) |
| `helper` | Proceso elevado con lista blanca | ✗ | ✗ |
| `ui` | Diagnóstico, experimentos, evidencia, perfiles por objetivo | ✅ (panel, páginas) | ◐ |

## 4. Del síntoma a la causa (ejemplos de reglas)

| Síntoma | Causas probables | Confirmación requerida |
|---|---|---|
| GPU al 60 % y FPS bajos | CPU-bound; tope de FPS; V-Sync; juego en iGPU | `CPUBusy ≈ frametime`; distribución plana en 60/144; adaptador del proceso |
| 1 % low malo, promedio bueno | Compilación de shaders; streaming (VRAM/disco); interrupciones; procesos de fondo | Stutters al entrar a zonas nuevas; VRAM > 90 %; correlación con picos de disco o de otros procesos |
| FPS caen tras 10–20 min | Térmico/potencia | Razones de límite activas + temperatura en ascenso |
| "Tirones" con frametime estable | Red | Jitter/pérdida en NETWORK DIAGNOSTICS |
| FPS iguales en todas las calidades | CPU-bound o tope | Ídem fila 1 |

Un caso se marca **confirmado** solo cuando la evidencia medida lo respalda; de lo contrario queda como **probable**.

## 5. Puntuación de utilidad (pesos documentados)

### 5.1 Compuertas (no son pesos: si fallan, el candidato se descarta)

1. **Seguridad:** nivel N5 o reglas no negociables (08) → nunca se propone.
2. **Compatibilidad:** la KB y el filtro de hardware/SO deben aprobarlo.
3. **Reversibilidad:** sin reversión automática → solo como guía manual (N4).
4. **Estabilidad:** cualquier evento nuevo de WHEA/4101/41 durante el experimento → regresión.

Separar compuertas de pesos evita que "muchos FPS" compensen un riesgo de seguridad o de estabilidad.

### 5.2 Utilidad medida (después del experimento)

```
U = Σ_k  w_k(perfil) · clamp( Δ_k / s_k , −1 , +1 )
```

- `Δ_k`: cambio de la métrica *k* (signo positivo = mejor; para frametime/latencia/temperatura se invierte).
- Solo entra `Δ_k` si su IC 95 % no cruza 0; si lo cruza, `Δ_k = 0` (sin evidencia, no suma ni resta).
- `s_k`: **escala de diferencia perceptible** (normaliza unidades). Valores iniciales, documentados como
  supuestos a revisar con datos del proyecto:

| Métrica *k* | `s_k` (supuesto inicial) | Motivo del supuesto |
|---|---|---|
| FPS promedio | 5 % | Por debajo de ~5 % las diferencias suelen quedar dentro del ruido entre corridas [NV] |
| 1 % low | 5 % | Ídem |
| P99 de frametime | 5 % | Ídem |
| Desvío de frametime | 10 % | Métrica más ruidosa |
| Latencia (medida) | 10 % o 2 ms (lo mayor) | Diferencias de pocos ms son difíciles de percibir [NV] |
| Temperatura GPU | 3 °C | Umbral de guarda de 07 |
| Potencia GPU | 10 % | Umbral de guarda de 07 |

### 5.3 Pesos por perfil (Parte 13)

| Perfil | FPS prom. | 1 % low | P99 | Desvío | Latencia | Temp. | Potencia | Justificación |
|---|---|---|---|---|---|---|---|---|
| MAXIMUM FPS | 0,45 | 0,25 | 0,10 | 0,05 | 0,05 | 0,05 | 0,05 | Prioriza FPS nativos sostenidos; temperatura evita picos no sostenibles |
| COMPETITIVE MODE | 0,20 | 0,30 | 0,10 | 0,05 | 0,30 | 0,05 | 0,00 | FPS, 1 % low y latencia por igual relevancia |
| ULTRA LOW LATENCY | 0,05 | 0,15 | 0,10 | 0,05 | 0,60 | 0,05 | 0,00 | La respuesta domina |
| SMOOTH GAMING | 0,10 | 0,30 | 0,25 | 0,25 | 0,05 | 0,05 | 0,00 | Estabilidad de frametime |
| AAA PERFORMANCE | 0,30 | 0,25 | 0,15 | 0,10 | 0,05 | 0,10 | 0,05 | FPS con calidad; la calidad visual es una restricción, no un peso |
| LOW-END RESCUE | 0,35 | 0,30 | 0,10 | 0,05 | 0,05 | 0,10 | 0,05 | Hacer jugable: FPS y mínimos; temperatura por equipos con refrigeración pobre |
| CUSTOM | definidos por el usuario (suman 1) | | | | | | | Objetivos propios |

Estos pesos **no son verdades empíricas**: son una política explícita, versionada y modificable por el
usuario. Se revisarán con los resultados de los experimentos.

### 5.4 Utilidad esperada (antes de probar) y prioridad

```
prioridad = P(beneficio | KB, diagnóstico) × U_esperada × confianza − costo(riesgo, reinicio, tiempo de prueba)
```

- `P(beneficio)` sale de la KB: N1 ≈ alta, N2 depende de que el diagnóstico confirme la condición, N3 baja.
- `confianza` según nivel de evidencia: OF > SEC > NV (NV nunca se propone automáticamente).
- `costo`: reinicio requerido, permisos de administrador, duración del experimento.

### 5.5 Perfiles = objetivos, no listas

Un perfil define pesos (5.3) + restricciones (p. ej., "sin generación de cuadros", "calidad ≥ Alta").
El `planner` arma la lista de candidatos **para este equipo y este juego**, los ordena por prioridad y
propone experimentos de a uno. Resultado: dos PCs con el mismo perfil reciben recomendaciones distintas.

## 6. Base de conocimiento

- Formato: JSON con esquema (JSON Schema) y pruebas que validan cada archivo.
- Versionado semántico: `kb_version`; cada hecho tiene `reviewed` y `sources` (IDs a la tabla de fuentes).
- Caducidad: hechos atados a versiones de driver/Windows se marcan "a revisar" cuando cambian.
- Evidencia local: resultados A/B del propio equipo pesan más que la KB general para ese equipo.

## 7. Integraciones investigadas (Parte 14)

| Proyecto | Funcionalidad | Licencia | Mantenimiento | Integración posible | Riesgos / límites | Evidencia |
|---|---|---|---|---|---|---|
| **PresentMon** (Intel GameTechDev) | Captura de frames vía ETW; CLI headless (2.4+); FrameType (2.3+); 2.6.0 con `dump-caps` | MIT | Activo (2.6.0) | Proceso externo verificado por hash + firma (ya hecho en `feat/benchmark`) | ETW requiere admin o *Performance Log Users*; columnas cambian entre versiones | [SEC] newreleases (notas de versión); `feat/benchmark` |
| **CapFrameX** | Captura/análisis basado en PresentMon | MIT según mirror de SourceForge (verificar en GitHub) | Activo (releases 1.9.x) | Importar CSV (ya soportado por `feat/benchmark` `import`) | No embeber código sin revisar licencia | [SEC] |
| **OCAT** (AMD GPUOpen) | Captura y overlay | MIT según agregador; mirror sin licencia declarada | Última versión conocida 1.6.1 (2020); probablemente inactivo | Solo importar CSV | Overlay = inyección (riesgo anticheat) | [SEC] |
| **LibreHardwareMonitor** | Sensores (incluida temperatura de CPU) | MPL-2.0 (mirror) | Activo; migró de WinRing0 a **PawnIO** (instalación aparte) | Opcional: proceso separado + driver PawnIO con consentimiento | Driver de kernel; antecedente WinRing0 marcado por Defender | [OF] Microsoft (WinRing0); [SEC] |
| **HWiNFO** | Sensores muy completos; memoria compartida | Propietaria (uso de su interfaz sujeto a sus términos) [NV] | Activo [NV] | Leer su memoria compartida **solo si el usuario la habilita** | Licencia y límites de tiempo de la interfaz [NV] | [NV] |
| **GPU-Z** | Información de GPU | Propietaria [NV] | Activo [NV] | No integrar; solo referencia | Licencia | [NV] |
| **MSI Afterburner / RTSS** | OC de GPU, overlay | Propietarias [NV] | Activo [NV] | No integrar (OC y overlay por inyección) | Anticheat, OC | [NV] |
| **NVIDIA** NVML / `nvidia-smi`, FrameView, Reflex SDK | Telemetría GPU, PC Latency | NVML con el driver; FrameView gratuito | Activo | `nvidia-smi` ya integrado; NVML directo (DLL del driver) a futuro | — | [OF] |
| **AMD** ADLX, GPUOpen | Telemetría y control Radeon | SDK de AMD (licencia propia) [NV] | Activo [NV] | Telemetría Radeon (solo lectura) | — | [NV] |
| **Intel** PresentMon, IGCL, APO | Frames, control/telemetría Arc, scheduling | MIT (PresentMon); otras [NV] | Activo | Telemetría Arc (solo lectura) | — | [SEC]/[NV] |
| **Windows Performance Toolkit** (WPR/WPA, ETW) | Trazas de CPU, DPC/ISR, disco | Herramienta gratuita de Microsoft [NV] | Activo | Modo avanzado: captura guiada de WPR para DPC/ISR | Trazas grandes; requiere admin | [NV] |
| **WMI / PowerShell** | Inventario y configuración | Windows | — | Ya usados (solo lectura salvo motor) | Costo de arranque de PowerShell (~1–2 s) | Medido en el proyecto |
| **psutil, pywin32, PySide6, pytest, Ruff** | Stack de la rama Python (`main`/`feat/benchmark`) | BSD-3 / PSF / LGPL-3 / MIT / MIT [NV] | Activos [NV] | Relevantes **solo si se elige la base Python** (decisión D0) | LGPL de PySide6 requiere enlazado dinámico [NV] | [NV] |
| **Go + Wails** (base actual de `go-rewrite`) | App nativa de un solo exe | Go BSD-3; Wails MIT [NV] | Activos | Base actual | WebView2 requerido | [NV] |
