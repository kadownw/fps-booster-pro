"""Detalle del hardware detectado y de las fuentes de métricas."""

from __future__ import annotations

from PySide6.QtWidgets import QGridLayout, QLabel, QVBoxLayout, QWidget

from fps_booster.core.formatting import fmt_bytes
from fps_booster.core.models import HardwareInfo
from fps_booster.ui import theme
from fps_booster.ui.pages.common import card, page_header, scroll_page, section


class SystemPage(QWidget):
    def __init__(self, parent: QWidget | None = None):
        super().__init__(parent)
        self.setObjectName("Root")
        outer = QVBoxLayout(self)
        outer.setContentsMargins(0, 0, 0, 0)
        area, lay = scroll_page()
        outer.addWidget(area)
        page_header(lay, "Sistema", "Hardware detectado (solo lectura).")
        self._grid = QGridLayout()
        self._grid.setSpacing(14)
        lay.addLayout(self._grid)
        lay.addStretch(1)
        self._loading = QLabel("Detectando hardware…")
        self._loading.setObjectName("Muted")
        self._grid.addWidget(self._loading, 0, 0)
        self.gpu_source = "—"

    def _block(self, title: str, lines: list[str], row: int, col: int, span: int = 1) -> None:
        frame, lay = card()
        lay.addWidget(section(title))
        text = QLabel("<br>".join(lines))
        text.setWordWrap(True)
        lay.addWidget(text)
        lay.addStretch(1)
        self._grid.addWidget(frame, row, col, 1, span)

    def set_hardware(self, hw: HardwareInfo, tier: str) -> None:
        while self._grid.count():
            w = self._grid.takeAt(0).widget()
            if w is not None:
                w.deleteLater()
        m = theme.MUTED
        cpu = hw.cpu
        self._block("Procesador", [
            f"<b>{cpu.name}</b>",
            f"<span style='color:{m}'>Núcleos:</span> {cpu.physical_cores or '?'} · "
            f"<span style='color:{m}'>Hilos:</span> {cpu.logical_cores or '?'}",
            f"<span style='color:{m}'>Frecuencia máx.:</span> "
            + (f"{cpu.max_mhz / 1000:.2f} GHz" if cpu.max_mhz else "N/D"),
        ], 0, 0)
        gpu_lines = []
        for g in hw.gpus:
            kind = "integrada" if g.integrated else "dedicada"
            gpu_lines.append(f"<b>{g.name}</b> <span style='color:{m}'>({kind})</span>")
            gpu_lines.append(f"<span style='color:{m}'>VRAM:</span> {fmt_bytes(g.vram_bytes)} · "
                             f"<span style='color:{m}'>Driver:</span> {g.driver_version or 'N/D'}")
        self._block("Gráficos", gpu_lines or ["No detectada"], 0, 1)
        self._block("Memoria", [
            f"<b>{fmt_bytes(hw.memory.total_bytes)}</b> instalada",
            f"<span style='color:{m}'>Disponible al iniciar:</span> {fmt_bytes(hw.memory.available_bytes)}",
        ], 1, 0)
        win = hw.windows
        self._block("Windows", [
            f"<b>{win.name}</b> {win.edition or ''}",
            f"<span style='color:{m}'>Versión:</span> {win.display_version or 'N/D'} · "
            f"<span style='color:{m}'>Compilación:</span> {win.full_build}",
            f"<span style='color:{m}'>Tipo de equipo:</span> {'Portátil' if hw.is_laptop else 'Escritorio'}",
            f"<span style='color:{m}'>Nivel estimado:</span> gama {tier}",
        ], 1, 1)
        sys_tag = f"  <span style='color:{theme.ACCENT_2}'>(sistema)</span>"
        disk_lines = [
            f"<b>{d.name}</b> · {d.media_type}{' · ' + d.bus_type if d.bus_type else ''} · "
            f"{fmt_bytes(d.size_bytes, 0)}{sys_tag if d.is_system else ''}"
            for d in hw.disks
        ]
        disk_lines += [
            f"<span style='color:{m}'>{v.mountpoint}</span> {v.fstype} · libre {fmt_bytes(v.free_bytes)} "
            f"de {fmt_bytes(v.total_bytes)} ({v.percent_used:.0f}% usado)" for v in hw.volumes
        ]
        self._block("Almacenamiento", disk_lines or ["N/D"], 2, 0, 2)
        self._block("Fuentes de métricas", [
            "<span style='color:%s'>CPU / RAM / disco / red:</span> psutil" % m,
            f"<span style='color:{m}'>GPU:</span> {self.gpu_source}",
            f"<span style='color:{m}'>Temperaturas:</span> solo GPU NVIDIA (nvidia-smi). CPU: fase 2",
            f"<span style='color:{m}'>FPS:</span> no medido todavía — integración con PresentMon en fase 2",
        ], 3, 0, 2)
