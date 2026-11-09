# 07 — Metodología de benchmark reproducible

Base existente: `docs/BENCHMARK.md` en la rama `feat/benchmark` (PresentMon 2.6.0 verificado, A/B con
bootstrap). Este documento lo formaliza y lo amplía con las definiciones de la Parte 1 y el proceso de
la Parte 17. Donde ya existe una decisión en `feat/benchmark`, se respeta.

## 1. Definiciones (Parte 1.1)

Notación: `t_i` = frametime del frame *i* en ms (intervalo entre presentaciones de la aplicación; en
PresentMon 2.x, columna de tiempo entre frames de la app), `N` = cantidad de frames, `T = Σ t_i`.

| # | Métrica | Definición operativa | Notas |
|---|---|---|---|
| 1 | **FPS promedio** | `N / (T/1000)` | No es el promedio de FPS instantáneos (ese sesga hacia frames rápidos) |
| 2 | **FPS mínimos** | `1000 / max(t_i)` | Muy sensible a un único evento; se informa pero no se usa para decidir |
| 3 | **1 % low** | `1000 / P99(t)` | Existe otra definición ("promedio del 1 % más lento"); `feat/benchmark` calcula **ambas** y lo documenta |
| 4 | **0,1 % low** | `1000 / P99.9(t)` | Solo con ≥ 1000 frames |
| 5 | **Frametime promedio** | `T / N` | Equivale a 1000 / FPS promedio |
| 6 | **P95 de frametime** | percentil 95 de `t` | Interpolación lineal; documentar el método |
| 7 | **P99 de frametime** | percentil 99 de `t` | |
| 8 | **Variabilidad** | desvío estándar de `t`; además `P99 − P50` y MAD | El desvío solo no distingue ruido de tirones |
| 9 | **Microstuttering** | proporción de frames con `t_i > 2 × mediana local` (ventana de 30 frames) o alternancia rápida | `feat/benchmark` usa ">2× mediana global"; propuesta: agregar la versión local |
| 10 | **Stuttering prolongado** | rachas de frames con `t_i > 2 × mediana` que suman > 100 ms, o un frame > 250 ms | Captura tirones de carga/compilación |
| 11 | **Latencia de entrada** | Tiempo desde la entrada hasta el inicio del frame (I2FS de NVIDIA) | Solo **medida** con marcadores (Reflex/PCL Stats) |
| 12 | **Latencia de renderizado** | Inicio del frame → `Present()` | Marcadores o estimación con `CPUBusy`+`GPUBusy` |
| 13 | **Latencia de presentación** | `Present()` → mostrado | PresentMon |
| 14 | **Latencia de extremo a extremo** | Clic → fotón | Requiere hardware (LDAT o equivalente); si no, "no disponible" |
| 15 | **Utilización de GPU** | `GPUBusy / t_i` por frame (PresentMon 2.x) y uso de motor 3D | La utilización baja **no implica** que haga falta optimizar la GPU (Parte 1.2) |
| 16 | **Utilización de CPU (total y por núcleo)** | PDH por núcleo + `CPUBusy / t_i` | El total esconde el hilo principal saturado |
| 17 | **Frecuencia efectiva de CPU** | `% Processor Performance × frecuencia base` (por núcleo) | |
| 18 | **Frecuencia efectiva de GPU** | reloj SM promedio ponderado por tiempo (NVML) | |
| 19 | **Limitación térmica** | razones `*_thermal_slowdown` activas (NVIDIA); CPU: solo "posible" sin sensor | |
| 20 | **Limitación de potencia** | razones `sw_power_cap`/`hw_power_brake_slowdown` | |
| 21 | **Presión de memoria** | `% Committed Bytes In Use`, `Pages Input/sec` (hard faults), memoria comprimida | |
| 22 | **Saturación de VRAM** | `memory.used / memory.total` sostenido > 0,9 + stutters correlacionados | |
| 23 | **Rendimiento por vatio** | `FPS del motor / potencia GPU media` (solo GPU; sistema completo requiere medidor externo) | Etiquetar como "por vatio de GPU" |
| 24 | **Rendimiento sostenible** | métricas por ventanas de 5 min durante ≥ 30 min | Comparar primera vs última ventana |

### Por qué subir el FPS promedio no siempre mejora la experiencia

1. **Tirones:** 200 FPS de promedio con un 1 % low de 60 se sienten peor que 144 estables.
2. **Cuadros generados:** la generación de cuadros sube los FPS mostrados pero no los del motor; la
   respuesta a la entrada no mejora y la latencia puede subir.
3. **Por encima del refresco** sin VRR, más FPS solo reducen latencia y aumentan tearing y consumo.
4. **Calor:** un pico de 5 minutos que termina en throttling empeora la partida real.
5. **Latencia:** con la GPU al 100 % sin Reflex/límite, la cola de render agrega latencia aunque suban los FPS.

## 2. Criterio objetivo de "funciona" (nunca un solo indicador)

Una optimización se considera **mejora demostrada** solo si se cumplen **todas**:

