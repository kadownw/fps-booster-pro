"""Proveedores de métricas de GPU.

Orden de preferencia:
1. ``nvidia-smi`` (herramienta oficial de NVIDIA): uso, VRAM y temperatura.
2. Contadores PDH de Windows ("GPU Engine"): uso y VRAM para cualquier marca (sin temperatura).
3. ``NullGpuProvider``: sin datos → la UI muestra "N/D".
"""

from __future__ import annotations

import ctypes
import os
import re
import shutil
import subprocess
import sys
from abc import ABC, abstractmethod
from collections import defaultdict
from pathlib import Path

from fps_booster.core.logging_setup import get_logger
from fps_booster.core.models import GpuSample

log = get_logger("gpu")
CREATE_NO_WINDOW = 0x08000000


class GpuMetricsProvider(ABC):
    name: str = "base"

    @abstractmethod
    def sample(self) -> GpuSample | None: ...

    def close(self) -> None:  # noqa: B027 - opcional
        pass


class NullGpuProvider(GpuMetricsProvider):
    name = "no disponible"

    def sample(self) -> GpuSample | None:
        return None


# ---------------------------------------------------------------- nvidia-smi
def find_nvidia_smi() -> str | None:
    found = shutil.which("nvidia-smi")
    if found:
        return found
    candidates = [
        Path(os.environ.get("SystemRoot", r"C:\Windows")) / "System32" / "nvidia-smi.exe",
        Path(os.environ.get("ProgramFiles", r"C:\Program Files"))
        / "NVIDIA Corporation" / "NVSMI" / "nvidia-smi.exe",
    ]
    return next((str(p) for p in candidates if p.is_file()), None)


def _num(text: str) -> float | None:
    text = text.strip()
    try:
        return float(text)
    except ValueError:
        return None  # "[N/A]", "[Not Supported]"


def parse_nvidia_smi(output: str) -> GpuSample | None:
    """Formato: utilization.gpu, memory.used, memory.total, temperature.gpu (MiB, °C)."""
    line = next((ln for ln in output.splitlines() if ln.strip()), "")
    parts = line.split(",")
    if len(parts) < 4:
        return None
    util, used, total, temp = (_num(p) for p in parts[:4])
    mib = 1024 * 1024
    return GpuSample(
        utilization=util,
        memory_used_bytes=int(used * mib) if used is not None else None,
        memory_total_bytes=int(total * mib) if total is not None else None,
        temperature_c=temp,
        source="nvidia-smi",
    )


class NvidiaSmiProvider(GpuMetricsProvider):
    name = "nvidia-smi"
    QUERY = "--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu"

    def __init__(self, exe: str, runner=subprocess.run):
        self._exe = exe
        self._run = runner

    def sample(self) -> GpuSample | None:
        kwargs = {"capture_output": True, "text": True, "timeout": 4, "check": False}
        if sys.platform == "win32":
            kwargs["creationflags"] = CREATE_NO_WINDOW
        try:
            proc = self._run([self._exe, self.QUERY, "--format=csv,noheader,nounits"], **kwargs)
        except (OSError, subprocess.SubprocessError) as exc:
            log.debug("nvidia-smi falló: %s", exc)
            return None
        return parse_nvidia_smi(proc.stdout or "") if proc.returncode == 0 else None


# ---------------------------------------------------------------- PDH (Windows)
_ENGINE_RE = re.compile(r"luid_(0x[0-9a-f]+_0x[0-9a-f]+)_phys_(\d+)_eng_(\d+)", re.IGNORECASE)


def aggregate_engine_utilization(items: list[tuple[str, float]]) -> float | None:
    """Igual que el Administrador de tareas: suma por motor (todas las apps) y toma el máximo."""
    per_engine: dict[tuple[str, str, str], float] = defaultdict(float)
    for name, value in items:
        m = _ENGINE_RE.search(name or "")
        if m:
            per_engine[(m.group(1).lower(), m.group(2), m.group(3))] += max(0.0, value)
    if not per_engine:
        return None
    return round(min(100.0, max(per_engine.values())), 1)


