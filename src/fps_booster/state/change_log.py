"""Registro de auditoría de cambios (JSON Lines, solo se agrega)."""

from __future__ import annotations

import json
import threading
from datetime import datetime
from pathlib import Path
from typing import Any

from fps_booster.core.logging_setup import get_logger

log = get_logger("changes")

ACTIONS = ("apply", "revert", "rollback", "error", "info")


class ChangeLog:
    def __init__(self, path: Path, mode: str):
        self.path = Path(path)
        self.mode = mode
        self._lock = threading.Lock()

    def record(
        self,
        action: str,
        optimization_id: str | None = None,
        *,
        success: bool = True,
        before: Any = None,
        after: Any = None,
        message: str = "",
    ) -> dict[str, Any]:
        if action not in ACTIONS:
            raise ValueError(f"Acción desconocida: {action}")
        entry = {
            "timestamp": datetime.now().astimezone().isoformat(timespec="seconds"),
            "mode": self.mode,
            "action": action,
            "optimization_id": optimization_id,
            "success": success,
            "before": before,
            "after": after,
            "message": message,
        }
        with self._lock:
            self.path.parent.mkdir(parents=True, exist_ok=True)
            with self.path.open("a", encoding="utf-8") as fh:
                fh.write(json.dumps(entry, ensure_ascii=False, default=str) + "\n")
        level = 20 if success else 40
        log.log(level, "[%s] %s %s %s", self.mode, action, optimization_id or "", message)
        return entry

    def entries(self, limit: int = 500) -> list[dict[str, Any]]:
        """Entradas más recientes primero. Ignora líneas dañadas."""
        if not self.path.exists():
            return []
        out: list[dict[str, Any]] = []
        with self.path.open(encoding="utf-8", errors="replace") as fh:
            for line in fh:
                line = line.strip()
                if not line:
                    continue
                try:
                    out.append(json.loads(line))
                except json.JSONDecodeError:
                    continue
        return list(reversed(out))[:limit]
