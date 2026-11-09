# 01 — Investigación de hardware: CPU, GPU, RAM, almacenamiento, firmware y refrigeración

Convenciones de evidencia usadas en todos los documentos:

- **[OF]** documentación oficial (Microsoft, NVIDIA, AMD, Intel) consultada en esta sesión (en general, extractos).
- **[SEC]** fuente secundaria (prensa técnica, laboratorios independientes, foros técnicos) consultada en esta sesión.
- **[NV]** conocimiento técnico previo **no verificado en esta sesión**. No implementar sin verificar.

Regla transversal: **el hardware se diagnostica, no se "optimiza" a ciegas.** La app detecta, explica y mide;
los cambios de firmware, voltaje o frecuencia son siempre manuales (Nivel 4).

---

## 1. CPU

### 1.1 Qué determina el rendimiento de CPU en juegos

En la mayoría de los motores, un hilo principal (lógica del juego + envío de comandos al driver)
fija el techo de FPS cuando la GPU tiene margen. Ese techo depende de:

| Factor | Por qué importa | Cómo se detecta | Evidencia |
|---|---|---|---|
| IPC y frecuencia efectiva del núcleo que corre el hilo principal | Tiempo de CPU por frame | PDH `\Processor Information(*)\% Processor Performance` × frecuencia base; por núcleo | [OF] contadores de Windows; [NV] metodología |
| Caché L3 (y latencia de memoria) | Los juegos tienen conjuntos de trabajo grandes y accesos poco predecibles; más caché = menos esperas a RAM. Es la razón del rendimiento de los Ryzen X3D | `GetLogicalProcessorInformationEx` (ya implementado: L2/L3 por paquete) | [NV] |
| Número de núcleos | Importa hasta que el motor deja de paralelizar; muchos juegos escalan poco más allá de 6–8 núcleos | Detección + `CPUBusy` de PresentMon | [NV] |
| SMT / Hyper-Threading | Ayuda en cargas paralelas; en algunos juegos puede costar algo por competencia de recursos, pero **desactivarlo no es una recomendación general** | Solo A/B medido | Regla del encargo + [NV] |
| Topología (CCD en Ryzen, P/E cores en Intel) | Si un hilo crítico cae en un CCD sin V-Cache o en un E-core, sube el frametime | Topología + afinidad observada | [SEC] (ver 1.3) |

**Saturación de un hilo principal:** el uso total de CPU puede marcar 35 % mientras un núcleo está al
100 %. Por eso **el uso total no sirve para decidir**: se necesita uso por núcleo + `CPUBusy` del frame
(PresentMon 2.x). Esta es la causa más común de "mi GPU está al 60 % y no sube".

### 1.2 Arquitecturas

| Familia | Rasgos relevantes para juegos | Riesgos de "tweaks" | Evidencia |
|---|---|---|---|
| AMD Zen 2 (Ryzen 3000) | Dos CCX de 4 núcleos por CCD, L3 partida por CCX | Afinidades manuales pueden empeorar | [NV] |
| AMD Zen 3 (Ryzen 5000) | CCX unificado de 8 núcleos, gran salto de rendimiento en juegos | — | [NV] |
| AMD Zen 4 / Zen 5 (Ryzen 7000/9000) | DDR5 obligatoria, EXPO; Zen 5 se benefició de un cambio del predictor de saltos en Windows 11 (KB5041587 / 24H2): reportes de hasta ~11 % en 1080p (Hardware Unboxed vía prensa) | Sin el parche, se compara mal | [SEC] guru3d, Tom's Hardware, TechRadar |
| Ryzen X3D (5800X3D, 7800X3D, 9800X3D, 7900X3D/7950X3D, 9900X3D/9950X3D) | 3D V-Cache: L3 mucho mayor → grandes ganancias en juegos CPU-bound. En modelos de **dos CCD**, solo uno tiene V-Cache: el driver "3D V-Cache Performance Optimizer" + archivos de provisión de energía + Game Bar estacionan el CCD sin caché durante juegos | Planes de energía de terceros o "desactivar core parking" pueden **romper** el estacionamiento de CCD | [SEC] foros AMD/EA, Tom's Hardware; el requisito de Game Bar es reporte de foros (sin confirmación oficial en esta sesión) |
| Intel Core 12.ª–14.ª gen (híbridos) | P-cores + E-cores; **Thread Director** informa al planificador de Windows 11. Intel APO ajusta el scheduling en una lista de juegos y CPUs (requiere Windows 11 y DTT) | Desactivar E-cores globalmente no es recomendable | [SEC] TechSpot, Neowin, Corsair (lista de juegos: verificar en Intel) |
| Intel Core Ultra (Meteor/Arrow/Lunar Lake) | Híbridos, sin HT en Arrow Lake [NV]; sensibles a versión de Windows y BIOS | Ídem | [NV] |
| Intel 9.ª gen (caso de la PC de desarrollo: i3-9100F) | 4C/4T, sin SMT, DDR4 hasta 2400 en i3 [NV — verificar ARK] | Muy CPU-bound con GPUs medias | Diagnóstico local |

