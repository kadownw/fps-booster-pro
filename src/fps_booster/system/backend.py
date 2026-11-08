"""Backends del sistema.

- ``SystemReader``: solo lectura (plan de energía activo, registro).
- ``SystemBackend``: lectura + escritura.
- ``SimulatedSystemBackend``: escribe en memoria. Recibe un *reader*, nunca un backend con escritura,
  así en modo simulación no existe ningún objeto capaz de modificar Windows.
- ``WindowsSystemBackend``: el real. Solo se crea con ``--modo-real``.
"""

from __future__ import annotations

import re
import subprocess
import sys
from abc import ABC, abstractmethod
from typing import Any

from fps_booster.core.logging_setup import get_logger
from fps_booster.system import registry

log = get_logger("backend")

CREATE_NO_WINDOW = 0x08000000

POWER_SCHEMES = {
    "high_performance": "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c",
    "balanced": "381b4222-f694-41f0-9685-ff5bb260df2e",
    "power_saver": "a1841308-3541-4fab-bc81-f71556f20b4a",
}
POWER_SCHEME_NAMES = {
    POWER_SCHEMES["high_performance"]: "Alto rendimiento",
    POWER_SCHEMES["balanced"]: "Equilibrado",
    POWER_SCHEMES["power_saver"]: "Economizador",
}

_GUID = r"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
GUID_RE = re.compile(rf"^{_GUID}$")
# Greedy hasta el último ")" de la línea: admite nombres con paréntesis, p. ej. "Plan (ExitLag)".
_SCHEME_LINE_RE = re.compile(rf"({_GUID})\s*\((.*)\)")


def parse_powercfg_list(output: str) -> dict[str, str]:
    """Parsea `powercfg /list`. Independiente del idioma: se basa en GUID + (nombre)."""
    return {m.group(1).lower(): m.group(2).strip() for m in _SCHEME_LINE_RE.finditer(output)}


def parse_active_scheme(output: str) -> str | None:
    m = re.search(_GUID, output)
    return m.group(0).lower() if m else None


def _norm_key(hive: str, key: str, name: str) -> tuple[str, str, str]:
    return hive.upper(), key.lower().strip("\\"), name.lower()


class SystemReader(ABC):
    @abstractmethod
    def get_active_power_scheme(self) -> str | None: ...

    @abstractmethod
    def list_power_schemes(self) -> dict[str, str]: ...

    @abstractmethod
    def read_registry_value(self, hive: str, key: str, name: str) -> Any | None: ...


class SystemBackend(SystemReader):
    is_simulation: bool = False

    @abstractmethod
    def set_active_power_scheme(self, guid: str) -> None: ...

    @abstractmethod
    def write_registry_dword(self, hive: str, key: str, name: str, value: int) -> None: ...

    @abstractmethod
    def delete_registry_value(self, hive: str, key: str, name: str) -> None: ...


class WindowsSystemReader(SystemReader):
    """Lectura real de Windows. No tiene ningún método de escritura."""

    def __init__(self, runner=subprocess.run):
        self._run = runner

    def _powercfg(self, *args: str) -> str:
        kwargs: dict[str, Any] = {
            "capture_output": True,
            "timeout": 15,
            "check": True,
            "encoding": "oem" if sys.platform == "win32" else "utf-8",
            "errors": "replace",
        }
        if sys.platform == "win32":
            kwargs["creationflags"] = CREATE_NO_WINDOW
        return self._run(["powercfg", *args], **kwargs).stdout or ""

    def get_active_power_scheme(self) -> str | None:
        try:
            return parse_active_scheme(self._powercfg("/getactivescheme"))
        except (OSError, subprocess.SubprocessError) as exc:
            log.warning("No se pudo leer el plan de energía activo: %s", exc)
            return None

    def list_power_schemes(self) -> dict[str, str]:
        try:
            return parse_powercfg_list(self._powercfg("/list"))
        except (OSError, subprocess.SubprocessError) as exc:
            log.warning("No se pudo listar planes de energía: %s", exc)
            return {}

    def read_registry_value(self, hive: str, key: str, name: str) -> Any | None:
        return registry.read_value(hive, key, name)


class WindowsSystemBackend(WindowsSystemReader, SystemBackend):
    """Backend REAL. Modifica Windows. Solo HKCU y powercfg /setactive."""

    is_simulation = False

    def set_active_power_scheme(self, guid: str) -> None:
        guid = guid.lower()
        if not GUID_RE.match(guid):
            raise ValueError(f"GUID inválido: {guid!r}")
        if guid not in self.list_power_schemes():
            raise LookupError(f"El plan {guid} no existe en este equipo")
        self._powercfg("/setactive", guid)

    def write_registry_dword(self, hive: str, key: str, name: str, value: int) -> None:
        registry.write_dword(hive, key, name, value)

    def delete_registry_value(self, hive: str, key: str, name: str) -> None:
        registry.delete_value(hive, key, name)


_DELETED = object()


class SimulatedSystemBackend(SystemBackend):
    """Sistema virtual: lee del reader real (opcional) y escribe en una capa en memoria."""

    is_simulation = True

    def __init__(
        self,
        reader: SystemReader | None = None,
        *,
        active_scheme: str = POWER_SCHEMES["balanced"],
        schemes: dict[str, str] | None = None,
        registry_values: dict[tuple[str, str, str], Any] | None = None,
    ):
        if isinstance(reader, SystemBackend) and not reader.is_simulation:
            raise TypeError("La simulación no puede envolver un backend con escritura real")
        self._reader = reader
        self._default_active = active_scheme
        self._default_schemes = schemes or {g: n for g, n in POWER_SCHEME_NAMES.items()}
        self._active: str | None = None
        self._registry: dict[tuple[str, str, str], Any] = {
            _norm_key(*k): v for k, v in (registry_values or {}).items()
        }
        self.operations: list[str] = []

    def get_active_power_scheme(self) -> str | None:
        if self._active is not None:
            return self._active
        if self._reader is not None:
            value = self._reader.get_active_power_scheme()
            if value:
                return value
        return self._default_active

    def list_power_schemes(self) -> dict[str, str]:
        if self._reader is not None:
            schemes = self._reader.list_power_schemes()
            if schemes:
                return schemes
        return dict(self._default_schemes)

    def set_active_power_scheme(self, guid: str) -> None:
        guid = guid.lower()
        if not GUID_RE.match(guid):
            raise ValueError(f"GUID inválido: {guid!r}")
        if guid not in self.list_power_schemes():
            raise LookupError(f"El plan {guid} no existe en este equipo")
        self._active = guid
        self.operations.append(f"powercfg /setactive {guid}")

    def read_registry_value(self, hive: str, key: str, name: str) -> Any | None:
        k = _norm_key(hive, key, name)
        if k in self._registry:
            value = self._registry[k]
            return None if value is _DELETED else value
        if self._reader is not None:
            return self._reader.read_registry_value(hive, key, name)
        return None

    def write_registry_dword(self, hive: str, key: str, name: str, value: int) -> None:
        if hive.upper() not in registry.HIVES_WRITE:
            raise PermissionError(f"Escritura no permitida en {hive}")
        self._registry[_norm_key(hive, key, name)] = int(value)
        self.operations.append(f"reg add {hive}\\{key} /v {name} /t REG_DWORD /d {int(value)}")

    def delete_registry_value(self, hive: str, key: str, name: str) -> None:
        if hive.upper() not in registry.HIVES_WRITE:
            raise PermissionError(f"Escritura no permitida en {hive}")
        self._registry[_norm_key(hive, key, name)] = _DELETED
        self.operations.append(f"reg delete {hive}\\{key} /v {name}")
