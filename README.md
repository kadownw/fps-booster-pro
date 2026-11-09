# South Tweaks

Ajustes **reversibles** de Windows para juegos, en un único `.exe` (Go + Wails). Sin Python ni runtimes: abrís y listo (usa WebView2, que ya viene con Windows 10/11).

- Detecta CPU, GPU (con VRAM), RAM y Windows, y muestra el uso en vivo.
- 75 ajustes en 10 categorías (Energía, Juegos, Entrada, Visual, Sistema, GPU, Red, Privacidad, Servicios y Acciones únicas): registro, planes de energía, servicios, tareas programadas, loggers de diagnóstico y perfil del driver NVIDIA.
- **South's Core power plan**: plan de energía de rendimiento (procesador siempre disponible con corriente, boost agresivo, núcleos sin estacionar, sin ahorro en discos/USB/PCIe/Wi-Fi). Con batería conserva el ahorro. Respeta el estacionamiento de núcleos en CPUs híbridas y Ryzen X3D de dos CCD. Al revertir vuelve el plan anterior y se borra.
- **Perfil NVIDIA** vía NvAPI oficial: energía "Preferir rendimiento máximo", baja latencia (1 frame en cola) y caché de shaders activada, con restauración de cada valor previo.
- Sin ajustes repetidos: lo que hacía lo mismo está unido en uno solo (por ejemplo, el perfil global de NVIDIA con máximo rendimiento y baja latencia, la telemetría con sus tareas, servicios y registradores, o el bloqueo de apps en segundo plano con su política). Si ya tenías aplicado alguno de los ajustes que se unieron, su backup pasa solo al nuevo y se revierte desde su tarjeta.
- Ajustes adaptados de [Ultimate](https://github.com/FR33THYFR33THY/Ultimate) (FR33THY, MIT): ver [`docs/ULTIMATE_ADAPTATION.md`](docs/ULTIMATE_ADAPTATION.md) y [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md).
- Investigación técnica, matriz de optimizaciones y plan por fases en [`docs/research/`](docs/research/).
- Antes de tocar nada guarda el valor original; verifica cada cambio y hace rollback si falla.
- Páginas:
  - **Inicio**: todo en una página: panel de salud con medidores que cambian de verde a amarillo y rojo (CPU, GPU, RAM, almacenamiento), atajos, gráfico en tiempo real (1 o 5 minutos) y detalle de procesador, GPU, memoria y discos.
  - **Mis juegos**: una portada por cada juego que jugaste (portadas de Steam; el resto usa el icono del juego). Al pasar el mouse se ve la media de FPS de la última sesión y los 0.1 % y 1 % low. Los FPS se miden con eventos ETW de Windows, sin programas externos y solo leyendo: DXGI para DirectX y el evento Present_Info de DxgKrnl para Vulkan y OpenGL. Requieren ejecutar como administrador. Validado con DirectX (compositor de Windows) y con un programa OpenGL de prueba (`tools/gltest`, error de ~4 % contra su propio contador); Vulkan usa el mismo evento pero no se probó con un juego real.
  - **Optimizar**: elegís los ajustes (activar o desactivar), abajo ves cuántos procesos hay por aplicar y, al tocar Aplicar, una pantalla de carga los ejecuta uno por uno. **Limpieza**: muestra el tamaño antes de borrar. **Inicio de Windows**: activar o desactivar programas de arranque, con el icono de cada uno.
- **Inicio de sesión con Discord**: la tarjeta de abajo a la izquierda muestra tu foto y tu tier (**Free Tier**, **Pro Tier** o **Propietario**). Configuración única: creá una aplicación en <https://discord.com/developers/applications>, agregá en OAuth2 → Redirects `http://127.0.0.1:53682/callback` y pegá su Application ID cuando la app lo pida. Solo se pide el permiso `identify` y no se guarda ningún token. El tier sale de [`tiers.json`](tiers.json) (IDs de Discord en `owners` y `pro`).
- **Acciones únicas** (TRIM de SSD, vaciar DNS): no tienen interruptor porque no se pueden deshacer; llevan botón **Aplicar** con confirmación y registran la última ejecución.
- Interfaz: tarjetas por categoría con icono propio e insignias NVIDIA / AMD / Intel en los ajustes exclusivos de cada fabricante (insignias propias, no son los logos oficiales).
- Datos en vivo sin drivers de terceros: contadores de Windows, `nvidia-smi` para la GPU NVIDIA. La temperatura de la CPU no se muestra porque Windows no la expone sin un driver de hardware.
- Botón **Revertir todo** y **modo simulación** (no modifica nada).
- No cierra procesos ni promete FPS: cada ajuste indica su riesgo y si necesita administrador o reinicio.
- Los datos de backup y el registro de cambios quedan en `%LOCALAPPDATA%\SouthTweaks\` (si existía la carpeta anterior `FPSBoosterPro`, se migra sola).

## Descargar

El ejecutable ya compilado está en [dist/South-Tweaks.exe](dist/South-Tweaks.exe). Doble clic y listo.

## Compilar

Requisitos: Go 1.23+, Node.js y [Wails v2](https://wails.io).

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build -trimpath  # genera build\bin\South-Tweaks.exe (sin rutas de tu equipo dentro del exe)
go test ./...          # prueba aplicar/revertir contra HKCU real y la simulación
```

## Estructura

| Archivo | Rol |
|---|---|
| `tweaks.go` | Catálogo de ajustes (registro / powercfg) |
| `engine.go` | Aplicar, verificar, backup, rollback y revertir |
| `regops.go` | Lectura/escritura de registro y powercfg (solo HKCU/HKLM, GUID validado) |
| `telemetry.go`, `games.go` | Datos en vivo (PDH, nvidia-smi) y detector de juegos |
| `profiles.go` | Perfiles de un clic y punto de restauración |
| `cleanup.go` | Limpieza de temporales (lista fija, sin seguir enlaces, respeta archivos recientes) |
| `startup.go` | Programas de inicio (StartupApproved, reversible) |
| `tasks.go` | Tareas programadas (consulta y cambio con PowerShell, rutas validadas) |
| `tweaks_extra.go`, `tweaks_services.go` | Más ajustes: privacidad, servicios, loggers |
| `sysinfo.go` | Hardware y uso en vivo (sin PowerShell) |
| `frontend/` | Interfaz |
| `tools/snapshot.ps1` | Compara el sistema antes/después de aplicar un ajuste, para descubrir qué cambia |

> Los ajustes que usan HKLM requieren ejecutar como administrador (la app ofrece reiniciarse con UAC).


