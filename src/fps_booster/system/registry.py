"""Acceso al registro de Windows. Lectura libre; escritura limitada a HKCU."""

from __future__ import annotations

import sys
from typing import Any

HIVES_READ = ("HKCU", "HKLM")
HIVES_WRITE = ("HKCU",)  # Fase 1: nunca se escribe en HKLM.


def _root(hive: str):
    import winreg

    return {"HKCU": winreg.HKEY_CURRENT_USER, "HKLM": winreg.HKEY_LOCAL_MACHINE}[hive]


def read_value(hive: str, key: str, name: str) -> Any | None:
    """Devuelve el valor o None si la clave/valor no existe (o no estamos en Windows)."""
    if sys.platform != "win32" or hive not in HIVES_READ:
        return None
    import winreg

    try:
        with winreg.OpenKey(_root(hive), key, 0, winreg.KEY_READ) as handle:
            value, _type = winreg.QueryValueEx(handle, name)
            return value
    except OSError:
        return None


def write_dword(hive: str, key: str, name: str, value: int) -> None:
    if hive not in HIVES_WRITE:
        raise PermissionError(f"Escritura no permitida en {hive}")
    if not 0 <= int(value) <= 0xFFFFFFFF:
        raise ValueError("DWORD fuera de rango")
    import winreg

    with winreg.CreateKeyEx(_root(hive), key, 0, winreg.KEY_SET_VALUE) as handle:
        winreg.SetValueEx(handle, name, 0, winreg.REG_DWORD, int(value))


def delete_value(hive: str, key: str, name: str) -> None:
    if hive not in HIVES_WRITE:
        raise PermissionError(f"Escritura no permitida en {hive}")
    import winreg

    try:
        with winreg.OpenKey(_root(hive), key, 0, winreg.KEY_SET_VALUE) as handle:
            winreg.DeleteValue(handle, name)
    except FileNotFoundError:
        pass
