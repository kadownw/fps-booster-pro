"""Detección de hardware y versión de Windows (solo lectura)."""

from __future__ import annotations

import os
import platform
import re
from collections.abc import Callable
from typing import Any

import psutil

from fps_booster.core.logging_setup import get_logger
from fps_booster.core.models import (
    CpuInfo,
    GpuInfo,
    HardwareInfo,
    MemoryInfo,
    PhysicalDisk,
    Volume,
    WindowsInfo,
)
from fps_booster.hardware.powershell import run_powershell_json
from fps_booster.system.registry import read_value

log = get_logger("hardware")

# Una sola consulta (cada arranque de PowerShell cuesta ~1 s).
HARDWARE_SCRIPT = r"""
$gpus = @(Get-CimInstance Win32_VideoController | Select-Object Name, AdapterRAM, DriverVersion)
$cls = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\0*'
$reg = @(Get-ItemProperty $cls -ErrorAction SilentlyContinue |
  Select-Object DriverDesc, @{n='Memory';e={$_.'HardwareInformation.qwMemorySize'}})
$disks = @(Get-PhysicalDisk -ErrorAction SilentlyContinue |
  Select-Object FriendlyName, Size,
    @{n='MediaType';e={[string]$_.MediaType}},
    @{n='BusType';e={[string]$_.BusType}})
$sys = $null
try { $sys = (Get-Partition -DriveLetter $env:SystemDrive[0] -ErrorAction Stop | Get-Disk).FriendlyName } catch {}
[pscustomobject]@{ gpus = $gpus; gpu_registry = $reg; disks = $disks; system_disk = $sys } |
  ConvertTo-Json -Depth 4 -Compress
"""

CPU_KEY = r"HARDWARE\DESCRIPTION\System\CentralProcessor\0"
WIN_KEY = r"SOFTWARE\Microsoft\Windows NT\CurrentVersion"


def _as_list(value: Any) -> list:
    if value is None:
        return []
    return value if isinstance(value, list) else [value]


def _to_int(value: Any) -> int | None:
    if value is None:
        return None
    if isinstance(value, list):  # REG_BINARY serializado como lista de bytes
        try:
            return int.from_bytes(bytes(int(b) & 0xFF for b in value), "little") or None
        except (TypeError, ValueError):
            return None
    try:
        number = int(value)
    except (TypeError, ValueError):
        return None
    return number if number > 0 else None


def vendor_from_name(name: str) -> str:
    n = name.lower()
    if "nvidia" in n or "geforce" in n or "quadro" in n or "rtx" in n:
        return "nvidia"
    if "amd" in n or "radeon" in n or "ati " in n:
        return "amd"
    if "intel" in n:
        return "intel"
    if "microsoft" in n or "basic" in n or "remote" in n:
        return "microsoft"
    return "desconocido"


def is_integrated_gpu(name: str, vendor: str) -> bool:
    n = name.lower()
    if vendor == "nvidia":
        return False
    if vendor == "intel":
        # Dedicadas: "Arc(TM) A770", "Arc B580". "Intel(R) Arc(TM) Graphics" (Core Ultra) es integrada.
        return re.search(r"arc(\(tm\))?\s+[ab]\d{3}", n) is None
    if vendor == "amd":
        if re.search(r"\brx\s*\d", n) or "pro w" in n or "radeon pro" in n:
            return False
        # "AMD Radeon(TM) Graphics", "Radeon Vega 8 Graphics", "Radeon 780M" → integradas.
        return "graphics" in n or "vega" in n or re.search(r"\d{3}m\b", n) is not None
    return vendor == "microsoft"


def parse_gpus(raw_gpus: Any, raw_registry: Any) -> tuple[GpuInfo, ...]:
    reg_memory: dict[str, int] = {}
    for item in _as_list(raw_registry):
        if isinstance(item, dict) and item.get("DriverDesc"):
            mem = _to_int(item.get("Memory"))
            if mem:
                reg_memory[str(item["DriverDesc"]).strip().lower()] = mem
    gpus: list[GpuInfo] = []
    for item in _as_list(raw_gpus):
        if not isinstance(item, dict) or not item.get("Name"):
            continue
        name = str(item["Name"]).strip()
        vendor = vendor_from_name(name)
        # AdapterRAM (WMI) es uint32 y satura en 4 GB: preferimos el valor del registro.
        vram = reg_memory.get(name.lower()) or _to_int(item.get("AdapterRAM"))
        gpus.append(
            GpuInfo(
                name=name,
                vendor=vendor,
                vram_bytes=vram,
                driver_version=item.get("DriverVersion"),
                integrated=is_integrated_gpu(name, vendor),
            )
        )
    return tuple(gpus)


def normalize_media_type(value: Any) -> str:
    v = str(value or "").strip().upper()
    if v in ("SSD", "4", "SCM", "5"):
        return "SSD"
    if v in ("HDD", "3"):
        return "HDD"
    return "Desconocido"


