# FPS BOOSTER — Arquitectura

## Principios

1. **Seguridad primero.** Ningún cambio en el sistema sin consentimiento explícito (`Consent`).
2. **Reversible.** Antes de modificar un ajuste se guarda su valor original (`BackupStore`).
   El original se guarda *una sola vez*: aplicar dos perfiles seguidos no pisa el valor real de fábrica.
3. **Honestidad.** No se prometen FPS. Toda recomendación lleva la nota "Impacto no medido".
4. **Nunca se cierran procesos.** El analizador solo *identifica*; no existe código que mate procesos
   (lo verifica `tests/test_safety_audit.py`).
5. **Modo simulación por defecto.** Las escrituras van a un sistema virtual en memoria.
   El modo real requiere `--modo-real` + confirmación en pantalla.
6. **Testeable.** Toda dependencia del sistema (psutil, PowerShell, registro, powercfg) se inyecta,
   así las pruebas corren sin tocar Windows.

## Capas

```
┌──────────────────────────────── UI (PySide6) ────────────────────────────────┐
│ MainWindow · páginas (Panel, Procesos, Perfiles, Recomendaciones, Sistema,    │
│ Historial) · widgets (LiveChart, StatCard, OptimizeButton, ConsentDialog)     │
│ workers.py → hilos Qt para monitoreo y detección (la UI nunca se bloquea)     │
└───────────────┬───────────────────────────────────────────────┬──────────────┘
                │ AppContext (inyección de dependencias)        │
┌───────────────▼────────────┐   ┌───────────────────────────────▼─────────────┐
│ Lectura (sin efectos)      │   │ Cambios (con consentimiento)                │
│ hardware/detector          │   │ optimizations/engine  ← Consent             │
│ monitoring/sampler + gpu   │   │ optimizations/catalog (plan, juego, DVR…)   │
│ processes/analyzer         │   │ profiles/ (Gaming, Equilibrado, Ahorro)     │
│ recommendations/recommender│   │ state/backup_store · state/change_log       │
└───────────────┬────────────┘   └───────────────┬─────────────────────────────┘
                │                                │
┌───────────────▼────────────────────────────────▼─────────────────────────────┐
│ system/backend.py                                                            │
│  SystemReader (solo lectura) ── WindowsSystemReader                          │
│  SystemBackend (lectura+escritura)                                           │
│     ├─ SimulatedSystemBackend  (por defecto: escribe en memoria)             │
│     └─ WindowsSystemBackend    (solo con --modo-real; escribe solo en HKCU)  │
└──────────────────────────────────────────────────────────────────────────────┘
```

En modo simulación **no se instancia ningún objeto capaz de escribir** en Windows:
el `SimulatedSystemBackend` recibe un `WindowsSystemReader`, que no tiene métodos de escritura.

## Flujo de una optimización

1. `engine.plan_profile(perfil)` lee el valor actual de cada ajuste → lista de `PlannedChange`
   (marca los no disponibles, p. ej. "Alto rendimiento" oculto en equipos con Modern Standby).
2. `ConsentDialog` muestra *actual → nuevo*, riesgo y si requiere reinicio. El usuario elige
   qué aplicar y marca "Entiendo y autorizo". Solo entonces se crea un `Consent`.
3. `engine.apply(plan, consent)`: por cada cambio → guarda original → aplica → **verifica**
   leyendo de nuevo → registra. Si algo falla, **revierte el lote completo** (rollback).
4. `engine.revert_all(consent)` restaura los originales guardados y los borra del backup.

## Datos en disco (`%LOCALAPPDATA%\FPSBooster`, o `FPS_BOOSTER_DATA_DIR`)

| Archivo | Contenido |
|---|---|
| `logs/app.log` | Log técnico rotativo (errores, avisos) |
| `logs/changes_<modo>.jsonl` | Auditoría: cada apply / revert / rollback / error |
| `backups/originals_<modo>.json` | Valores originales para revertir (escritura atómica) |

Simulación y real usan archivos separados para que una prueba nunca contamine los originales reales.

## Estructura de carpetas

```
fps-booster/
├─ pyproject.toml · requirements*.txt · README.md
├─ docs/            ARCHITECTURE.md · ROADMAP.md · AUDIT.md (guía para Codex)
├─ packaging/       fps_booster.spec (PyInstaller) · installer.iss (Inno Setup) · build.ps1
├─ src/fps_booster/
│  ├─ app.py                 punto de entrada, argumentos, AppContext
│  ├─ core/                  modelos, rutas, logging, formato
│  ├─ system/                backends (lectura / simulación / real), registro
│  ├─ hardware/              detección CPU, GPU, RAM, discos, Windows
│  ├─ monitoring/            muestreo psutil + proveedores GPU (nvidia-smi, PDH)
│  ├─ processes/             análisis de procesos (sin cerrar nada)
│  ├─ optimizations/         base, catálogo, motor transaccional
│  ├─ profiles/              Gaming / Equilibrado / Ahorro
│  ├─ recommendations/       nivel de hardware + recomendaciones
│  ├─ state/                 backups y registro de cambios
│  └─ ui/                    tema, widgets, páginas, workers, ventana principal
└─ tests/                    pytest (unitarias + auditoría de seguridad + UI offscreen)
```

## Fuentes de métricas

| Métrica | Fase 1 | Fase 2 |
|---|---|---|
| CPU / RAM / disco / red | psutil | — |
| Uso GPU | `nvidia-smi` (NVIDIA) o contadores PDH de Windows (cualquier marca) | — |
| VRAM usada | nvidia-smi / PDH "GPU Adapter Memory" | — |
| Temperatura GPU | nvidia-smi (solo NVIDIA) | LibreHardwareMonitor |
| Temperatura CPU | No disponible | LibreHardwareMonitor |
| FPS / frametime | No disponible | PresentMon (Intel, open source) |