### 1.3 Gestión de energía y planificador

| Mecanismo | Qué hace | Postura del proyecto | Evidencia |
|---|---|---|---|
| Plan **Equilibrado** | Escala frecuencia y estaciona núcleos según carga | Base recomendada en Ryzen (especialmente X3D) y portátiles; medir antes de cambiar | [NV] + [SEC] X3D |
| Plan **Alto rendimiento** | Mínimo de estado de CPU alto, menos estacionamiento | Puede ayudar en CPUs viejas o desktops; sube consumo y temperatura | [NV] |
| Plan **Máximo rendimiento** (Ultimate) | Basado en Alto rendimiento; quita ahorro y polling para reducir micro-latencias; pensado para estaciones de trabajo, oculto en Home/Pro, no se habilita en batería | Experimental en juegos; no aplicar en portátiles | [SEC] (no se encontró la página oficial actual) |
| Core parking (`CPMinCores`, etc.) | Microsoft documenta que con `CPMinCores=100 %` el algoritmo de estacionamiento se apaga | **No se desactiva indiscriminadamente**: rompe el estacionamiento de CCD en X3D de dos CCD y sube consumo | [OF] learn.microsoft.com (opciones de core parking) |
| CPPC / preferred cores | El firmware informa núcleos más rápidos; Windows los prefiere | Solo diagnóstico (BIOS: CPPC habilitado) | [NV] |
| EcoQoS / Power Throttling | Windows reduce prioridad energética de procesos en segundo plano | Desactivarlo afecta sobre todo a procesos de fondo, no al juego en primer plano → valor bajo | [NV] |

**Hallazgo en la PC de desarrollo:** el plan activo es "Paragon Core Power Plan", instalado por otra
herramienta. La app debe **detectar planes de terceros** y ofrecer compararlos con benchmark, no
sobreescribirlos a ciegas.

### 1.4 PBO, Curve Optimizer, undervolting (solo Nivel 4, manual)

- **PBO** (Precision Boost Overdrive) amplía límites de potencia/corriente; **Curve Optimizer** ajusta la
  curva voltaje-frecuencia por núcleo. Puede mejorar rendimiento sostenido y temperaturas, pero
  la estabilidad varía por chip y puede afectar garantía según el fabricante [NV].
- Undervolt en Intel puede estar bloqueado por mitigaciones (p. ej., "Plundervolt") según BIOS [NV].
- **Postura:** la app nunca los aplica. Puede detectar si están activos solo cuando exista una
  interfaz confiable; si no, únicamente explica y enlaza documentación del fabricante, y propone
  **pruebas de estabilidad** (ver 07, §6) después de cualquier cambio manual.

### 1.5 Algoritmo "CPU inteligente" (diseño)