def parse_disks(raw_disks: Any, system_disk_name: str | None) -> tuple[PhysicalDisk, ...]:
    disks: list[PhysicalDisk] = []
    sys_name = (system_disk_name or "").strip().lower()
    for item in _as_list(raw_disks):
        if not isinstance(item, dict):
            continue
        name = str(item.get("FriendlyName") or "Disco").strip()
        bus = item.get("BusType") or None
        media = normalize_media_type(item.get("MediaType"))
        if media == "Desconocido" and str(bus).upper() == "NVME":
            media = "SSD"
        disks.append(
            PhysicalDisk(
                name=name,
                media_type=media,
                bus_type=str(bus) if bus else None,
                size_bytes=_to_int(item.get("Size")),
                is_system=bool(sys_name) and name.lower() == sys_name,
            )
        )
    return tuple(disks)


def build_windows_info(
    build: Any, display_version: Any, edition: Any, ubr: Any, fallback_version: str = ""
) -> WindowsInfo:
    build_num = _to_int(build)
    if build_num is None and fallback_version:
        parts = fallback_version.split(".")
        build_num = _to_int(parts[2]) if len(parts) >= 3 else None
    if build_num is None:
        name = "Windows (desconocido)"
    elif build_num >= 22000:
        name = "Windows 11"  # ProductName sigue diciendo "Windows 10" en Win11: usamos el build.
    elif build_num >= 10240:
        name = "Windows 10"
    else:
        name = "Windows (no compatible)"
    return WindowsInfo(
        name=name,
        edition=str(edition) if edition else None,
        display_version=str(display_version) if display_version else None,
        build=build_num,
        ubr=int(ubr) if isinstance(ubr, int) and ubr >= 0 else _to_int(ubr),
    )


class HardwareDetector:
    def __init__(
        self,
        ps_runner: Callable[[str], object | None] = run_powershell_json,
        reg_reader: Callable[[str, str, str], Any] = read_value,
        psutil_mod: Any = psutil,
    ):
        self._ps = ps_runner
        self._reg = reg_reader
        self._psutil = psutil_mod

    def detect(self) -> HardwareInfo:
        raw = self._ps(HARDWARE_SCRIPT)
        raw = raw if isinstance(raw, dict) else {}
        info = HardwareInfo(
            cpu=self.detect_cpu(),
            gpus=parse_gpus(raw.get("gpus"), raw.get("gpu_registry")),
            memory=self.detect_memory(),
            disks=parse_disks(raw.get("disks"), raw.get("system_disk")),
            volumes=self.detect_volumes(),
            windows=self.detect_windows(),
            is_laptop=self.detect_is_laptop(),
            system_drive=os.environ.get("SystemDrive", "C:"),
        )
        log.info(
            "Hardware: CPU=%s GPUs=%s RAM=%.1fGB %s",
            info.cpu.name,
            [g.name for g in info.gpus],
            info.memory.total_bytes / 1024**3,
            info.windows.name,
        )
        return info

    def detect_cpu(self) -> CpuInfo:
        name = self._reg("HKLM", CPU_KEY, "ProcessorNameString") or platform.processor() or "CPU"
        freq = None
        try:
            f = self._psutil.cpu_freq()
            freq = float(f.max or f.current) if f else None
        except Exception:  # noqa: BLE001 - psutil puede fallar en VMs
            freq = None
        return CpuInfo(
            name=" ".join(str(name).split()),
            physical_cores=self._psutil.cpu_count(logical=False),
            logical_cores=self._psutil.cpu_count(logical=True),
            max_mhz=freq,
        )

    def detect_memory(self) -> MemoryInfo:
        vm = self._psutil.virtual_memory()
        return MemoryInfo(total_bytes=int(vm.total), available_bytes=int(vm.available))

    def detect_volumes(self) -> tuple[Volume, ...]:
        vols: list[Volume] = []
        for part in self._psutil.disk_partitions(all=False):
            if not part.fstype or "cdrom" in (part.opts or ""):
                continue
            try:
                usage = self._psutil.disk_usage(part.mountpoint)
            except (OSError, PermissionError):
                continue
            vols.append(Volume(part.mountpoint, part.fstype, int(usage.total), int(usage.free)))
        return tuple(vols)

    def detect_windows(self) -> WindowsInfo:
        build = self._reg("HKLM", WIN_KEY, "CurrentBuildNumber") or self._reg(
            "HKLM", WIN_KEY, "CurrentBuild"
        )
        return build_windows_info(
            build=build,
            display_version=self._reg("HKLM", WIN_KEY, "DisplayVersion"),
            edition=self._reg("HKLM", WIN_KEY, "EditionID"),
            ubr=self._reg("HKLM", WIN_KEY, "UBR"),
            fallback_version=platform.version(),
        )

    def detect_is_laptop(self) -> bool:
        try:
            return self._psutil.sensors_battery() is not None
        except Exception:  # noqa: BLE001
            return False
