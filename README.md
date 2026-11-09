# South Tweaks

Ajustes **reversibles** de Windows para juegos, en un Ãºnico `.exe` (Go + Wails). Sin Python ni runtimes: abrÃ­s y listo (usa WebView2, que ya viene con Windows 10/11).

- Detecta CPU, GPU (con VRAM), RAM y Windows, y muestra el uso en vivo.
- 73 ajustes en 10 categorÃ­as (EnergÃ­a, Juegos, Entrada, Visual, Sistema, GPU, Red, Privacidad, Servicios y Acciones Ãºnicas): registro, planes de energÃ­a, servicios, tareas programadas, loggers de diagnÃ³stico y perfil del driver NVIDIA.
- **South's Core power plan**: plan de energÃ­a de rendimiento (procesador siempre disponible con corriente, boost agresivo, nÃºcleos sin estacionar, sin ahorro en discos/USB/PCIe/Wi-Fi). Con baterÃ­a conserva el ahorro. Respeta el estacionamiento de nÃºcleos en CPUs hÃ­bridas y Ryzen X3D de dos CCD. Al revertir vuelve el plan anterior y se borra.
- **Perfil NVIDIA** vÃ­a NvAPI oficial: energÃ­a "Preferir rendimiento mÃ¡ximo", baja latencia (1 frame en cola) y cachÃ© de shaders activada, con restauraciÃ³n de cada valor previo.
- Ajustes adaptados de [Ultimate](https://github.com/FR33THYFR33THY/Ultimate) (FR33THY, MIT): ver [`docs/ULTIMATE_ADAPTATION.md`](docs/ULTIMATE_ADAPTATION.md) y [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md).
- InvestigaciÃ³n tÃ©cnica, matriz de optimizaciones y plan por fases en [`docs/research/`](docs/research/).
- Antes de tocar nada guarda el valor original; verifica cada cambio y hace rollback si falla.
- PÃ¡ginas:
  - **Inicio**: panel de salud con medidores de marcas que cambian de verde a amarillo y rojo segÃºn la carga o la temperatura (CPU, GPU, RAM, almacenamiento), atajos y perfiles de un clic (Gaming, RÃ¡pido sin admin, Privacidad) con punto de restauraciÃ³n opcional.
  - **Mi PC**: grÃ¡fico en tiempo real (1 o 5 minutos) y detalle de procesador, GPU, memoria y discos.
  - **Mis juegos**: detecta juegos (Steam, Epic, Riot, Xbox, GOG, Ubisoft, EA, Rockstar, Roblox) y guarda cada sesiÃ³n con uso promedio de CPU/GPU/RAM y temperatura mÃ¡xima, para comparar. No mide FPS.
  - **Optimizar**: elegís los ajustes (activar o desactivar), abajo ves cuántos procesos hay por aplicar y, al tocar Aplicar, una pantalla de carga los ejecuta uno por uno. **Limpieza**: muestra el tamaÃ±o antes de borrar. **Inicio de Windows**: activar o desactivar programas de arranque.
- **Acciones Ãºnicas** (TRIM de SSD, vaciar DNS): no tienen interruptor porque no se pueden deshacer; llevan botÃ³n **Aplicar** con confirmaciÃ³n y registran la Ãºltima ejecuciÃ³n.
- Interfaz: tarjetas por categorÃ­a con icono propio e insignias NVIDIA / AMD / Intel en los ajustes exclusivos de cada fabricante (insignias propias, no son los logos oficiales).
- Datos en vivo sin drivers de terceros: contadores de Windows, `nvidia-smi` para la GPU NVIDIA. La temperatura de la CPU no se muestra porque Windows no la expone sin un driver de hardware.
- BotÃ³n **Revertir todo** y **modo simulaciÃ³n** (no modifica nada).
- No cierra procesos ni promete FPS: cada ajuste indica su riesgo y si necesita administrador o reinicio.
- Los datos de backup y el registro de cambios quedan en `%LOCALAPPDATA%\SouthTweaks\` (si existÃ­a la carpeta anterior `FPSBoosterPro`, se migra sola).

## Descargar

El ejecutable ya compilado estÃ¡ en [dist/South-Tweaks.exe](dist/South-Tweaks.exe). Doble clic y listo.

## Compilar

Requisitos: Go 1.23+, Node.js y [Wails v2](https://wails.io).

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build            # genera build\bin\South-Tweaks.exe
go test ./...          # prueba aplicar/revertir contra HKCU real y la simulaciÃ³n
```

## Estructura

| Archivo | Rol |
|---|---|
| `tweaks.go` | CatÃ¡logo de ajustes (registro / powercfg) |
| `engine.go` | Aplicar, verificar, backup, rollback y revertir |
| `regops.go` | Lectura/escritura de registro y powercfg (solo HKCU/HKLM, GUID validado) |
| `telemetry.go`, `games.go` | Datos en vivo (PDH, nvidia-smi) y detector de juegos |
| `profiles.go` | Perfiles de un clic y punto de restauraciÃ³n |
| `cleanup.go` | Limpieza de temporales (lista fija, sin seguir enlaces, respeta archivos recientes) |
| `startup.go` | Programas de inicio (StartupApproved, reversible) |
| `tasks.go` | Tareas programadas (consulta y cambio con PowerShell, rutas validadas) |
| `tweaks_extra.go`, `tweaks_services.go` | MÃ¡s ajustes: privacidad, servicios, loggers |
| `sysinfo.go` | Hardware y uso en vivo (sin PowerShell) |
| `frontend/` | Interfaz |
| `tools/snapshot.ps1` | Compara el sistema antes/despuÃ©s de aplicar un ajuste, para descubrir quÃ© cambia |

> Los ajustes que usan HKLM requieren ejecutar como administrador (la app ofrece reiniciarse con UAC).