```
entrada: topología (núcleos, hilos, CCD/P-E, L2/L3), frecuencia base/boost esperada (KB),
         muestras por núcleo (uso, % Processor Performance), PresentMon (CPUBusy, GPUBusy, frametime)
1. Normalidad de frecuencia:
   efectiva_p95 durante carga < 0,85 × boost_esperado  → sospecha de límite (térmico/potencia/plan)
2. Throttling: correlacionar caída de frecuencia con temperatura (si hay sensor confiable) o con
   cambio de plan; sin sensor → "posible", nunca "confirmado".
3. Saturación de hilo principal: algún núcleo > 90 % de ocupación sostenida mientras el uso total < 60 %.
4. Limitado por CPU (confirmado): mediana(CPUBusy) ≥ 0,85 × mediana(frametime) y GPUBusy claramente menor.
5. Recomendaciones candidatas (consultando la KB por arquitectura):
   - si limitado por CPU y FPS > refresco → límite de FPS / Reflex (latencia, consumo)
   - si limitado por CPU y FPS < refresco → bajar opciones que cargan CPU (distancia de visión, multitudes,
     física) según perfil del juego; revisar RAM (canales/velocidad); revisar plan de energía (A/B)
   - si X3D dos CCD → verificar driver de chipset y estado de estacionamiento (diagnóstico)
6. Beneficio esperado: rango de la KB con su nivel de evidencia; si no hay datos, "desconocido".
7. Evidencia: cada recomendación guarda IDs de fuente de la matriz y, tras aplicar, el resultado A/B.
```

---

## 2. GPU

### 2.1 Compatibilidad y detección

| Caso | Cómo detectar | Notas |
|---|---|---|
| NVIDIA | `nvidia-smi` / NVML (ya implementado parcialmente) | Temperatura, uso, relojes, potencia, ventilador, **razones de limitación de reloj** (`clocks_event_reasons.*`; antes `clocks_throttle_reasons.*`) [OF] NVML / [SEC] |
| AMD Radeon | ADLX (SDK oficial de AMD) [NV] | Sin integrar. Hasta entonces: "datos no disponibles" |
| Intel Arc / iGPU | IGCL (Intel Graphics Control Library) [NV] o contadores de Windows `GPU Engine` | Sin integrar |
| Cualquier GPU (uso) | PDH `\GPU Engine(*engtype_3D)\Utilization Percentage` [NV] | Independiente del fabricante; sin temperatura |
| Híbridos (iGPU + dGPU) | Enumerar adaptadores; preferencia por app en `HKCU\Software\Microsoft\DirectX\UserGpuPreferences` (formato `GpuPreference=2;`) | Formato **no documentado oficialmente** [SEC]; usar la API/Configuración cuando sea posible |
| Multi-GPU | Varias claves en la clase Display | Ya contemplado en el catálogo por fabricante |

### 2.2 Tecnologías de rendimiento y latencia

