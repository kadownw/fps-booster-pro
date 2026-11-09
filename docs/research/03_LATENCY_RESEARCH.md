# 03 — Investigación de latencia: entrada, render, presentación y red

Convenciones: [OF] oficial consultado · [SEC] secundario consultado · [NV] no verificado.

## 1. Modelo de la cadena de latencia

```
clic/movimiento ─► periférico (polling USB) ─► SO (ISR/DPC, raw input) ─► juego: muestreo de entrada
  ─► simulación (CPU) ─► envío de render (CPU) ─► cola de render ─► GPU ─► Present()
  ─► composición (DWM) / flip independiente ─► escaneo del monitor ─► respuesta de píxel (fotón)
```

| Tramo | Nombre en NVIDIA PC Latency | Medible por software | Herramienta válida |
|---|---|---|---|
| Periférico (polling, debounce) | — (fuera de "PC Latency") | No | Hardware (LDAT, cámara de alta velocidad, analizadores USB) |
| Entrada → inicio de frame | I2FS | Solo en juegos con marcadores (Reflex/PCL Stats) | PresentMon 2.x / FrameView |
| Inicio de frame → Present | FS2P | Ídem | Ídem |
| Present → mostrado | P2D | Sí (PresentMon: tiempo hasta mostrarse) | PresentMon |
| Monitor (procesamiento + respuesta) | — | No | Hardware / reseñas del monitor |
| Red (RTT, jitter, pérdida) | — | Sí (ping/UDP propio, contadores del juego) | NETWORK DIAGNOSTICS (ver 04) |

NVIDIA define **PC Latency** como la suma de los promedios de I2FS, FS2P y P2D, ignorando frames
descartados; aclara que la medición de I2FS no es correcta por debajo de ~10 FPS y que, para la
latencia completa de extremo a extremo, se necesita hardware (LDAT) [OF].

**Regla del proyecto:** nunca mostrar frametime como "latencia". Toda cifra de latencia lleva una etiqueta:

| Etiqueta | Significado |
|---|---|
| **Medida (PC)** | PC Latency reportada por el juego (Reflex/PCL Stats) vía PresentMon |
| **Estimada (render)** | Derivada de tiempos de PresentMon sin marcadores de entrada (p. ej., Present→mostrado + tiempo de CPU/GPU) |
| **Medida (sistema)** | Medición de hardware importada (LDAT u otra) |
| **No disponible** | Cuando no hay datos: no se inventa un número |

## 2. Factores investigados

| # | Factor | Efecto | Qué puede hacer la app | Nivel | Evidencia |
|---|---|---|---|---|---|
| 1 | Input lag total | Suma de todo lo anterior | Mostrar desglose disponible | — | [OF] NVIDIA |
| 2–4 | Polling de mouse/teclado/USB | 1000 Hz ≈ 1 ms de intervalo vs 125 Hz ≈ 8 ms [NV]; frecuencias más altas cuestan CPU | Solo informar (se configura en el software del periférico) | N2 | [NV] |
| 5 | Suspensión selectiva de USB | Puede dormir dispositivos; relevante si hay cortes | Diagnóstico; el catálogo actual tiene `usb_recovery` (otra cosa) | N3 | [NV] |
| 6–7 | DPC/ISR | Drivers con DPC largos causan tirones y audio cortado | Diagnóstico con ETW (Windows Performance Toolkit/xperf) en modo avanzado; no "arreglar" por registro | N3 | [NV] |
| 8 | Latencia de render | Tiempo de CPU+GPU del frame | PresentMon (`CPUBusy`, `GPUBusy`) | — | [SEC] PresentMon 2.x |
| 9–10 | Cola de render (CPU/GPU) | Cuando la GPU es el cuello, la CPU adelanta frames y crece la latencia | Recomendar Reflex/Anti-Lag 2 integrados o un límite de FPS | N1 | [OF] NVIDIA |
| 11 | Frame pacing | Variabilidad de frametime percibida como tirones | Medir desvío, p99, stutters | — | `BENCHMARK.md` |
| 12–13 | Respuesta y refresco del monitor | Más Hz = menos tiempo de escaneo y más muestras | Detectar Hz actual vs máximo (PC de desarrollo: 60/75) | **N1** | Diagnóstico local |
| 14 | VRR | Evita tearing sin la latencia de V-Sync dentro del rango | Detectar soporte; recomendar límite bajo el máximo | N2 | [NV] |
| 15 | V-Sync | Agrega latencia si los FPS superan el refresco | Recomendar según objetivo (competitivo vs calidad) | N2 | [NV] |
| 16 | Limitadores de FPS | Un límite estable reduce la cola y la variabilidad | Preferir limitador del juego o Reflex | N1/N2 | [NV] |
| 17 | NVIDIA Reflex | Elimina la cola de render | Recomendar activarlo en el juego | N1 (si existe) | [OF] |
| 18 | AMD Anti-Lag 2 | Equivalente integrado por juego (RX 5000+). **Anti-Lag+** por driver causó baneos de VAC en CS2 | Recomendar solo la versión integrada en el juego | N1 (si existe) | [SEC] |
| 19 | Raw input | Evita aceleración y suavizado de Windows | Recomendar opción del juego; `mouse_accel` del catálogo solo afecta al escritorio | N1 (preferencia) | [NV] |
| 20 | Pantalla completa vs sin bordes | Con flip model moderno la diferencia se reduce; con *blt* (DX10/11 en ventana) hay más latencia | "Optimizaciones para juegos en ventana" (Win11) | N1/N2 | [OF] DirectX Blog |
| 21 | Compositor (DWM) | En modo composición agrega un ciclo de refresco | PresentMon reporta el modo de presentación | Diagnóstico | [NV] |
| 22 | Latencia de pantalla | Procesamiento del monitor (modo juego del monitor) | Informar | Manual | [NV] |
| 23–25 | Latencia, jitter y pérdida de red | Ver `04_NETWORK_RESEARCH.md` | — | — | — |

