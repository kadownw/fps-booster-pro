"""Fixtures compartidas. Ninguna prueba modifica la configuración real de Windows."""

from __future__ import annotations

import os
import subprocess
from types import SimpleNamespace

import pytest

os.environ.setdefault("QT_QPA_PLATFORM", "offscreen")

from fps_booster.core.models import (  # noqa: E402
    CpuInfo,
    GpuInfo,
    HardwareInfo,
    MemoryInfo,
    PhysicalDisk,
    Volume,
    WindowsInfo,
)
from fps_booster.optimizations.catalog import default_catalog  # noqa: E402
from fps_booster.optimizations.engine import OptimizationEngine  # noqa: E402
from fps_booster.state.backup_store import BackupStore  # noqa: E402
from fps_booster.state.change_log import ChangeLog  # noqa: E402
from fps_booster.system.backend import POWER_SCHEMES, SimulatedSystemBackend  # noqa: E402

GB = 1024**3


@pytest.fixture(autouse=True)
def _isolate(tmp_path, monkeypatch):
    """Datos en carpeta temporal y prohibido ejecutar comandos que escriban en el sistema."""
    monkeypatch.setenv("FPS_BOOSTER_DATA_DIR", str(tmp_path / "data"))
    real_run = subprocess.run

    def guarded_run(cmd, *args, **kwargs):
        text = " ".join(map(str, cmd)) if isinstance(cmd, (list, tuple)) else str(cmd)
        lowered = text.lower()
        forbidden = ("/setactive", "reg add", "reg delete", "taskkill", "set-itemproperty",
                     "stop-process", "bcdedit", "/change", "/duplicatescheme")
        if any(f in lowered for f in forbidden):
            raise AssertionError(f"Una prueba intentó modificar el sistema: {text}")
        return real_run(cmd, *args, **kwargs)

    monkeypatch.setattr(subprocess, "run", guarded_run)
    yield


@pytest.fixture
def sim_backend():
    return SimulatedSystemBackend(
        reader=None,
        active_scheme=POWER_SCHEMES["balanced"],
        registry_values={("HKCU", r"Software\Microsoft\GameBar", "AutoGameModeEnabled"): 0},
    )


@pytest.fixture
def engine(tmp_path, sim_backend):
    backups = BackupStore(tmp_path / "originals.json")
    log = ChangeLog(tmp_path / "changes.jsonl", "simulacion")
    return OptimizationEngine(sim_backend, backups, log, default_catalog())


def make_hw(*, threads=12, cores=6, ram_gb=32, gpu_name="NVIDIA GeForce RTX 4070", vram_gb=12,
            integrated=False, laptop=False, media="SSD", free_ratio=0.5, build=26100) -> HardwareInfo:
    vendor = "nvidia" if "nvidia" in gpu_name.lower() else "intel" if "intel" in gpu_name.lower() else "amd"
    total = 500 * GB
    return HardwareInfo(
        cpu=CpuInfo("Test CPU", cores, threads, 4000.0),
        gpus=(GpuInfo(gpu_name, vendor, int(vram_gb * GB), "1.0", integrated),),
        memory=MemoryInfo(int(ram_gb * GB), int(ram_gb * GB / 2)),
        disks=(PhysicalDisk("Disk0", media, "NVMe", total, is_system=True),),
        volumes=(Volume("C:\\", "NTFS", total, int(total * free_ratio)),),
        windows=WindowsInfo("Windows 11" if build >= 22000 else "Windows 10", "Professional",
                            "24H2", build, 1),
        is_laptop=laptop,
        system_drive="C:",
    )


class FakeProc:
    def __init__(self, pid, name, cpu, rss):
        self.info = {"pid": pid, "name": name, "memory_info": SimpleNamespace(rss=rss)}
        self._cpu = cpu
        self.killed = False

    def cpu_percent(self, interval=None):
        return self._cpu


class FakePsutil:
    def __init__(self, procs=(), logical=8):
        self.procs = list(procs)
        self.logical = logical
        self.read = 0
        self.sent = 0

    def cpu_percent(self, interval=None, percpu=False):
        return [10.0] * self.logical if percpu else 25.0

    def cpu_count(self, logical=True):
        return self.logical if logical else self.logical // 2

    def cpu_freq(self):
        return SimpleNamespace(current=3000.0, max=4200.0)

    def virtual_memory(self):
        return SimpleNamespace(total=16 * GB, available=6 * GB, percent=62.5)

    def disk_io_counters(self):
        self.read += 10 * 1024 * 1024
        return SimpleNamespace(read_bytes=self.read, write_bytes=self.read // 2)

    def net_io_counters(self):
        self.sent += 1024
        return SimpleNamespace(bytes_sent=self.sent, bytes_recv=self.sent * 2)

    def disk_partitions(self, all=False):  # noqa: A002
        return [SimpleNamespace(mountpoint="C:\\", fstype="NTFS", opts="rw,fixed"),
                SimpleNamespace(mountpoint="D:\\", fstype="", opts="cdrom")]

    def disk_usage(self, path):
        return SimpleNamespace(total=500 * GB, free=200 * GB)

    def sensors_battery(self):
        return None

    def process_iter(self, attrs=None):
        return iter(self.procs)


@pytest.fixture
def fake_psutil():
    return FakePsutil
