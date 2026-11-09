# 02 — Investigación de Windows: funciones, energía, servicios, registro y drivers

Convenciones: [OF] oficial consultado · [SEC] secundario consultado · [NV] no verificado en esta sesión.
Niveles (Parte 25): N1 seguras y comprobadas · N2 dependientes del hardware · N3 experimentales ·
N4 manuales avanzadas · N5 descartadas.

---

## 1. Funciones de Windows relevantes para juegos

| Función | Qué hace (mecanismo) | Impacto esperado | Riesgo / efectos secundarios | Nivel | Evidencia |
|---|---|---|---|---|---|
| **Modo Juego** | Prioriza el juego en primer plano, limita trabajo en segundo plano y evita instalaciones de drivers/notificaciones de Windows Update durante el juego. Objetivo declarado: **consistencia**, no FPS máximos | Pequeño; más notable en equipos cargados | Muy bajo; activado por defecto | N1 (dejar activado) | [SEC] PC Gamer/Windows Central (cita del equipo de Xbox) |
| **Game Bar / captura en segundo plano (Game DVR)** | Graba continuamente con el codificador de la GPU para "grabar lo último" | Pequeño (usa codificador y disco) | Ninguno al apagar la captura en segundo plano | N1 | [NV] |
| **Optimizaciones para juegos en ventana** (Win11) | Migra juegos DX10/DX11 en ventana/sin bordes del modelo *blt* al *flip*: menor latencia, permite VRR y Auto HDR. Puede producir *tearing*: Microsoft sugiere V-Sync o igualar FPS al refresco. Auto HDR lo activa automáticamente | Menor latencia en ventana sin bordes | Tearing en algunos juegos; excepción por app disponible | N1/N2 | [OF] DirectX Blog 16/02/2022, soporte de Microsoft |
| **Optimizaciones de pantalla completa (FSO)** | Windows trata la pantalla completa exclusiva como ventana *flip* independiente para permitir superposiciones rápidas | Desactivarlo **globalmente** no tiene respaldo oficial; puede romper VRR/HDR/overlays | Medio | N3 por juego / **N5 global** | [NV] |
| **HAGS** | El scheduler de la GPU gestiona su cola (WDDM 2.7+) | Variable; necesario para DLSS FG | Requiere reinicio; algunos drivers viejos con problemas | N2 | [OF] DirectX Blog; [SEC] |
| **VRR (Windows)** | Habilita VRR en juegos que no lo soportan nativamente | Fluidez si el monitor lo soporta | Bajo | N2 | [NV] |
| **Preferencia de GPU por app** | Elige iGPU/dGPU por ejecutable | Alto en portátiles híbridos si el juego corría en la iGPU; nulo si no hay iGPU | Bajo | N2 | [SEC] (formato de registro no documentado oficialmente) |
| **Frecuencia de refresco** | El monitor puede quedar en 60 Hz aunque soporte más | Alto en fluidez percibida | Bajo; reversible | **N1** | Diagnóstico local (PC de desarrollo: 60 de 75 Hz) |
| **Escalado / resolución** | Resolución no nativa + escalado del monitor agrega desenfoque | Calidad, no FPS | — | Diagnóstico | [NV] |
| **Aplicaciones de inicio** | Programas que arrancan con Windows consumen RAM/CPU | Variable; mayor en equipos con poca RAM | Bajo (reversible vía StartupApproved) | N2 | Implementado |
| **Integridad de memoria (HVCI) / VBS** | Aísla con el hipervisor la verificación de código del kernel. Microsoft publicó la guía "Options to optimize gaming performance in Windows 11" sugiriendo **desactivar HVCI y Virtual Machine Platform para jugar y volver a activarlos después**; no detalla el impacto exacto. Mediciones de terceros: desde "un par de FPS" hasta 5–15 % según hardware | **Seguridad**: protege contra drivers maliciosos | **No se desactiva por defecto** (regla del encargo). Se documenta el costo y se ofrece medirlo con consentimiento explícito y reactivación guiada | **N4** (decisión informada del usuario) | [SEC] Neowin, Winaero, Digital Trends, PCGH, landian (24H2) — página oficial no leída directamente |
| **Hyper-V / Windows Hypervisor Platform / WSL / VMP** | Activan el hipervisor; necesario para WSL2, Android, Docker | Similar a VBS | Romper herramientas del usuario | N4 | [SEC] |
| **Windows Defender** | Antivirus en tiempo real | Escaneos durante el juego pueden causar picos | **Nunca desactivar.** Alternativa segura: verificar que no haya escaneos programados en horario de juego | — (fuera de alcance) | Regla del encargo |
| **Windows Update** | Descargas/instalaciones en segundo plano | Picos de disco/red | Desactivar actualizaciones = riesgo de seguridad. Alternativa: horas activas, pausa temporal, ampliar el límite de pausa | N2 (pausa), **N5 (desactivar)** | [NV] |
| **Optimización de distribución** | Comparte actualizaciones P2P | Uso de subida mientras se juega | Bajo | N2 | [NV] |
| Drivers de chipset / almacenamiento | Provisión de energía (AMD), estacionamiento de CCD en X3D | Alto en X3D de dos CCD | Bajo | N2 (diagnóstico de versión) | [SEC] |
| Parche de predictor de saltos Zen (KB5041587 / 24H2) | Mejoras en Ryzen Zen 3/4/5 | Reportes de hasta ~11 % (Ryzen 7 9700X, 1080p) | Ninguno | N2 (verificar que esté instalado) | [SEC] |

