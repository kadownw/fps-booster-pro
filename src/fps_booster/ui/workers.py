"""Trabajo en segundo plano: la UI nunca se bloquea esperando a psutil, PDH o PowerShell."""

from __future__ import annotations

from collections.abc import Callable

from PySide6.QtCore import QObject, QThread, QTimer, Signal, Slot

from fps_booster.core.logging_setup import get_logger

log = get_logger("workers")


class MonitorWorker(QObject):
    sample_ready = Signal(object)  # ResourceSample
    processes_ready = Signal(object)  # list[ProcessInfo]
    gpu_source_ready = Signal(str)

    def __init__(self, sampler_factory: Callable, analyzer_factory: Callable,
                 interval_ms: int = 1000, process_every: int = 3, gpu_every: int = 2):
        super().__init__()
        self._sampler_factory = sampler_factory
        self._analyzer_factory = analyzer_factory
        self._interval = interval_ms
        self._process_every = process_every
        self._gpu_every = gpu_every
        self._tick_count = 0
        self._sampler = None
        self._analyzer = None
        self._timer: QTimer | None = None

    @Slot()
    def start(self) -> None:
        # Se crean dentro del hilo del worker (PDH tiene afinidad de hilo).
        self._sampler = self._sampler_factory()
        self._analyzer = self._analyzer_factory()
        self.gpu_source_ready.emit(self._sampler.gpu_source)
        self._timer = QTimer(self)
        self._timer.timeout.connect(self._tick)
        # Sin tick inmediato: psutil necesita un intervalo real para que el primer % de CPU sea válido.
        self._timer.start(self._interval)

    @Slot()
    def _tick(self) -> None:
        try:
            include_gpu = self._tick_count % self._gpu_every == 0
            self.sample_ready.emit(self._sampler.sample(include_gpu=include_gpu))
            if self._tick_count % self._process_every == 0:
                self.processes_ready.emit(self._analyzer.snapshot())
        except Exception:  # noqa: BLE001
            log.exception("Error en el ciclo de monitoreo")
        self._tick_count += 1

    @Slot()
    def stop(self) -> None:
        if self._timer is not None:
            self._timer.stop()
        if self._sampler is not None:
            self._sampler.close()


class HardwareThread(QThread):
    detected = Signal(object)  # HardwareInfo
    failed = Signal(str)

    def __init__(self, detector_factory: Callable, parent: QObject | None = None):
        super().__init__(parent)
        self._factory = detector_factory

    def run(self) -> None:
        try:
            self.detected.emit(self._factory().detect())
        except Exception as exc:  # noqa: BLE001
            log.exception("Fallo la detección de hardware")
            self.failed.emit(str(exc))