class PdhGpuProvider(GpuMetricsProvider):
    name = "contadores de Windows (PDH)"
    PDH_FMT_DOUBLE = 0x00000200
    PDH_MORE_DATA = 0x800007D2
    VALID = (0, 1)  # PDH_CSTATUS_VALID_DATA, PDH_CSTATUS_NEW_DATA

    def __init__(self):
        if sys.platform != "win32":
            raise OSError("PDH solo existe en Windows")
        from ctypes import wintypes

        class _Value(ctypes.Structure):
            _fields_ = [("CStatus", wintypes.DWORD), ("doubleValue", ctypes.c_double)]

        class _Item(ctypes.Structure):
            _fields_ = [("szName", ctypes.c_wchar_p), ("FmtValue", _Value)]

        self._Item = _Item
        self._wt = wintypes
        pdh = ctypes.WinDLL("pdh.dll")
        pdh.PdhOpenQueryW.argtypes = [wintypes.LPCWSTR, ctypes.c_size_t, ctypes.POINTER(wintypes.HANDLE)]
        pdh.PdhOpenQueryW.restype = wintypes.DWORD
        pdh.PdhAddEnglishCounterW.argtypes = [
            wintypes.HANDLE, wintypes.LPCWSTR, ctypes.c_size_t, ctypes.POINTER(wintypes.HANDLE)
        ]
        pdh.PdhAddEnglishCounterW.restype = wintypes.DWORD
        pdh.PdhCollectQueryData.argtypes = [wintypes.HANDLE]
        pdh.PdhCollectQueryData.restype = wintypes.DWORD
        pdh.PdhGetFormattedCounterArrayW.argtypes = [
            wintypes.HANDLE, wintypes.DWORD, ctypes.POINTER(wintypes.DWORD),
            ctypes.POINTER(wintypes.DWORD), ctypes.c_void_p,
        ]
        pdh.PdhGetFormattedCounterArrayW.restype = wintypes.DWORD
        pdh.PdhCloseQuery.argtypes = [wintypes.HANDLE]
        pdh.PdhCloseQuery.restype = wintypes.DWORD
        self._pdh = pdh
        self._query = wintypes.HANDLE()
        status = pdh.PdhOpenQueryW(None, 0, ctypes.byref(self._query))
        if status != 0:
            raise OSError(f"PdhOpenQuery: 0x{status:08x}")
        self._util = self._add(r"\GPU Engine(*engtype_3D)\Utilization Percentage")
        self._mem = self._add(r"\GPU Adapter Memory(*)\Dedicated Usage")
        if self._util is None:
            self.close()
            raise OSError("Contador 'GPU Engine' no disponible")
        pdh.PdhCollectQueryData(self._query)  # primera lectura: base para los contadores de tasa

    def _add(self, path: str):
        handle = self._wt.HANDLE()
        status = self._pdh.PdhAddEnglishCounterW(self._query, path, 0, ctypes.byref(handle))
        return handle if status == 0 else None

    def _items(self, counter) -> list[tuple[str, float]]:
        size, count = self._wt.DWORD(0), self._wt.DWORD(0)
        fmt = self.PDH_FMT_DOUBLE
        status = self._pdh.PdhGetFormattedCounterArrayW(
            counter, fmt, ctypes.byref(size), ctypes.byref(count), None
        )
        if status != self.PDH_MORE_DATA or size.value == 0:
            return []
        buf = (ctypes.c_byte * size.value)()
        status = self._pdh.PdhGetFormattedCounterArrayW(
            counter, fmt, ctypes.byref(size), ctypes.byref(count), buf
        )
        if status != 0:
            return []
        arr = ctypes.cast(buf, ctypes.POINTER(self._Item))
        return [
            (arr[i].szName or "", arr[i].FmtValue.doubleValue)
            for i in range(count.value)
            if arr[i].FmtValue.CStatus in self.VALID
        ]

    def sample(self) -> GpuSample | None:
        if self._pdh.PdhCollectQueryData(self._query) != 0:
            return None
        util = aggregate_engine_utilization(self._items(self._util))
        mem = None
        if self._mem is not None:
            values = [v for _, v in self._items(self._mem)]
            mem = int(max(values)) if values else None
        return GpuSample(util, mem, None, None, source="PDH")

    def close(self) -> None:
        if getattr(self, "_query", None):
            self._pdh.PdhCloseQuery(self._query)
            self._query = None


def create_default_provider() -> GpuMetricsProvider:
    exe = find_nvidia_smi()
    if exe:
        provider = NvidiaSmiProvider(exe)
        if provider.sample() is not None:
            log.info("Métricas GPU: nvidia-smi")
            return provider
    try:
        provider = PdhGpuProvider()
        log.info("Métricas GPU: PDH")
        return provider
    except OSError as exc:
        log.info("Sin métricas de GPU: %s", exc)
    return NullGpuProvider()
