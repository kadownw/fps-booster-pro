"""Rutas de datos de la aplicación."""

from __future__ import annotations

import os
from pathlib import Path

APP_DIR_NAME = "FPSBooster"


def data_dir(base: Path | None = None) -> Path:
    """Directorio de datos. Orden: argumento → FPS_BOOSTER_DATA_DIR → %LOCALAPPDATA%\\FPSBooster."""
    if base is None:
        env = os.environ.get("FPS_BOOSTER_DATA_DIR")
        if env:
            base = Path(env)
        else:
            local = os.environ.get("LOCALAPPDATA") or str(Path.home() / "AppData" / "Local")
            base = Path(local) / APP_DIR_NAME
    base.mkdir(parents=True, exist_ok=True)
    (base / "logs").mkdir(exist_ok=True)
    (base / "backups").mkdir(exist_ok=True)
    return base