| Tecnología | Qué hace | Implicancia para la app | Evidencia |
|---|---|---|---|
| **NVIDIA Reflex** | Elimina la cola de render (CPU listo "justo a tiempo"); reporta PC Latency vía marcadores | Recomendar activarla **dentro del juego** cuando exista. Medir con PresentMon/FrameView (PC Latency = I2FS + FS2P + P2D promediados) | [OF] blog de desarrolladores de NVIDIA, guía de FrameView |
| **DLSS Super Resolution** | Renderiza a menor resolución y reconstruye | Sube FPS **nativos de motor** si el juego está limitado por GPU; sin efecto si está limitado por CPU | [NV] |
| **DLSS Frame Generation** (RTX 40+) | Interpola cuadros; sube FPS mostrados, no los del motor; agrega latencia (mitigada con Reflex). Requiere HAGS activado según soporte de Nixxes | Nunca contar cuadros generados como FPS nativos (ver 07) | [SEC] Nixxes; NVIDIA |
| **DLSS 4 Multi Frame Generation** (solo RTX 50) | Hasta 3 cuadros generados por cada renderizado; depende de *flip metering* por hardware (Blackwell); DLSS 4.5 agrega modos 5×/6× y MFG dinámico | Ídem; excluyente por arquitectura | [OF] NVIDIA GeForce News; [SEC] TechPowerUp, Tom's Hardware |
| **AMD FSR** (upscaling) / **FSR Frame Generation** | Equivalentes multiplataforma | Ídem DLSS | [NV] |
| **AMD Fluid Motion Frames (AFMF)** | Generación de cuadros a nivel de driver | PresentMon 2.3+ reporta tipo de frame incluyendo AFMF | [SEC] KitGuru (PresentMon 2.3) |
| **AMD Anti-Lag 2** | Reducción de latencia integrada por juego (SDK), RX 5000+; debutó en CS2 | **Anti-Lag+** (por driver) provocó **baneos de VAC** y problemas con anticheats en CS2, CoD y Apex → la app nunca debe forzar inyecciones de driver; solo recomendar opciones integradas en el juego | [SEC] Tom's Hardware, Windows Central, KitGuru |
| **Intel XeSS / XeFG / XeLL** | Upscaling, generación de cuadros y baja latencia de Intel | PresentMon 2.3 agregó soporte de XeFG/XeLL | [SEC] KitGuru |
| **HAGS** | El scheduler de la GPU asume tareas que hacía un hilo de CPU (WDDM 2.7+, desde Windows 10 2004); Microsoft lo describe como un cambio grande para los drivers | Impacto variable; **requisito** para Frame Generation | [OF] DirectX Developer Blog (30/06/2020); [SEC] TechPowerUp, PCGH |
| **Resizable BAR / SAM** | La CPU accede a toda la VRAM | NVIDIA usa lista blanca por juego; TechSpot midió **regresiones** en algunos títulos (p. ej., −4 % repetible en Far Cry New Dawn al forzarlo); TechPowerUp promedió +2–4 % con soporte | [SEC] TechSpot, TechPowerUp. **No forzarlo por juego** |
| PCIe link | Verificar generación/ancho efectivo vs máximo (`nvidia-smi pcie.link.*`) | Diagnóstico: x8 o Gen1 por error de slot/BIOS | Diagnóstico local (OK en la PC de desarrollo: Gen3 x16) |
| V-Sync / VRR (G-SYNC, FreeSync) / limitadores | Con VRR, limitar unos FPS por debajo del refresco suele mejorar frametimes y latencia | Recomendación de configuración, medible | [NV] + `PERFORMANCE_PLAN.md` |
| Caché de shaders | Evita recompilar; borrarla obliga a reconstruir y puede **causar tirones** | **No borrar de forma rutinaria** (regla del encargo). Solo ante corrupción o tras actualización de driver con síntomas | [SEC] (guías y foro de desarrolladores de NVIDIA) |
| Power limits / curvas | Cambian sostenido y temperatura | Nivel 4 (manual) | — |

### 2.3 Diferenciar FPS reales de generados

PresentMon 2.3 incorporó la métrica **FrameType** (tipo de cada frame) y extendió "Displayed FPS" a
cuadros generados [SEC]. La app debe reportar por separado:

| Métrica | Fuente | Uso |
|---|---|---|
| FPS del motor (renderizados por la app) | frames con FrameType "Application" | Rendimiento real de CPU/GPU |
| FPS presentados | todos los `Present()` | Pipeline |
| FPS mostrados | frames con tiempo en pantalla > 0 | Lo que ve el usuario |
| FPS generados | FrameType distinto de "Application" (DLSS FG, FSR FG, AFMF, XeFG) | Suavidad visual, no respuesta |
| Latencia | PC Latency / Click-to-Photon si se reporta | Respuesta |

Regla de UI: **"X FPS (Y nativos)"**, nunca solo X cuando hay generación de cuadros.

### 2.4 Perfiles por GPU (conceptuales)

Los perfiles no aplican listas fijas: cada uno es un **objetivo** + **restricciones** que el motor
resuelve con la KB y el benchmark (ver 09). Ejemplos de lógica:

| Perfil | Objetivo | Preferencias típicas (si el juego las ofrece) |
|---|---|---|
| Competitive FPS | 1 % low y latencia | Reflex/Anti-Lag 2 ON; sin frame generation; límite de FPS estable; calidad baja en efectos que cargan CPU |
| Maximum Native FPS | FPS del motor | Upscaling de calidad si GPU-bound; sin FG |
| Balanced Gaming | FPS ≥ refresco con buena calidad | Upscaling Calidad; límite = refresco − 3 con VRR |
| High Quality | Calidad visual con FPS mínimos aceptables | Ray tracing según margen; FG permitido si FPS base ≥ ~60 [NV] |
| Low Latency | Latencia mínima estable | Reflex + límite de FPS; fullscreen o flip model |
| Stable Frametime | Desvío y p99 mínimos | Límite de FPS bajo el promedio sostenido |
| GPU Limited | Bajar costo de GPU | Upscaling, sombras, RT |
| VRAM Limited | Evitar desbordes de VRAM | Texturas un nivel abajo; detectar uso de VRAM > ~90 % sostenido |

