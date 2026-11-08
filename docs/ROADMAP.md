# Plan de desarrollo

## Fase 1 — MVP (implementada)

- [x] Detección de CPU, GPU(s) con VRAM real, RAM, discos físicos (SSD/HDD), volúmenes y Windows 10/11.
- [x] Monitoreo en tiempo real (CPU, núcleos, RAM, disco, red, GPU) en un hilo separado.
- [x] Procesos de alto consumo, clasificados en sistema (protegido) / segundo plano / usuario. **Sin cerrar procesos.**
- [x] Perfiles Gaming, Equilibrado y Ahorro.
- [x] Recomendaciones según el nivel de hardware (baja / media / alta), todas marcadas "impacto no medido".
- [x] Diálogo de consentimiento por cambio, con confirmación explícita.
- [x] Backup de valores originales + revertir todo.
- [x] Registro de cambios (JSONL) y log de errores.
- [x] Modo simulación por defecto. Modo real tras `--modo-real` + confirmación.
- [x] Pruebas automatizadas + guía de auditoría (`docs/AUDIT.md`).

Ajustes reversibles de la fase 1 (todos en HKCU o `powercfg`, sin permisos de administrador):

| Ajuste | Mecanismo | Revertir |
|---|---|---|
| Plan de energía | `powercfg /setactive` | Reactiva el GUID original |
| Modo Juego | `HKCU\Software\Microsoft\GameBar\AutoGameModeEnabled` | Restaura valor o lo borra si no existía |
| Grabación en segundo plano (Game DVR) | `HKCU\System\GameConfigStore\GameDVR_Enabled` + `...\GameDVR\AppCaptureEnabled` | Ídem |
| Efectos de transparencia | `HKCU\...\Themes\Personalize\EnableTransparency` | Ídem |

## Fase 2 — Medición

- Integrar **PresentMon** para FPS, 1% low y frametime por juego.
- Benchmark A/B: medir antes → aplicar perfil → medir después → mostrar diferencia real.
- Temperaturas CPU/GPU con LibreHardwareMonitor (opcional, si está instalado).
- Exportar informe de sesión.

## Fase 3 — Optimizaciones avanzadas (requieren admin, con punto de restauración)

- Crear punto de restauración del sistema antes de aplicar.
- Programación de GPU acelerada por hardware (HAGS), prioridad de GPU por juego (Configuración de gráficos).
- Prioridad de proceso del juego (temporal, se revierte al cerrar el juego).
- Cierre de procesos de segundo plano **solo con autorización por proceso**.
- Detección automática de juegos (Steam/Epic) y perfiles por juego.

## Fase 4 — Distribución

- Build con PyInstaller + instalador Inno Setup firmado.
- Auto-actualización, telemetría opt-in (desactivada por defecto).