1. **Validez:** ≥ 3 corridas por lado, ≥ 300 frames por corrida (ideal ≥ 60 s), misma huella de hardware,
   mismo driver, misma escena, sin topes de FPS detectados en ambos lados (o comparando estabilidad si
   hay tope).
2. **Métrica objetivo** del perfil (p. ej., 1 % low en Competitivo) mejora con el IC 95 % bootstrap de la
   diferencia de medianas entero por encima de 0 y mayor al 2 % (umbral de `feat/benchmark`).
3. **Métricas de guarda** (no deben empeorar más allá de su IC): FPS promedio, P99, latencia (si existe),
   temperatura media de GPU (+3 °C máx.), potencia (+10 % máx., salvo que el perfil lo permita), y
   **0 eventos nuevos** de inestabilidad (WHEA, 4101, cierres).
4. **Variabilidad entre corridas** (coeficiente de variación de la métrica objetivo) < 3 %; si no, el
   veredicto es "inconcluso" y se piden más corridas.

## 3. Clasificación de resultados (Parte 17)

| Resultado | Regla |
|---|---|
| **Mejora demostrada** | Se cumplen 1–4 de §2 |
| **Mejora probable** | Métrica objetivo mejora con IC que roza 0 o con 2 corridas; guardas OK |
| **Resultado inconcluso** | IC cruza 0 con dispersión alta, o datos insuficientes |
| **Sin mejora** | IC dentro de ±2 % con dispersión baja |
| **Regresión** | Métrica objetivo o una guarda empeora fuera del IC |
| **Incompatible** | El cambio no aplica o falla la verificación del motor |
| **Riesgo excesivo** | Inestabilidad, temperatura o seguridad fuera de límites, aunque suban los FPS |

## 4. Protocolo A/B (Parte 17, pasos 1–16)

1. **Hipótesis** y **mecanismo** escritos en la ficha del experimento (enlazados a la matriz 06).
2. **Compatibilidad** verificada por el motor (filtro de hardware/SO).
3. **Riesgo** evaluado y consentimiento del usuario.
4. **Escenario:** escena reproducible (benchmark integrado o recorrido fijo), resolución, calidad,
   límite de FPS y modo de ventana idénticos; cerrar aplicaciones no esenciales (la app lo sugiere, no las cierra).
5. **Calentamiento:** 5 s de retardo + primera corrida descartada si hubo compilación de shaders.
6. **Baseline:** 3 × 60 s (`--tag antes`).
7. **Un único cambio**, aplicado por el motor (backup + verificación).
8. Reinicio si el cambio lo requiere; luego **calentamiento** otra vez.
9. **Repetición:** 3 × 60 s (`--tag despues`).
10. **Comparación** de medianas + IC bootstrap (10 000 remuestreos sobre corridas; con 3 corridas el IC
    es amplio: se informa con honestidad).
11. **Variabilidad** entre corridas (§2.4).
12. **Temperaturas** y razones de límite durante ambas series.
13. **Estabilidad:** eventos WHEA/4101/41 entre series.
14. **Documentar** en el almacén de evidencia (JSON + CSV crudo).
15. **Revertir** automáticamente si el veredicto es regresión o riesgo excesivo (con confirmación).
16. **Conclusión** registrada en la KB local del equipo.

### Diseño contra sesgos

- **Orden:** alternar A-B-A-B cuando sea posible (en vez de AAA-BBB) para que el calor o procesos de fondo
  no favorezcan a un lado; si el cambio requiere reinicio, AAA-BBB con reinicio también en el lado A.
- **Temperatura inicial:** comenzar cada corrida con la GPU dentro de ±3 °C de la anterior.
- **Huella:** abortar si cambia el driver, la build de Windows o la resolución.
- **Ciego:** la app no muestra resultados parciales al usuario durante la serie (evita sesgo al repetir).

## 5. Prueba de rendimiento sostenido

30–60 min en la misma escena (o sesión real del usuario con detector de juegos), ventanas de 5 min:
FPS, 1 % low, frecuencia efectiva CPU/GPU, potencia y temperatura GPU, razones de límite. Comparar la
ventana 1 con la última: caída > 5 % con razón térmica activa → "limitación térmica sostenida".

## 6. Estabilidad tras cambios manuales (BIOS, RAM)

Checklist guiado (la app no ejecuta herramientas externas sin autorización): diagnóstico de memoria de
Windows, una prueba de estrés del fabricante o de código abierto elegida por el usuario, y 60 min de juego
con conteo de WHEA/4101/41 = 0.

## 7. Herramientas de medición válidas por categoría

| Categoría | Herramienta |
|---|---|
| Frames, FPS, frametime, CPU/GPU busy, tipo de frame | PresentMon 2.x (MIT) — también base de CapFrameX y FrameView |
| PC Latency | PresentMon/FrameView en juegos con Reflex/PCL Stats |
| Latencia de sistema | Hardware (LDAT u otro), importado como dato externo |
| GPU NVIDIA | NVML / `nvidia-smi` |
| CPU, memoria, disco | Contadores PDH de Windows |
| DPC/ISR | Windows Performance Recorder/Analyzer (ETW) en modo avanzado |
| Red | Pruebas propias de RTT/jitter/pérdida + guía de bufferbloat |
