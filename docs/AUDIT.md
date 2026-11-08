# Guía de auditoría (para Codex u otro revisor)

## Cómo correr todo

```powershell
py -3.11 -m venv .venv          # o cualquier Python >= 3.11
.\.venv\Scripts\python -m pip install -r requirements-dev.txt
.\.venv\Scripts\python -m pytest -q          # todas las pruebas
.\.venv\Scripts\python -m pytest -q -m "not windows and not ui"   # sin Windows ni Qt
.\.venv\Scripts\python -m ruff check src tests
```

Las pruebas **no modifican Windows**: `tests/conftest.py` reemplaza `subprocess.run` por una versión
que falla ante `/setactive`, `reg add`, `taskkill`, `Stop-Process`, etc., y todos los datos se escriben
en una carpeta temporal (`FPS_BOOSTER_DATA_DIR`).

## Invariantes que deben mantenerse

| # | Invariante | Dónde se verifica |
|---|---|---|
| 1 | Ningún código cierra, suspende o termina procesos | `test_safety_audit.py::test_no_process_termination_anywhere` |
| 2 | Sin `shell=True`, `os.system`, `os.popen` | `test_no_shell_true_or_os_system` |
| 3 | Escritura en registro solo en `system/registry.py` y solo en HKCU | `test_registry_writes_only_in_whitelisted_module`, `test_simulated_rejects_hklm_and_unknown_scheme` |
| 4 | `WindowsSystemBackend` solo se instancia en `context.py` | `test_real_backend_only_instantiated_in_context` |
| 5 | El modo por defecto es simulación | `test_default_context_is_simulation` |
| 6 | La simulación no puede envolver un backend real | `test_simulated_refuses_to_wrap_real_backend` |
| 7 | Sin `Consent` no se aplica ni revierte nada | `test_apply_requires_consent`, `test_revert_requires_consent` |
| 8 | El original se guarda una sola vez y revertir lo restaura | `test_original_is_kept_across_profiles`, `test_apply_and_revert_roundtrip` |
| 9 | Fallo a mitad de lote → rollback completo | `test_failure_rolls_back_whole_batch`, `test_verification_failure_triggers_rollback` |
| 10 | Ninguna recomendación promete FPS | `test_no_fps_promises` |
| 11 | El diálogo exige confirmación explícita | `test_consent_dialog_requires_ack` |
| 12 | Todas las optimizaciones son reversibles y sin admin | `test_every_optimization_is_reversible_and_low_risk` |

## Puntos a revisar con atención

- `optimizations/engine.py`: orden *guardar original → aplicar → verificar → registrar* y rollback.
- `system/backend.py`: validación del GUID antes de llamar a `powercfg` (evita inyección de argumentos).
- `monitoring/gpu.py` (`PdhGpuProvider`): estructuras `ctypes` y manejo de `PDH_MORE_DATA`.
- `hardware/detector.py`: el script PowerShell es solo lectura (`Get-*`).
- Hilos: `ui/workers.py` crea sampler/analyzer dentro del hilo del worker.

## Agregar una optimización nueva

1. Crear la instancia en `optimizations/catalog.py` (subclase de `Optimization` o `RegistryOptimization`).
2. Debe ser reversible y su estado JSON-serializable.
3. Agregarla a un perfil en `profiles/profiles.py` y, si aplica, a `recommendations/recommender.py`.
4. Agregar pruebas de aplicar + revertir con `SimulatedSystemBackend`.
