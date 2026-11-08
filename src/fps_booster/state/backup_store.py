"""Almacén de valores ORIGINALES para poder revertir.

Regla clave: el original de un ajuste se guarda solo la primera vez. Si el usuario aplica
"Gaming" y luego "Ahorro", revertir vuelve al valor que tenía antes de usar FPS BOOSTER.
"""

from __future__ import annotations

import json
import os
import threading
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from fps_booster.core.logging_setup import get_logger

log = get_logger("backup")


def _now() -> str:
    return datetime.now(UTC).isoformat(timespec="seconds")


class BackupStore:
    VERSION = 1

    def __init__(self, path: Path):
        self.path = Path(path)
        self._lock = threading.Lock()
        self._entries: dict[str, dict[str, Any]] = {}
        self._load()

    def _load(self) -> None:
        if not self.path.exists():
            return
        try:
            data = json.loads(self.path.read_text(encoding="utf-8"))
            entries = data.get("originals", {})
            if not isinstance(entries, dict):
                raise ValueError("formato inválido")
            self._entries = entries
        except (OSError, ValueError) as exc:
            stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
            corrupt = self.path.with_suffix(f".corrupt-{stamp}.json")
            log.error("Backup ilegible (%s). Se conserva como %s", exc, corrupt.name)
            try:
                self.path.replace(corrupt)
            except OSError:
                pass
            self._entries = {}

    def _save(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        tmp = self.path.with_suffix(".tmp")
        payload = {"version": self.VERSION, "updated_at": _now(), "originals": self._entries}
        tmp.write_text(json.dumps(payload, indent=2, ensure_ascii=False), encoding="utf-8")
        os.replace(tmp, self.path)  # atómico: nunca queda un archivo a medio escribir

    def remember_original(self, optimization_id: str, value: Any) -> bool:
        """Guarda el valor si no había uno. Devuelve True si se guardó ahora."""
        with self._lock:
            if optimization_id in self._entries:
                return False
            self._entries[optimization_id] = {"value": value, "captured_at": _now()}
            self._save()
            return True

    def has(self, optimization_id: str) -> bool:
        return optimization_id in self._entries

    def get(self, optimization_id: str) -> Any:
        return self._entries[optimization_id]["value"]

    def all(self) -> dict[str, Any]:
        return {k: v["value"] for k, v in self._entries.items()}

    def entries(self) -> dict[str, dict[str, Any]]:
        return {k: dict(v) for k, v in self._entries.items()}

    def forget(self, optimization_id: str) -> None:
        with self._lock:
            if self._entries.pop(optimization_id, None) is not None:
                self._save()

    def clear(self) -> None:
        with self._lock:
            self._entries = {}
            self._save()