---

## 2. Planes de energía

Ver `01_HARDWARE_RESEARCH.md §1.3`. Resumen de postura:

- No hay un plan "mejor" universal. **Se mide** (experimento E2).
- Detectar planes de terceros (la PC de desarrollo usa "Paragon Core Power Plan") y mostrarlos al usuario.
- En portátiles: no tocar el plan con batería; respetar el modo de energía del fabricante.
- En X3D de dos CCD: no usar planes que desactiven el estacionamiento.

---

## 3. Servicios y tareas programadas

**Hallazgo central:** no existe evidencia pública rigurosa de que desactivar servicios de Windows aumente
FPS en equipos actuales. El costo de un servicio inactivo es mínimo; el riesgo de romper funciones
(búsqueda, impresión, Xbox/Game Pass, Bluetooth, diagnósticos) es real. `PERFORMANCE_PLAN.md` ya los
descarta "en masa".

| Servicio / grupo | Afirmación popular | Realidad | Decisión |
|---|---|---|---|
| SysMain (Superfetch) | "Desactivalo en SSD" | Precarga apps usadas. En SSD el beneficio es chico; en algunos casos causa uso alto de disco. Microsoft (IoT) lo lista como "no desactivar". Sin evidencia de FPS | **N5** como optimización de FPS; solo diagnóstico si el servicio consume disco de forma anómala |
| Windows Search (WSearch) | "Desactivalo para ganar FPS" | Indexa en segundo plano con baja prioridad; desactivarlo degrada la búsqueda | **N5** para rendimiento (hoy en el catálogo como `svc_search_manual`) |
| DiagTrack / telemetría | "Consume recursos" | Impacto de rendimiento no demostrado; es una decisión de **privacidad** | Mantener como *privacidad*, nunca como *rendimiento* |
| Servicios de Xbox | "Desactivalos" | Rompen Game Pass, logros, guardado en la nube | **N5** para rendimiento |
| Cola de impresión, Server | "Innecesarios" | Inactivos casi no consumen; rompen impresión/compartir | Privacidad/superficie de ataque, no rendimiento |
| Autologgers (WMI) | "Menos escritura a disco" | Eliminan registros que sirven para diagnosticar fallas (incluido WiFiSession) | **N5** (va contra la Parte 22: diagnóstico de estabilidad) |
| Tareas de telemetría (CEIP, Compatibility Appraiser) | "Causan picos" | El Appraiser puede causar picos ocasionales de CPU/disco [NV] | N3: solo si se observa el pico durante sesiones |

---

## 4. Registro: ajustes populares analizados (Parte 5.2 y 15, formato A–J)

### 4.1 `NetworkThrottlingIndex` (MMCSS)