**Nunca** se aplica una opción de NVIDIA a AMD ni viceversa: el filtro de compatibilidad es la primera
etapa del motor.

---

## 3. Memoria RAM

| Tema | Hecho técnico | Qué hace la app | Evidencia |
|---|---|---|---|
| MT/s vs MHz | DDR transfiere dos veces por ciclo: "3200" son MT/s (1600 MHz de reloj) | Mostrar MT/s y aclarar | [NV] |
| Frecuencia configurada vs nominal | `Win32_PhysicalMemory.Speed` (nominal) vs `ConfiguredClockSpeed` (actual) | Detectar XMP/EXPO apagado o RAM limitada por plataforma | [OF] WMI (dato observable) |
| Dual vs single channel | Un solo módulo o módulos en el mismo canal reducen el ancho de banda a la mitad | Detectar por `DeviceLocator`/`BankLabel` y capacidades por canal | Diagnóstico local (PC de desarrollo: asimétrica 4 / 4+8) |
| Módulos mixtos | Distintas marcas/timings → el sistema usa el más lento; riesgo de inestabilidad con XMP | Avisar; recomendar pares iguales | [NV] |
| XMP / EXPO | Perfiles del fabricante del módulo; es un overclock de memoria certificado por el módulo, no por el CPU | Recomendación **manual** (BIOS) + prueba de estabilidad | [NV] |
| Capacidad | 8 GB es insuficiente para muchos juegos actuales junto a navegador/Discord; 16 GB es la base; 32 GB ayuda en títulos pesados y multitarea; 64 GB rara vez aporta en juegos | Diagnóstico por **presión real** (commit, hard faults), no por "uso alto" | [NV] |
| Presión de memoria | Indicadores: commit cerca del límite, *hard faults*/s sostenidos, memoria comprimida creciendo | PDH `\Memory\Pages Input/sec`, `\Memory\Committed Bytes`, `% Committed Bytes In Use` | [NV] |
| Standby / "limpiadores de RAM" | La lista standby es caché reutilizable; vaciarla fuerza relecturas de disco | **Descartado** (`PERFORMANCE_PLAN.md` y este encargo) | Regla del proyecto |
| Pagefile | Necesario para el commit; desactivarlo puede causar cierres | **Nunca desactivar** | Regla del encargo |
| iGPU | Usa RAM del sistema; dual channel es crítico | Priorizar diagnóstico de canales en equipos con iGPU | [NV] |

**¿Cuándo más frecuencia de RAM sube FPS?** Cuando el juego está limitado por CPU y el conjunto de trabajo
no cabe en L3 (CPUs sin X3D, juegos de simulación/multitudes). En X3D o en juegos limitados por GPU, el
beneficio suele ser chico [NV]. La app debe **medirlo** (experimento E5 en `10_IMPLEMENTATION_ROADMAP.md`).

---

## 4. Almacenamiento

| Tema | Hecho | Qué hace la app | Evidencia |
|---|---|---|---|
| NVMe vs SATA SSD vs HDD | Afecta tiempos de carga y *streaming* de texturas; en juegos de mundo abierto un HDD puede causar tirones; en FPS sostenidos el efecto suele ser nulo | Detectar tipo de unidad donde está instalado el juego (`Get-PhysicalDisk` requiere el módulo Storage) | [NV] |
| Espacio libre | SSD casi llenos rinden peor en escritura; Windows necesita espacio para actualizaciones y pagefile | Ya implementado (umbral 10/20 %, **pesos a documentar**) | [NV] |
| Salud / SMART / temperatura | Fallo inminente o *throttling* térmico de NVMe | `MSFT_StorageReliabilityCounter` (Temperature, Wear) [NV] | — |
| DirectStorage | API para cargas con descompresión en GPU; beneficio en cargas, depende del juego | Solo informativo | [NV] |
| Caché de shaders | Ver §2.2. **No borrar rutinariamente** | La limpieza actual la selecciona por defecto → **corregir (F0)** | Regla del encargo |
| Limpieza de temporales | **No aumenta FPS.** Solo libera espacio | Reformular textos: "liberar espacio", no "rendimiento" | Regla del encargo |

---