## 3. Módulo LATENCY LAB (diseño)

Objetivo: encontrar la configuración de **menor latencia estable** por combinación juego × GPU × monitor × periféricos.

**Entradas:** perfil del juego (soporta Reflex/Anti-Lag 2, limitador propio, modos de ventana), monitor
(Hz, VRR), GPU (fabricante), datos de PresentMon por corrida.

**Variables de prueba (de a una por corrida, Parte 17):**
1. Reflex/Anti-Lag 2: apagado → activado → activado+boost.
2. Límite de FPS: ninguno → refresco−3 → valor sostenible (p. ej., p5 de FPS).
3. Modo de ventana: exclusiva → sin bordes (con flip model).
4. V-Sync: apagado → activado (solo con VRR para objetivos de suavidad).

**Métricas por corrida:** PC Latency (si existe) mediana y p95; Present→mostrado; FPS mostrados y del
motor; 1 % low; desvío de frametime; % de frames con tearing posible (si PresentMon lo indica) [NV].

**Criterio de "mejor":** mínima mediana de latencia **con** p95 dentro de +10 % de la mediana (estabilidad)
y 1 % low no peor que el baseline fuera del intervalo de confianza. Los pesos se documentan en `09`.

**Límites honestos:**
- Sin marcadores Reflex/PCL, la latencia de entrada **no se mide**; solo se estima el tramo de render/presentación.
- La latencia del periférico y del monitor solo se conoce con hardware o datos de reseñas.
- Las opciones del juego las cambia el usuario (o la app vía archivo de configuración con el juego cerrado y backup, solo en juegos sin restricciones de anticheat sobre esos archivos).

## Fuentes consultadas en esta sesión

- NVIDIA — Understanding and Measuring PC Latency: https://developer.nvidia.com/blog/understanding-and-measuring-pc-latency
- NVIDIA — FrameView user guide: https://images.nvidia.com/content/geforce/technologies/frameview/frameview-1-7-user-guide-web-version.pdf
- NVIDIA — Reviewer Toolkit: https://www.nvidia.com/en-us/geforce/news/nvidia-reviewer-toolkit/
- Microsoft — Optimizations for windowed games: https://devblogs.microsoft.com/directx/updates-in-graphics-and-gaming/
- Anti-Lag 2 / Anti-Lag+: https://www.tomshardware.com/pc-components/gpu-drivers/amd-introduces-radeon-anti-lag-2-with-full-integration-for-counter-strike-2-the-nvidia-reflex-alternative-shouldnt-trigger-cheating-bans-this-time · https://windowscentral.com/gaming/pc-gaming/amds-anti-lag-feature-is-wreaking-havoc-for-counter-strike-2-apex-legends-and-call-of-duty-players
- Comparativa independiente Reflex vs Anti-Lag: https://www.igorslab.de/en/radeon-anti-lag-vs-nvidia-reflex-im-test-latenzvergleich/