- **A. Qué modifica:** `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\NetworkThrottlingIndex`. Limita el procesamiento de paquetes de red mientras MMCSS atiende multimedia.
- **B. Documentación oficial:** la página de MMCSS en Microsoft Learn **no documenta este valor** (en los extractos consultados). Solo fuentes de terceros, que **no coinciden** en su semántica ni en el valor por defecto. Se atribuye su origen a Vista SP1.
- **C. Validez actual:** desconocida.
- **D/E. Beneficia / empeora:** sin datos confiables; terceros advierten posible inestabilidad con multimedia simultánea.
- **F. Seguridad:** ninguna.
- **G. Evidencia independiente:** anecdótica.
- **H. Reversible:** sí.
- **I. Medición:** PresentMon (1 % low) + ping/jitter durante streaming de audio.
- **J. Decisión:** **N5 — descartar** (coincide con `PERFORMANCE_PLAN.md`). En `go-rewrite` está implementado dentro de `net_throttle` → retirar del perfil y marcar "sin evidencia".

### 4.2 `SystemResponsiveness` (MMCSS)

- **A.** Mismo key. Porcentaje de CPU **garantizado a tareas de baja prioridad** frente a hilos registrados en MMCSS.
- **B.** **Documentado** [OF]: `REG_DWORD`; con 100, MMCSS se desactiva. Una página antigua indica que los valores no múltiplos de 10 se redondean; el código de OptimizerDuck documenta que valores por debajo de 10 se ajustan a 20 [SEC, comentario en código]. El valor por defecto es 20 [NV].
- **C.** Vigente para aplicaciones que registran hilos con MMCSS (audio profesional y algunos juegos).
- **D.** Solo hilos que usan `AvSetMmThreadCharacteristics` (p. ej., "Games", "Audio"); la mayoría de los juegos no lo hace [NV].
- **E.** Valores bajos pueden quitar CPU a tareas de fondo (audio, red).
- **H.** Sí. **I.** PresentMon + DPC/audio. 
- **J.** **N3** (experimental, solo con juegos que usen MMCSS). Nota: `go-rewrite` escribe **0**, que según lo anterior probablemente equivale a 20 (sin efecto) → corregir o retirar.

### 4.3 MMCSS `Tasks\Games` (`GPU Priority`, `Priority`, `Scheduling Category`, `SFIO Priority`) y `NoLazyMode`/`AlwaysOn`

- **A.** Parámetros de la tarea "Games" de MMCSS; `NoLazyMode`/`AlwaysOn` son valores **no documentados** en los extractos oficiales.
- **B.** La página oficial de MMCSS describe tareas y prioridades [OF], pero no `NoLazyMode`/`AlwaysOn`.
- **D.** Solo afecta a hilos registrados en la tarea "Games" [NV].
- **J.** `Tasks\Games`: **N3** (medir en juegos que registren MMCSS); `NoLazyMode`/`AlwaysOn`: **N5** hasta tener documentación. En `go-rewrite`: `games_priority` y `mmcss_extra`.

### 4.4 `Win32PrioritySeparation`

- **A.** `HKLM\SYSTEM\CurrentControlSet\Control\PriorityControl`. Tres campos de 2 bits: duración del *quantum* (corto/largo), fijo/variable y magnitud del *boost* de primer plano.
- **B.** Documentado en el blog oficial archivado de Microsoft ("Master your quantum") [OF]: el diálogo de Opciones de rendimiento escribe **0x26 para "Programas"** y 0x18 para "Servicios en segundo plano"; no requiere reinicio.
- **C.** Vigente.
- **D/E.** 0x26 = valor que Windows ya usa con "Programas" (comportamiento cliente). Cambiarlo a otros valores (0x28, 0x2A, 0x14…) no tiene evidencia sólida; foros (Blur Busters) muestran opiniones divergentes.
- **J.** **N5** como "optimización" (0x26 es el comportamiento por defecto en equipos cliente, por lo tanto `foreground_boost` en `go-rewrite` es, en el mejor caso, **sin efecto**). Podría ser **N3** probar valores no estándar con PresentMon, sin expectativas.

### 4.5 Resolución del temporizador (`timeBeginPeriod`, "timer resolution tools")

