"""Configuración de logging: archivo rotativo + consola."""

from __future__ import annotations

import logging
import logging.handlers
from pathlib import Path

LOGGER_NAME = "fps_booster"


def setup_logging(log_dir: Path, level: int = logging.INFO) -> logging.Logger:
    logger = logging.getLogger(LOGGER_NAME)
    logger.setLevel(level)
    if any(getattr(h, "_fps_booster", False) for h in logger.handlers):
        return logger
    log_dir.mkdir(parents=True, exist_ok=True)
    fmt = logging.Formatter("%(asctime)s [%(levelname)s] %(name)s: %(message)s")
    file_handler = logging.handlers.RotatingFileHandler(
        log_dir / "app.log", maxBytes=1_000_000, backupCount=3, encoding="utf-8"
    )
    file_handler.setFormatter(fmt)
    file_handler._fps_booster = True  # type: ignore[attr-defined]
    console = logging.StreamHandler()
    console.setFormatter(fmt)
    console.setLevel(logging.WARNING)
    console._fps_booster = True  # type: ignore[attr-defined]
    logger.addHandler(file_handler)
    logger.addHandler(console)
    return logger


def get_logger(name: str) -> logging.Logger:
    return logging.getLogger(f"{LOGGER_NAME}.{name}")
