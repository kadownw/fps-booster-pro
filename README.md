# FPS BOOSTER

Optimizador de rendimiento para juegos en Windows 10/11. Seguro, transparente y reversible.

- Detecta CPU, GPU (VRAM real), RAM, discos SSD/HDD y versión de Windows.
- Monitoreo en tiempo real de CPU, RAM, disco, red y GPU (nvidia-smi o contadores de Windows).
- Identifica procesos de alto consumo — **nunca los cierra**.
- Perfiles Gaming / Equilibrado / Ahorro y recomendaciones según tu hardware.
- Cada cambio pide consentimiento, guarda el valor original y se puede revertir.
- **No promete FPS**: toda recomendación dice "impacto no medido" hasta tener medición (fase 2).

## Ejecutar

```powershell
py -3.11 -m venv .venv
.\.venv\Scripts\python -m pip install -r requirements-dev.txt
$env:PYTHONPATH = "src"
.\.venv\Scripts\python -m fps_booster              # MODO SIMULACIÓN (por defecto)
.\.venv\Scripts\python -m fps_booster --modo-real  # aplica cambios reales, pide confirmación
```

En **modo simulación** la app lee tu configuración real pero escribe en un sistema virtual en memoria:
podés probar todo el flujo sin tocar Windows.

## Pruebas

```powershell
.\.venv\Scripts\python -m pytest -q
.\.venv\Scripts\python -m ruff check src tests
```

## Documentación

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — capas, flujo y estructura de carpetas
- [docs/ROADMAP.md](docs/ROADMAP.md) — plan por fases
- [docs/AUDIT.md](docs/AUDIT.md) — invariantes de seguridad y guía para auditar

## Datos

`%LOCALAPPDATA%\FPSBooster\` → `logs\app.log`, `logs\changes_<modo>.jsonl`, `backups\originals_<modo>.json`.
