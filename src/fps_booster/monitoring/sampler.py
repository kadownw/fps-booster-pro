"""Muestreo periódico de recursos con psutil (no bloqueante)."""

from __future__ import annotations

import time
from collections.abc import Callable
from typing import Any

import psutil

from fps_booster.core.logging_setup import get_logger
from fps_booster.core.models import ResourceSample
from fps_booster.monitoring.gpu import GpuMetricsProvider

log = get_logger("sampler")


class ResourceSampler:
    def __init__(
        self,
        psutil_mod: Any = psutil,
        gpu_provider: GpuMetricsProvider | None = None,
        clock: Callable[[], float] = time.monotonic,
    ):
        self._ps = psutil_mod
        self._gpu = gpu_provider
        self._clock = clock
        self._last_t: float | None = None
        self._last_disk: tuple[int, int] | None = None
        self._last_net: tuple[int, int] | None = None
        self._last_gpu = None
        # Primera llamada: psutil necesita una referencia para calcular porcentajes.
        self._ps.cpu_percent(interval=None)
        self._ps.cpu_percent(interval=None, percpu=True)

    @property
    def gpu_source(self) -> str:
        return self._gpu.name if self._gpu else "no disponible"

    def _io(self) -> tuple[tuple[int, int] | None, tuple[int, int] | None]:
        disk = net = None
        try:
            d = self._ps.disk_io_counters()
            disk = (int(d.read_bytes), int(d.write_bytes)) if d else None
        except Exception:  # noqa: BLE001
            disk = None
        try:
            n = self._ps.net_io_counters()
            net = (int(n.bytes_sent), int(n.bytes_recv)) if n else None
        except Exception:  # noqa: BLE001
            net = None
        return disk, net

    @staticmethod
    def _rate(now: tuple[int, int] | None, prev: tuple[int, int] | None, dt: float):
        if not now or not prev or dt <= 0:
            return 0.0, 0.0
        return max(0.0, (now[0] - prev[0]) / dt), max(0.0, (now[1] - prev[1]) / dt)

    def sample(self, include_gpu: bool = True) -> ResourceSample:
        now = self._clock()
        cpu = float(self._ps.cpu_percent(interval=None))
        per_cpu = tuple(float(x) for x in self._ps.cpu_percent(interval=None, percpu=True))
        vm = self._ps.virtual_memory()
        disk, net = self._io()
        dt = (now - self._last_t) if self._last_t is not None else 0.0
        d_read, d_write = self._rate(disk, self._last_disk, dt)
        n_sent, n_recv = self._rate(net, self._last_net, dt)
        self._last_t, self._last_disk, self._last_net = now, disk, net

        if include_gpu and self._gpu is not None:
            try:
                self._last_gpu = self._gpu.sample()
            except Exception as exc:  # noqa: BLE001 - nunca romper el monitoreo por la GPU
                log.debug("Error leyendo GPU: %s", exc)
                self._last_gpu = None

        return ResourceSample(
            timestamp=time.time(),
            cpu_percent=cpu,
            per_cpu=per_cpu,
            ram_percent=float(vm.percent),
            ram_used_bytes=int(vm.total - vm.available),
            ram_total_bytes=int(vm.total),
            disk_read_bps=d_read,
            disk_write_bps=d_write,
            net_sent_bps=n_sent,
            net_recv_bps=n_recv,
            gpu=self._last_gpu,
        )

    def close(self) -> None:
        if self._gpu is not None:
            self._gpu.close()