## 5. BIOS / UEFI y placa madre (Nivel 4: solo recomendaciones manuales)

| Elemento | Detección posible | Recomendación |
|---|---|---|
| Versión de BIOS | `Win32_BIOS` (PC de desarrollo: 1.3.8, 2021) | Comparar con la página del fabricante (manual); explicar riesgos de actualizar |
| AGESA / microcódigo | Parcial (versión de microcódigo vía registro) [NV] | Informativo |
| XMP/EXPO | Inferible (velocidad configurada vs nominal) | Guía manual + estabilidad |
| ReBAR / Above 4G | BAR1 en NVIDIA (`nvidia-smi -q`), soporte por GPU | Solo si GPU compatible (Ampere/RDNA2+ [NV]) |
| Generación PCIe | `nvidia-smi` link gen/width | Detectar slot mal elegido |
| Virtualización / seguridad (VBS, TPM, Secure Boot) | `Win32_DeviceGuard` | **Mantener activadas**; documentar costo |
| Ventiladores / límites térmicos | Sin interfaz confiable sin drivers | Solo guía |

**Verificación de estabilidad tras cambios de firmware/memoria** (manual, guiada): pruebas de memoria
(p. ej., Windows Memory Diagnostic incluido en Windows), prueba de estrés de CPU y una sesión de juego de
30–60 min vigilando WHEA (`Microsoft-Windows-WHEA-Logger`), reinicios de driver (evento 4101) y cierres
inesperados (evento 41). La app ya puede leer esos eventos (PC de desarrollo: 0 en 30 días).

---

## 6. Temperaturas y refrigeración

| Medición | Disponibilidad sin drivers de terceros | Postura |
|---|---|---|
| GPU NVIDIA (temp, potencia, ventilador, razones de límite) | Sí (`nvidia-smi`/NVML) | Implementar razones de límite (térmico/potencia) |
| GPU hotspot / VRAM | No vía `nvidia-smi` en GeForce de forma general [NV] | "No disponible" |
| CPU | **No** sin driver de kernel. Las zonas térmicas ACPI (`MSAcpi_ThermalZoneTemperature`) dieron 301–303 K (~28–30 °C) en la PC de desarrollo: **no representan el CPU** | No mostrar como temperatura de CPU |
| SSD | `MSFT_StorageReliabilityCounter` [NV] | Posible |

**Sobre LibreHardwareMonitor:** licencia MPL-2.0 [SEC: mirror]. En marzo de 2025 Microsoft Defender empezó
a marcar el driver **WinRing0** (CVE-2020-14979, lectura/escritura arbitraria de memoria) usado por varias
herramientas de monitoreo [OF] support.microsoft.com; LibreHardwareMonitor migró a **PawnIO**, que se
instala aparte [SEC]. Integrar un driver de kernel para leer la temperatura de CPU es una decisión de
**seguridad** (superficie de ataque, firma, anticheats) que requiere autorización explícita; se propone
como opcional y desactivado por defecto (ver 08).

**Diagnóstico térmico (diseño):** distinguir "límite normal" (GPU al 99 % a temperatura estable con
`clocks_event_reasons` = ninguno o "power cap" dentro del TDP) de "problema térmico" (caída de reloj con
razón `sw_thermal_slowdown`/`hw_thermal_slowdown` y temperatura en ascenso). Para CPU sin sensor: solo
"posible" si la frecuencia efectiva cae con el tiempo bajo carga constante. Prueba sostenida: 5, 15, 30 y
60 minutos con la misma escena; comparar FPS y relojes del minuto 1–5 vs 25–30.

---

## 7. Base de conocimiento por hardware (Parte 21)

Formato propuesto: archivos JSON versionados (`kb/v{N}/...`) con esquema validado y fecha de revisión.

```json
{
  "id": "cpu.amd.x3d.dual_ccd",
  "applies_to": { "cpu_vendor": "AMD", "cpu_model_regex": "Ryzen 9 (7900|7950|9900|9950)X3D" },
  "facts": [{ "text": "Solo un CCD tiene V-Cache; el driver de chipset estaciona el otro en juegos",
              "evidence": "SEC", "sources": ["SRC-AMD-VCACHE-FORUM"], "reviewed": "2026-11-08" }],
  "recommendations": [{ "optimization_id": "DIAG-CHIPSET-DRIVER", "priority": 1 }],
  "contraindications": ["OPT-CORE-PARKING-OFF", "OPT-POWERPLAN-ULTIMATE"],
  "valid_for": { "windows_min_build": 22000, "driver_min": null }
}
```

