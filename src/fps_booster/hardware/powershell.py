"""Ejecución de consultas PowerShell de SOLO LECTURA que devuelven JSON."""

from __future__ import annotations

import json
import subprocess
import sys

from fps_booster.core.logging_setup import get_logger

log = get_logger("powershell")
CREATE_NO_WINDOW = 0x08000000


def run_powershell_json(script: str, timeout: float = 25.0) -> object | None:
    if sys.platform != "win32":
        return None
    prefix = "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; "
    cmd = ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", prefix + script]
    try:
        proc = subprocess.run(
            cmd,
            capture_output=True,
            timeout=timeout,
            creationflags=CREATE_NO_WINDOW,
            check=False,
        )
    except (OSError, subprocess.SubprocessError) as exc:
        log.warning("PowerShell no disponible: %s", exc)
        return None
    out = proc.stdout.decode("utf-8", errors="replace").strip()
    if proc.returncode != 0 or not out:
        log.warning("Consulta PowerShell falló (%s): %s", proc.returncode, proc.stderr[:300])
        return None
    try:
        return json.loads(out)
    except json.JSONDecodeError as exc:
        log.warning("Salida JSON inválida de PowerShell: %s", exc)
        return None