- **B.** [OF] Desde **Windows 10 2004** `timeBeginPeriod` **ya no cambia la resolución global**: la resolución elevada aplica a los procesos que la piden. En Windows 11, para procesos con ventana totalmente ocultos o minimizados no se garantiza la resolución elevada. Microsoft marca los temporizadores multimedia como heredados y apunta a MMCSS. Un análisis independiente (2020) sostiene que el intervalo de interrupción sigue siendo global y que lo que cambió es el comportamiento de `Sleep` en otros procesos → **conflicto de fuentes documentado**.
- **D.** Los juegos que lo necesitan ya lo piden para su propio proceso.
- **J.** **N5** para herramientas que "fuerzan" resolución global (además, impacto en consumo).

### 4.6 HPET / `useplatformclock` / `disabledynamictick` / `useplatformtick` (bcdedit)

- **B.** La documentación de `bcdedit` describe estas opciones; varias están marcadas como **solo para depuración** [OF vía citas secundarias]. La documentación no promete FPS ni menor latencia.
- **E.** Reportes mixtos: peor *frame pacing* y más latencia en algunos sistemas; un error en el BCD puede impedir el arranque.
- **J.** **N5 — descartar** (coincide con `PERFORMANCE_PLAN.md`).

### 4.7 Modo MSI (Message-Signaled Interrupts)

- **A.** `Enum\...\Device Parameters\Interrupt Management\MessageSignaledInterruptProperties\MSISupported`.
- **B.** [OF] Microsoft documenta que **el INF del driver** habilita MSI durante la instalación con ese valor (`REG_DWORD`, 1 = habilitado) y `MessageNumberLimit`. Es una decisión del fabricante del driver.
- **C.** Las GPUs modernas ya usan MSI/MSI-X con sus drivers [NV].
- **E.** Forzar MSI en un dispositivo cuyo driver no lo soporta puede causar pantallas azules o pérdida del dispositivo.
- **J.** **N5** (forzar). Diagnóstico de solo lectura (¿el dispositivo usa MSI?) puede ser **N3** informativo.

### 4.8 Afinidad de interrupciones / afinidad de procesos

- **J.** **N5** global. Afinidad de proceso por juego: **N3** solo para casos documentados (p. ej., X3D de dos CCD cuando el driver falla, como diagnóstico) y nunca en juegos con anticheat que lo prohíban [NV].

### 4.9 TCP: `TcpAckFrequency`, `TCPNoDelay` (Nagle)

- **B.** [OF] KB 328890: `TcpAckFrequency` por interfaz, rango 0–255, por defecto 2; Microsoft **no recomienda cambiarlo** sin estudiar el entorno. `TCPNoDelay` no figura en los extractos oficiales consultados.
- **C/D.** Nagle y el ACK diferido aplican **solo a TCP**. La gran mayoría de los juegos competitivos usa **UDP** para el tráfico de juego [NV — conocimiento general]; los ajustes solo afectarían lobby, login o descargas.
- **J.** **N5** (coincide con `PERFORMANCE_PLAN.md`).

### 4.10 Prioridad de proceso del juego

- **A.** `SetPriorityClass` / IFEO `PerfOptions\CpuPriorityClass`.
- **E.** "Tiempo real" puede bloquear entrada/audio/drivers; "Alta" puede quitar CPU a audio/voz/streaming.
- **J.** "Alta" durante la sesión y revertida al cerrar: **N3** (medir). "Tiempo real": **N5**. Juegos con anticheat: no tocar procesos protegidos.

### 4.11 Vaciar la lista standby / "limpiar RAM"

- **J.** **N5** (regla del encargo y de `PERFORMANCE_PLAN.md`).

### 4.12 Desactivar fullscreen optimizations globalmente (`GameDVR_FSEBehavior*` en `GameConfigStore`)

- **B.** Valores no documentados oficialmente [NV]; el mecanismo oficial es la casilla por ejecutable en Propiedades → Compatibilidad.
- **J.** Global: **N5**. Por juego: **N3** (medir). En `go-rewrite`: `fullscreen_opt` es global → retirar del perfil.

### 4.13 `DisableDynamicPstate` (NVIDIA) / `EnableULPS` (AMD) / claves de driver

