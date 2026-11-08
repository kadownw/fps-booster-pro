"""Modelos de datos inmutables compartidos por todas las capas."""

from __future__ import annotations

from dataclasses import asdict, dataclass, field


@dataclass(frozen=True)
class CpuInfo:
    name: str
    physical_cores: int | None
    logical_cores: int | None
    max_mhz: float | None


@dataclass(frozen=True)
class GpuInfo:
    name: str
    vendor: str  # nvidia | amd | intel | microsoft | desconocido
    vram_bytes: int | None
    driver_version: str | None
    integrated: bool


@dataclass(frozen=True)
class MemoryInfo:
    total_bytes: int
    available_bytes: int


@dataclass(frozen=True)
class PhysicalDisk:
    name: str
    media_type: str  # SSD | HDD | Desconocido
    bus_type: str | None
    size_bytes: int | None
    is_system: bool = False


@dataclass(frozen=True)
class Volume:
    mountpoint: str
    fstype: str
    total_bytes: int
    free_bytes: int

    @property
    def percent_used(self) -> float:
        if self.total_bytes <= 0:
            return 0.0
        return round(100.0 * (self.total_bytes - self.free_bytes) / self.total_bytes, 1)


@dataclass(frozen=True)
class WindowsInfo:
    name: str  # "Windows 11", "Windows 10", ...
    edition: str | None
    display_version: str | None  # p. ej. 24H2
    build: int | None
    ubr: int | None

    @property
    def is_supported(self) -> bool:
        return self.name in ("Windows 10", "Windows 11")

    @property
    def full_build(self) -> str:
        if self.build is None:
            return "desconocida"
        return f"{self.build}.{self.ubr}" if self.ubr is not None else str(self.build)


@dataclass(frozen=True)
class HardwareInfo:
    cpu: CpuInfo
    gpus: tuple[GpuInfo, ...]
    memory: MemoryInfo
    disks: tuple[PhysicalDisk, ...]
    volumes: tuple[Volume, ...]
    windows: WindowsInfo
    is_laptop: bool
    system_drive: str = "C:"

    def primary_gpu(self) -> GpuInfo | None:
        """GPU dedicada con más VRAM; si no hay, la primera disponible."""
        real = [g for g in self.gpus if g.vendor != "microsoft"] or list(self.gpus)
        if not real:
            return None
        dedicated = [g for g in real if not g.integrated]
        pool = dedicated or real
        return max(pool, key=lambda g: g.vram_bytes or 0)

    def system_disk(self) -> PhysicalDisk | None:
        for disk in self.disks:
            if disk.is_system:
                return disk
        return self.disks[0] if len(self.disks) == 1 else None

    def system_volume(self) -> Volume | None:
        prefix = self.system_drive.rstrip("\\").upper()
        for vol in self.volumes:
            if vol.mountpoint.rstrip("\\").upper() == prefix:
                return vol
        return None

    def to_dict(self) -> dict:
        return asdict(self)


@dataclass(frozen=True)
class GpuSample:
    utilization: float | None
    memory_used_bytes: int | None
    memory_total_bytes: int | None
    temperature_c: float | None
    source: str


@dataclass(frozen=True)
class ResourceSample:
    timestamp: float
    cpu_percent: float
    per_cpu: tuple[float, ...]
    ram_percent: float
    ram_used_bytes: int
    ram_total_bytes: int
    disk_read_bps: float
    disk_write_bps: float
    net_sent_bps: float
    net_recv_bps: float
    gpu: GpuSample | None = None


@dataclass(frozen=True)
class ProcessInfo:
    pid: int
    name: str
    cpu_percent: float  # normalizado 0–100 sobre todos los núcleos
    memory_bytes: int
    category: str  # protegido | segundo_plano | usuario | propio
    heavy: bool = False
    reasons: tuple[str, ...] = field(default_factory=tuple)
