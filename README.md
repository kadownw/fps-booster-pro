# FPS Booster Pro

Ajustes **reversibles** de Windows para juegos, en un único `.exe` (Go + Wails). Sin Python ni runtimes: abrís y listo (usa WebView2, que ya viene con Windows 10/11).

- Detecta CPU, GPU (con VRAM), RAM y Windows, y muestra el uso en vivo.
- 52 ajustes en 9 categorías (Energía, Juegos, Entrada, Visual, Sistema, GPU, Red, Privacidad y Servicios): registro, plan de energía, servicios, tareas programadas y loggers de diagnóstico.
- Antes de tocar nada guarda el valor original; verifica cada cambio y hace rollback si falla.
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
| `tasks.go` | Tareas programadas (consulta y cambio con PowerShell, rutas validadas) |
| `tweaks_extra.go`, `tweaks_services.go` | Más ajustes: privacidad, servicios, loggers |
| `sysinfo.go` | Hardware y uso en vivo (sin PowerShell) |
| `frontend/` | Interfaz |
| `tools/snapshot.ps1` | Compara el sistema antes/después de aplicar un ajuste, para descubrir qué cambia |

> Los ajustes que usan HKLM requieren ejecutar como administrador (la app ofrece reiniciarse con UAC).
