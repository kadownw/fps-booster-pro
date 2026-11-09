# FPS Booster Pro

Ajustes **reversibles** de Windows para juegos, en un único `.exe` (Go + Wails). Sin Python ni runtimes: abrís y listo (usa WebView2, que ya viene con Windows 10/11).

- Detecta CPU, GPU (con VRAM), RAM y Windows, y muestra el uso en vivo.
- 66 ajustes en 10 categorías (Energía, Juegos, Entrada, Visual, Sistema, GPU, Red, Privacidad, Servicios y Acciones únicas): registro, planes de energía, servicios, tareas programadas, loggers de diagnóstico y perfil del driver NVIDIA.
- **Plan de energía propio "Rendimiento estable"**: la CPU sube de frecuencia al instante y baja de a poco, boost agresivo, sin ahorro en USB/PCIe/Wi-Fi, conservando los estados de reposo; respeta el estacionamiento de núcleos en CPUs híbridas y Ryzen X3D de dos CCD. Al revertir vuelve el plan anterior y se borra.
- **Perfil NVIDIA** vía NvAPI oficial: energía "Preferir rendimiento máximo", baja latencia (1 frame en cola) y caché de shaders activada, con restauración de cada valor previo.
- Investigación técnica, matriz de optimizaciones y plan por fases en [`docs/research/`](docs/research/).
- Antes de tocar nada guarda el valor original; verifica cada cambio y hace rollback si falla.
- Páginas:
  - **Inicio**: panel de salud con medidores de marcas que cambian de verde a amarillo y rojo según la carga o la temperatura (CPU, GPU, RAM, almacenamiento), atajos y perfiles de un clic (Gaming, Rápido sin admin, Privacidad) con punto de restauración opcional.
  - **Mi PC**: gráfico en tiempo real (1 o 5 minutos) y detalle de procesador, GPU, memoria y discos.
  - **Mis juegos**: detecta juegos (Steam, Epic, Riot, Xbox, GOG, Ubisoft, EA, Rockstar, Roblox) y guarda cada sesión con uso promedio de CPU/GPU/RAM y temperatura máxima, para comparar. No mide FPS.
  - **Optimizar**: los 66 ajustes. **Limpieza**: muestra el tamaño antes de borrar. **Inicio de Windows**: activar o desactivar programas de arranque.
- **Acciones únicas** (TRIM de SSD, vaciar DNS): no tienen interruptor porque no se pueden deshacer; llevan botón **Aplicar** con confirmación y registran la última ejecución.
- Interfaz: tarjetas por categoría con icono propio e insignias NVIDIA / AMD / Intel en los ajustes exclusivos de cada fabricante (insignias propias, no son los logos oficiales).
- Datos en vivo sin drivers de terceros: contadores de Windows, `nvidia-smi` para la GPU NVIDIA. La temperatura de la CPU no se muestra porque Windows no la expone sin un driver de hardware.
- Botón **Revertir todo** y **modo simulación** (no modifica nada).
- No cierra procesos ni promete FPS: cada ajuste indica su riesgo y si necesita administrador o reinicio.
- Los datos de backup y el registro de cambios quedan en `%LOCALAPPDATA%\FPSBoosterPro\`.

## Descargar

El ejecutable ya compilado está en [dist/FPS-Booster-Pro.exe](dist/FPS-Booster-Pro.exe). Doble clic y listo.

## Compilar

Requisitos: Go 1.23+, Node.js y [Wails v2](https://wails.io).

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build            # genera build\bin\fbp-go.exe
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