Claves de aplicabilidad: fabricante/modelo/arquitectura de CPU y GPU, núcleos, caché, VRAM, RAM
(capacidad, velocidad, canales), placa, BIOS, almacenamiento, refrigeración (si se conoce), monitor
(resolución, Hz, VRR), build de Windows, juego, resolución y calidad. Cada recomendación referencia IDs de
la matriz (06) y caduca si cambia la versión de driver/Windows indicada.

## 8. Rendimiento sostenible (Parte 22)

Indicadores a registrar en sesiones largas: FPS y 1 % low por ventana de 5 min, frecuencia efectiva
(CPU/GPU), potencia GPU, temperatura GPU, razones de límite, WHEA, eventos 4101 (TDR), cierres del juego.
Un ajuste que mejora el minuto 1 pero empeora el minuto 30 se clasifica como **regresión**.

## Fuentes consultadas en esta sesión

- Microsoft — core parking (`CPMinCores`, `CPConcurrency`): https://learn.microsoft.com/en-us/windows-hardware/customize/power-settings/options-for-core-parking-cpmincores
- Microsoft — HAGS, DirectX Developer Blog (30/06/2020): https://devblogs.microsoft.com/directx/hardware-accelerated-gpu-scheduling/
- Microsoft — alerta WinRing0: https://support.microsoft.com/windows/security/threat-malware-protection/microsoft-defender-antivirus-alert-vulnerabledriver-winnt-winring0
- NVIDIA — Understanding and Measuring PC Latency: https://developer.nvidia.com/blog/understanding-and-measuring-pc-latency
- NVIDIA — DLSS 4 Multi Frame Generation: https://www.nvidia.com/en-us//geforce/news/dlss4-multi-frame-generation-ai-innovations
- NVIDIA — NVML clocks event reasons: https://docs.nvidia.com/deploy/nvml-api/group__nvmlClocksEventReasons.html
- Nixxes — requisito de HAGS para DLSS FG: https://support.nixxes.com/hc/en-us/articles/48432027293971-How-to-use-DLSS-Frame-Generation
- TechPowerUp — RTX 50 deep dive (flip metering): https://techpowerup.com/review/nvidia-geforce-rtx-50-technical-deep-dive/4.html
- Tom's Hardware — Anti-Lag 2 / Anti-Lag+ y anticheats: https://www.tomshardware.com/pc-components/gpu-drivers/amd-introduces-radeon-anti-lag-2-with-full-integration-for-counter-strike-2-the-nvidia-reflex-alternative-shouldnt-trigger-cheating-bans-this-time · https://www.tomshardware.com/news/amds-anti-lag-triggers-anti-cheat-protection-in-multiple-games
- TechSpot / TechPowerUp — Resizable BAR: https://www.techspot.com/review/2234-nvidia-resizable-bar/ · https://techpowerup.com/review/nvidia-pci-express-resizable-bar-performance-test/27.html
- Guru3D / Tom's Hardware — KB5041587 (predictor de saltos Zen): https://guru3d.com/story/amd-ryzen-processor-branch-prediction-optimization-now-available-in-windows-11-23h2-kb-update · https://www.tomshardware.com/software/windows/microsoft-backports-branch-prediction-improvements-to-windows-11-23h2-more-users-will-see-ryzen-performance-improvements
- AMD 3D V-Cache / core parking (comunidad): https://community.amd.com/t5/pc-processors/amd-3d-v-cache-performance-optimizer-amp-amd-ppm-provisioning/m-p/598238 · https://hothardware.com/news/amds-chipset-driver-for-v-cache-cpus
- Intel APO (secundarias): https://www.techspot.com/news/101569-intel-backporting-frame-boosting-tech-select-past-generation.html
- KitGuru — PresentMon 2.3 (XeFG, XeLL, AFMF): https://www.kitguru.net/gaming/joao-silva/presentmon-2-3-adds-support-for-xefg-xell-and-amd-fluid-motion-frames/