- **A.** Valores en la clave de clase del adaptador; afectan la gestión de energía del driver.
- **B.** No documentados públicamente por NVIDIA/AMD [NV]. Provienen de herramientas de terceros (OptimizerDuck los marca Safe/Moderate).
- **E.** Mayor consumo y temperatura en reposo; comportamiento puede cambiar entre versiones de driver.
- **J.** **N3** (experimental, con medición de consumo y temperatura). En `go-rewrite`: `nv_pstate`, `amd_ulps`.

### 4.14 `SvcHostSplitThresholdInKB`

- **A.** Agrupa servicios en menos procesos `svchost.exe`.
- **J.** Sin relación demostrada con FPS; afecta aislamiento de servicios (si un servicio falla arrastra a otros). **N5** para rendimiento. En `go-rewrite`: `svchost_split`.

### 4.15 Desactivar actualizaciones, Defender, mitigaciones, UAC, firewall, debloat scripts y "packs de FPS"

- **J.** **N5** sin excepción (seguridad, regla del encargo). Los scripts de "debloat" desinstalan componentes de forma **irreversible** y ejecutan código no auditado.

---

## 5. Drivers

| Elemento | Qué hace la app |
|---|---|
| Versión y fecha del driver de GPU | Detectar (ya disponible vía `nvidia-smi`/registro) y comparar contra la versión mínima recomendada en la KB por juego/arquitectura |
| Driver de chipset AMD (provisión de energía, 3D V-Cache) | Detectar presencia/versión; recomendar actualizar desde AMD (manual) |
| Intel DTT / APO | Detectar en Windows 11 con CPU compatible; informar |
| Driver de red | Detectar versión y opciones de ahorro de energía (ver 04) |
| Recuperación de driver (TDR) | Contar eventos 4101 por sesión (estabilidad) |

## Fuentes consultadas en esta sesión

- MMCSS (Microsoft Learn): https://learn.microsoft.com/en-us/windows/win32/procthread/multimedia-class-scheduler-service
- "Master your quantum" (Microsoft, archivado): https://learn.microsoft.com/en-us/archive/blogs/embedded/master-your-quantum
- `timeBeginPeriod` (Microsoft Learn): https://learn.microsoft.com/en-us/windows/win32/api/timeapi/nf-timeapi-timebeginperiod · Timer Resolution: https://learn.microsoft.com/en-us/windows/win32/multimedia/timer-resolution
- Análisis independiente del cambio de temporizador (2020): https://hacklines.com/en/editions/206755/click
- MSI (Microsoft Learn): https://learn.microsoft.com/windows-hardware/drivers/kernel/enabling-message-signaled-interrupts-in-the-registry
- KB 328890 TcpAckFrequency: https://support.microsoft.com/kb/328890
- Optimizaciones para juegos en ventana: https://devblogs.microsoft.com/directx/updates-in-graphics-and-gaming/ · https://support.microsoft.com/topic/3f006843-2c7e-4ed0-9a5e-f9389e535952
- Guía de Microsoft sobre HVCI/VMP para juegos (cobertura): https://www.neowin.net/amp/disable-these-features-to-improve-gaming-performance-in-windows-11/ · https://winaero.com/heres-how-to-improve-gaming-performance-on-windows-11 · https://landian.news/article/3401.html · https://digitaltrends.com/computing/are-windows-11-security-features-killing-gaming-performance
- Game Mode (cobertura): https://pcgamer.com/microsoft-explains-how-windows-10-game-mode-will-improve-performance-in-uwp-and-win32-games
- bcdedit / HPET (citas y foros): https://www.techbloat.com/bcdedit-tweaks-on-windows-10-which-are-worth-trying-and-which-arent.html · https://forums.blurbusters.com/viewtopic.php?p=116451
- Preferencia de GPU por app (secundarias): https://elevenforum.com/t/change-graphics-performance-preference-for-apps-in-windows-11.5675/
- SysMain (secundarias): https://blog.gridinsoft.com/sysmain-high-disk-usage/ · https://www.askwoody.com/forums/topic/disable-superfetch-if-using-an-ssd-as-system-disk
- Plan Máximo rendimiento (secundarias): https://www.howtogeek.com/368781/how-to-enable-ultimate-performance-power-plan-in-windows-10/
- KB5041587: https://www.tomshardware.com/software/windows/microsoft-backports-branch-prediction-improvements-to-windows-11-23h2-more-users-will-see-ryzen-performance-improvements
