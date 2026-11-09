# Adaptación de «Ultimate» (FR33THY) a South Tweaks

Fuente: https://github.com/FR33THYFR33THY/Ultimate (licencia MIT, ver `THIRD-PARTY-NOTICES.md`). Se **leyeron** los scripts; no se
ejecutó ninguno. Ultimate es una colección de scripts de PowerShell interactivos (opción 1 aplicar / 2 revertir) que, entre otras
cosas, descargan programas, compilan servicios y desactivan protecciones. South Tweaks toma solo lo que cumple las reglas del proyecto:
registro, reversible con backup del valor original, sin debilitar la seguridad y sin ejecutar código descargado.

## Adaptado (8 ajustes nuevos)

| Ajuste en South Tweaks | Script de Ultimate | Notas |
|---|---|---|
| Optimizaciones para juegos en ventana (flip) | `8 Advanced\11 Mpo.ps1` | Es la opción oficial de Windows 11. Ultimate además pone `VRROptimizeEnable=0`; **no** se adopta (apaga VRR en ventana). Se fusiona con las claves existentes en vez de pisarlas. |
| Mouse sin límite de sondeo en segundo plano | `7 Hardware\2 Background Polling Rate Cap.ps1` | `RawMouseThrottleEnabled=0` |
| Puntero 1:1 sin aceleración (raw aim) | `7 Hardware\1 Scaling Higher No Accel.ps1` | Curvas lineales para escala 100 %. La curva X del script original tiene **81 caracteres** (un dígito de más): como `REG_BINARY` ese valor no se puede escribir, así que ahí el comando original falla en silencio. Aquí se usa la curva correcta de 80 caracteres y una prueba lo verifica. |
| Desactivar Multiplane Overlay (MPO) | `8 Advanced\11 Mpo.ps1` | `OverlayTestMode=5`. Solución para parpadeos/tirones, no una mejora general. |
| Flip inmediato forzado (experimental) | `8 Advanced\13 Hardware Composed Independent Flip.ps1` | Ultimate revierte borrando toda la clave `Scheduler`; South Tweaks solo toca el valor propio. |
| Adaptador de red sin ahorro de energía | `6 Windows\26 Network Adapter Power Savings & Wake.ps1` | Solo adaptadores físicos (Ethernet/Wi-Fi), sin virtuales, VPN ni Bluetooth; los valores EEE solo si el driver los tiene. |
| Prohibir apps en segundo plano (política) | `3 Setup\9 Background Apps.ps1` | Versión de administrador del ajuste de usuario que ya existía. |
| Pausar Windows Update 35 días | `3 Setup\12 Updates Pause.ps1` | Ultimate pausa **365 días**; aquí 35 (el máximo de la interfaz de Windows). Las fechas se calculan al aplicar y vuelve el valor anterior al revertir. |

Ya estaban cubiertos por ajustes propios: plan de energía (`29 Power Plan`), hibernación, Inicio rápido y Power Throttling
(`29 Power Plan`), ULPS de AMD (`14 Ulps`), estado P0 de NVIDIA (`9 P0 State`), Widgets/Noticias (`8 Widgets`), Copilot
(`9 Copilot`), apps en segundo plano del usuario.

## No adaptado, y por qué

| Script | Motivo |
|---|---|
| `3 Spectre Meltdown`, `4 Data Execution Prevention`, `5 File Download Security Warning`, `15 Driver Whql Secure Boot Bypass` | **Debilitan la seguridad.** Regla del proyecto: las protecciones permanecen activadas. |
| `6 Windows\33 Defender Optimize`, `20 Edge & WebView`, `13 Bloatware` | Desactivan o desinstalan componentes de forma **irreversible** o reducen la protección. |
| `30 Timer Resolution` | Compila e instala un servicio de Windows con código C#. Es código arbitrario con privilegios; y desde Windows 10 2004 el efecto global es discutido (ver `docs/research/02`). |
| `10 Msi Mode`, `25 Device Manager Power Savings & Wake` | Forzar MSI puede dejar un dispositivo inutilizable; el ahorro de energía de dispositivos recorre y modifica todos los dispositivos USB/PCI (a revisar con más cuidado). |
| `28 Write Cache Buffer Flushing` | `CacheIsPowerProtected=1` le dice a Windows que el disco tiene protección contra cortes de luz: **riesgo de pérdida de datos**. |
| `19 NVME Faster Driver` | Activa funciones experimentales con claves de `FeatureManagement` y agrega entradas de arranque seguro. |
| `6 MMAgent Features`, `18 Start Search Shell Mobsync` | Desactivan prefetch, compresión de memoria y Windows Search: sin evidencia de FPS y con costo en uso diario (ver `docs/research/02 §3`). |
| `5 Nvidia Settings`, `2–4 Driver ... Install`, `4 Installers` | Descargan y ejecutan programas externos (Profile Inspector, instaladores). South Tweaks usa la API oficial NvAPI. `RmProfilingAdminOnly=0` reabre los contadores de GPU a usuarios sin permisos (se considera un retroceso de seguridad). |
| `10 Priority` | Permite prioridad «Tiempo real» y cierra launchers. South Tweaks no cierra procesos ni usa Tiempo real. |
| `1 Check`, `2 Refresh` (reinstalar, reset, claves, activación), `5 Graphics\13 Resolution Refresh Rate`, `8 Smt Ht` | Fuera de alcance o requieren una función propia (el cambio de Hz está en el plan de investigación). |

Todos los ajustes adaptados siguen siendo hipótesis de rendimiento: **no están medidos con FPS**. La medición con PresentMon es la
prioridad 0 del plan (`docs/research/10_IMPLEMENTATION_ROADMAP.md`).