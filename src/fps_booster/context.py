"""Contenedor de dependencias. Es el ÚNICO lugar donde se decide simulación vs. real."""

from __future__ import annotations

import sys
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path

from fps_booster.core.paths import data_dir as resolve_data_dir
from fps_booster.hardware.detector import HardwareDetector
from fps_booster.monitoring.gpu import create_default_provider
from fps_booster.monitoring.sampler import ResourceSampler
from fps_booster.optimizations.catalog import default_catalog
from fps_booster.optimizations.engine import OptimizationEngine
from fps_booster.processes.analyzer import ProcessAnalyzer
from fps_booster.state.backup_store import BackupStore
from fps_booster.state.change_log import ChangeLog
from fps_booster.system.backend import (
    SimulatedSystemBackend,
    SystemBackend,
    SystemReader,
    WindowsSystemBackend,
    WindowsSystemReader,
)

MODE_SIMULATION = "simulacion"
MODE_REAL = "real"


@dataclass
class AppContext:
    mode: str
    data_dir: Path
    backend: SystemBackend
    backups: BackupStore
    change_log: ChangeLog
    engine: OptimizationEngine
    detector_factory: Callable[[], HardwareDetector]
    sampler_factory: Callable[[], ResourceSampler]
    analyzer_factory: Callable[[], ProcessAnalyzer]

    @property
    def is_simulation(self) -> bool:
        return self.mode == MODE_SIMULATION


def build_context(
    real_mode: bool = False,
    data_dir: Path | None = None,
    reader: SystemReader | None = None,
    detector_factory: Callable[[], HardwareDetector] | None = None,
    sampler_factory: Callable[[], ResourceSampler] | None = None,
    analyzer_factory: Callable[[], ProcessAnalyzer] | None = None,
) -> AppContext:
    base = resolve_data_dir(data_dir)
    if real_mode:
        if sys.platform != "win32":
            raise RuntimeError("El modo real solo está disponible en Windows")
        mode = MODE_REAL
        backend: SystemBackend = WindowsSystemBackend()
    else:
        mode = MODE_SIMULATION
        if reader is None and sys.platform == "win32":
            reader = WindowsSystemReader()  # solo lectura: no puede modificar Windows
        backend = SimulatedSystemBackend(reader)

    backups = BackupStore(base / "backups" / f"originals_{mode}.json")
    if mode == MODE_SIMULATION:
        backups.clear()  # cada sesión simulada parte de un sistema virtual limpio
    change_log = ChangeLog(base / "logs" / f"changes_{mode}.jsonl", mode)
    engine = OptimizationEngine(backend, backups, change_log, default_catalog())
    return AppContext(
        mode=mode,
        data_dir=base,
        backend=backend,
        backups=backups,
        change_log=change_log,
        engine=engine,
        detector_factory=detector_factory or HardwareDetector,
        sampler_factory=sampler_factory or (lambda: ResourceSampler(gpu_provider=create_default_provider())),
        analyzer_factory=analyzer_factory or ProcessAnalyzer,
    )
