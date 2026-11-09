# 11 · Aplicaciones parecidas, fuentes de optimizaciones y dónde pedir feedback

Investigación hecha el 2026-11-09 con búsqueda web y lectura de las páginas oficiales de cada proyecto. Cada dato dice de dónde sale. **No se probó ninguna de estas herramientas ni ninguno de estos ajustes en una PC**; todo lo de abajo es lo que afirman sus autores o terceros. Las páginas de fabricantes que venden su propio producto están marcadas como no neutrales.

## 1. Aplicaciones parecidas

| Herramienta | Qué es | Qué hace que nosotros no | Qué hacemos nosotros que ella no | Fuente |
|---|---|---|---|---|
| **WinUtil** (Chris Titus) | Script de PowerShell, MIT, se ejecuta con `irm … \| iex` | Instala programas, presets Standard/Minimal/Advanced | Verificación de cada cambio, rollback completo, interfaz propia | [GitHub](https://github.com/ChrisTitusTech/winutil) |
| **Winhance** | App de escritorio (WinUI 3), código abierto | Categorías de energía, juegos, privacidad y Windows Update; punto de restauración propio. Sus docs dicen que **no reaplica** los ajustes si Windows Update los cambia | Medición de FPS, detección de juegos | [GitHub](https://github.com/memstechtips/Winhance) · [docs](https://winhance.net/docs/features/optimize.html) · [reseña](https://www.windowscentral.com/software-apps/i-tried-winhance-to-debloat-and-optimize-my-windows-11-pc) |
| **Sophia Script / SophiApp** | Más de 150 funciones, cada una con su función de restaurar | Quitar OneDrive, DNS sobre HTTPS, mover carpetas de usuario, tareas de limpieza programadas | Nivel de riesgo por ajuste, explicaciones leídas del sistema | [GitHub](https://github.com/farag2/Sophia-Script-for-Windows) · [SophiApp](https://alternativeto.net/software/sophiapp/) |
| **optimizerDuck** | App con 42 ajustes en 7 categorías y riesgo Seguro/Moderado/Riesgoso | Tiene su propio Discord; gestor de inicio, tareas programadas y desinstalador de bloatware | Verificación posterior y FPS medidos | [GitHub](https://github.com/itsfatduck/optimizerDuck) |
| **O&O ShutUp10++** | Casi 300 opciones de privacidad en más de 20 categorías, con nivel de recomendación | Su FAQ avisa que Windows puede restaurar la telemetría tras una actualización grande | — | [manual](https://manuals.oo-software.com/ooshutup10/docs/features/overview/) · [FAQ](https://manuals.oo-software.com/ooshutup10/docs/faq) |
| **AtlasOS / ReviOS** | Una imagen entera de Windows modificada | Cambios muy profundos | Nosotros no pedimos reinstalar; la reversión de ellos es solo reinstalando | [tier1settings (no neutral)](https://tier1settings.com/best-pc-optimization-tools-for-gaming/) |
| **Razer Cortex, Wise Game Booster** | «Boosters» que cierran procesos | Cierran procesos al abrir un juego | Nosotros no cerramos procesos ni prometemos FPS | [AlternativeTo](https://alternativeto.net/software/razer-cortex/) |
| **Process Lasso** | Prioridad y afinidad de CPU por proceso | Automatiza prioridades (ProBalance); funciones persistentes de pago | — | [alternativas](https://alternativeto.net/software/process-lasso/) |
| **NVIDIA Profile Inspector** | Edita el perfil del driver NVIDIA (MIT) | Muchos más ajustes del driver | Usamos la API oficial con restauración | [GitHub](https://github.com/Orbmu2k/nvidiaProfileInspector) |
| **Hone, Tier1Suite** | Productos comerciales | — | — | referencias en los resultados de búsqueda; no se evaluaron |

Medición, no optimización: **PresentMon, FrameView, CapFrameX, LatencyMon**. La guía [PC-Tuning](https://github.com/valleyofdoom/PC-Tuning) las recomienda para medir antes de tocar nada.

## 2. Lo que dicen las comunidades (relevante para nosotros)

- **r/OptimizedGaming** es escéptica con las herramientas de ajustes: hay un hilo fijado de un moderador que advierte contra ellas ([espejo](https://redlib.hbubli.cc/r/OptimizedGaming/comments/1kye9u0/warning_avoid_any_tweak_or_optimization_tools)). Lo que esa comunidad valora: código abierto, decir exactamente qué cambia y poder revertirlo.
- **valleyofdoom / BoringBoredom (PC-Tuning y PC-Optimization-Hub)** insisten en **medir** y **no seguir a ciegas**. La guía desaconseja los scripts de «debloat» y avisa que su propio script de registro **no se puede revertir**. Solo leí las primeras 39 mil caracteres de ~188 mil; el resto es un resumen generado.
- **El ejemplo de Memory Integrity / VBS:** Tom's Hardware midió entre 3 % y 8 % en pruebas con CPU Intel (11700K y 10700K) y alrededor de 4 % en un Ryzen 5000 ([nota](https://www.tomshardware.com/news/windows-11-gaming-benchmarks-performance-vbs-hvci-security/)). HotHardware, sobre 15 juegos, midió cerca de 5 % ([nota](https://hothardware.com/news/default-windows-setting-choke-game-performance-on-flagship-gpus)). PCWorld y Digital Trends no vieron una caída drástica. Son pruebas de CPUs de 2021; con CPUs más nuevas el efecto puede ser menor.
- **Los anticheats ya exigen esas protecciones:** [FACEIT](https://support.faceit.com/hc/en-us/articles/23117181142556) pide VBS para IOMMU, y [Vanguard](https://www.tomshardware.com/video-games/pc-gaming/riot-vanguard-adds-an-on-demand-mode-that-stops-anti-cheat-loading-at-boot-on-secured-windows-11-pcs) pide Secure Boot, TPM 2.0, VBS, HVCI e IOMMU. Los requisitos cambian en 2026; hay que confirmarlos en las páginas oficiales.

## 3. Ajustes que ellos tienen y nosotros no (cruce con nuestro catálogo de 75)

Comprobé con búsqueda en el código que **ninguna** de estas claves existe en nuestros ajustes (salvo `StartupDelayInMSec`, que ya tenemos).

| Candidato | Dónde aparece | Qué es | Evidencia | Riesgo | Prioridad |
|---|---|---|---|---|---|
| **Excluir drivers de Windows Update** (`ExcludeWUDriversInQualityUpdate`) | Winhance | Evita que Windows Update pise el driver de la GPU | Solo afirmaciones de los autores | Bajo, reversible | **Alta** |
| **VRR en juegos con ventana** (`VRROptimizeEnable`) | Winhance; [Microsoft](https://support.microsoft.com/en-us/windows/hardware/display-graphics/optimizations-for-windowed-games-in-windows-11) | Sumar la opción a nuestro ajuste de juegos en ventana | Documentado por Microsoft (flip model y VRR) | Bajo | **Alta** |
| **Mantenimiento automático** (`MaintenanceDisabled`) | Winhance | Evita tareas de mantenimiento mientras jugás | Sin pruebas | Bajo-medio | Media |
| **Impedir el cifrado automático de BitLocker** (`PreventDeviceEncryption`) | Winhance | Complementa «Quitar BitLocker» | Sin pruebas de rendimiento | Medio (seguridad) | Media |
| **Nagle / TcpAckFrequency / TCPNoDelay** | Winhance | Latencia de red en algunos juegos | Sin pruebas verificadas en las fuentes que leí | Bajo-medio | Media (medir antes) |
| **Prefetch** (`EnablePrefetcher`) | Winhance | Para SSD | Sin pruebas | Bajo | Baja |
| **Último acceso a archivos / nombres 8.3** | PC-Tuning (resumen) | Menos escrituras en disco | Sin pruebas | Medio (afecta programas de backup) | Baja |
| **Bloqueo de copia de OneDrive** (`KFMBlockOptIn`) | Winhance | Privacidad | — | Bajo | Baja |
| **Quitar apps preinstaladas** (lista fija: Clipchamp, Noticias, etc.) | optimizerDuck, Winhance | Acción única con presencia verificada, como la de Copilot | — | Medio (no se deshace) | Media |
| **DNS sobre HTTPS** | Sophia | Privacidad y DNS cifrado | — | Medio | Baja |
| **Alt+Tab filter, Storage Sense** | Winhance | Cosmético | — | Bajo | Baja |

No recomiendo agregar: desactivar Defender, mitigaciones de Spectre/Meltdown, VBS/HVCI, TPM o Secure Boot. Los anticheats los exigen y la ganancia, cuando existe, es chica y no uniforme.

## 4. Funciones nuevas que me parecen más valiosas (no son ajustes)

1. **Antes y después con FPS medidos.** Ya medimos FPS; ninguna de las herramientas que revisé mide el efecto de sus propios cambios. Sería una diferencia real y responde al escepticismo de r/OptimizedGaming.
2. **Detección de cambios (drift).** Winhance y O&O admiten que Windows Update puede deshacer los ajustes. Nuestra lectura al abrir Optimizar ya sabe qué quedó parcial; falta avisar «Windows cambió X ajustes desde la última vez».
3. **Comprobador de compatibilidad con anticheats (solo lectura):** estado de Secure Boot, TPM, VBS, HVCI e IOMMU, para saber si Vanguard y FACEIT van a arrancar.
4. **Prioridad automática del juego detectado** (estilo Process Lasso). Hay que probar que no moleste a anticheats; las fuentes dicen que no, pero no lo verifiqué.
5. **Prueba de compatibilidad con anticheats de nuestro propio «máximo rendimiento».** Su lista incluye servicios como `SgrmBroker` (System Guard). No sé si algún anticheat depende de él; es una **hipótesis** que habría que probar con Vanguard, FACEIT, EAC y BattlEye antes de recomendarlo.

## 5. Dónde pedir feedback

| Lugar | Para qué | Condición |
|---|---|---|
| [r/OptimizedGaming](https://redlib.hbubli.cc/r/OptimizedGaming/comments/1d8nfsf/i_released_a_free_opensource_tool_to_build_a/l7eg67o/?context=3) | Presentar la herramienta | Escéptica: hay que decir qué cambia y cómo revertir |
| [Foro de Chris Titus (WinUtil)](https://forum.christitus.com/c/winutil-support/5) | Feedback técnico de gente que ya usa estas herramientas | Hay un hilo de un fork con pestaña de juegos que pide justo qué ajustes son imprescindibles; ese hilo no tenía respuestas cuando lo leí ([hilo](https://forum.christitus.com/t/project-gaming-centric-fork-of-winutil-seeking-feedback-on-new-gaming-tab/38)) |
| Issues y Discussions de Winhance y optimizerDuck | Leer qué piden los usuarios y qué falla | Solo lectura al principio |
| Discord propio | Soporte y pruebas tempranas | optimizerDuck y otros lo usan así |

**Antes de pedir feedback público** conviene tener: un archivo `LICENSE` (no hay uno en el repo y usamos código MIT adaptado de Ultimate), la política de privacidad del inicio de sesión con Discord, y la lista de qué cambia cada ajuste, que ya tenemos en el README y en la app.

## 6. Limitaciones

- No pude leer la guía de Microsoft vía TechSpot (error 403); lo de Microsoft sobre juegos en ventana viene de su página de soporte.
- Los datos de rendimiento de sitios de guías (por ejemplo «15–25 % más FPS») no traen pruebas y no los uso.
- Varias comparaciones de Winhance, Atlas y boosters vienen de resultados de blogs y foros; son direccionales.
- No hay ningún benchmark independiente que compare estas herramientas entre sí en FPS o latencia.
