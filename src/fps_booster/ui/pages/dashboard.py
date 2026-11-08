"""Panel principal: métricas, gráfico en vivo y botón de optimización."""

from __future__ import annotations

from PySide6.QtCore import Qt, Signal
from PySide6.QtWidgets import (
    QComboBox,
    QGridLayout,
    QHBoxLayout,
    QLabel,
    QVBoxLayout,
    QWidget,
)

from fps_booster.core.formatting import fmt_bytes, fmt_rate
from fps_booster.core.models import HardwareInfo, ProcessInfo, ResourceSample
from fps_booster.processes.analyzer import heavy_candidates
from fps_booster.profiles.profiles import PROFILES
from fps_booster.ui import theme
from fps_booster.ui.pages.common import card, section
from fps_booster.ui.widgets.line_chart import LiveChart
from fps_booster.ui.widgets.optimize_button import OptimizeButton
from fps_booster.ui.widgets.stat_card import StatCard


class DashboardPage(QWidget):
    optimize_requested = Signal(str)  # id de perfil

    def __init__(self, parent: QWidget | None = None):
        super().__init__(parent)
        self.setObjectName("Root")
        self._hw: HardwareInfo | None = None
        root = QVBoxLayout(self)
        root.setContentsMargins(28, 22, 28, 22)
        root.setSpacing(16)

        cards = QHBoxLayout()
        cards.setSpacing(14)
        self.cpu_card = StatCard("CPU", theme.ACCENT_2)
        self.gpu_card = StatCard("GPU", theme.PURPLE)
        self.ram_card = StatCard("RAM", theme.ACCENT)
        self.disk_card = StatCard("Disco", theme.OK)
        for c in (self.cpu_card, self.gpu_card, self.ram_card, self.disk_card):
            cards.addWidget(c, 1)
        root.addLayout(cards)

        middle = QHBoxLayout()
        middle.setSpacing(14)
        self.chart = LiveChart("RENDIMIENTO EN TIEMPO REAL", max_points=90)
        self.chart.add_series("CPU", theme.ACCENT_2)
        self.chart.add_series("RAM", theme.ACCENT)
        self.chart.add_series("GPU", theme.PURPLE)
        middle.addWidget(self.chart, 3)

        boost, boost_lay = card(accent=True)
        boost_lay.setAlignment(Qt.AlignmentFlag.AlignHCenter)
        boost_lay.addWidget(section("Optimización"), 0, Qt.AlignmentFlag.AlignHCenter)
        self.button = OptimizeButton()
        self.button.clicked.connect(self._emit_optimize)
        boost_lay.addWidget(self.button, 0, Qt.AlignmentFlag.AlignHCenter)
        self.profile_combo = QComboBox()
        for p in PROFILES.values():
            self.profile_combo.addItem(f"Perfil {p.name} — {p.tagline}", p.id)
        self.profile_combo.currentIndexChanged.connect(self._profile_changed)
        boost_lay.addWidget(self.profile_combo)
        self.status = QLabel("Revisás cada cambio antes de aplicarlo.")
        self.status.setObjectName("Muted")
        self.status.setWordWrap(True)
        self.status.setAlignment(Qt.AlignmentFlag.AlignCenter)
        boost_lay.addWidget(self.status)
        boost.setMinimumWidth(300)
        middle.addWidget(boost, 1)
        root.addLayout(middle, 1)

        bottom = QHBoxLayout()
        bottom.setSpacing(14)
        proc_card, proc_lay = card()
        proc_lay.addWidget(section("Mayor consumo ahora"))
        self.proc_grid = QGridLayout()
        self.proc_grid.setHorizontalSpacing(18)
        proc_lay.addLayout(self.proc_grid)
        proc_lay.addStretch(1)
        bottom.addWidget(proc_card, 3)

        sys_card, sys_lay = card()
        sys_lay.addWidget(section("Tu equipo"))
        self.sys_label = QLabel("Detectando hardware…")
        self.sys_label.setWordWrap(True)
        self.sys_label.setObjectName("Muted")
        sys_lay.addWidget(self.sys_label)
        sys_lay.addStretch(1)
        bottom.addWidget(sys_card, 2)
        root.addLayout(bottom)

    # --------------------------------------------------------------- slots
    def _profile_changed(self) -> None:
        profile = PROFILES[self.profile_combo.currentData()]
        self.button.set_subtitle(f"Perfil {profile.name}")

    def _emit_optimize(self) -> None:
        self.optimize_requested.emit(self.profile_combo.currentData())

    def set_status(self, text: str) -> None:
        self.status.setText(text)

    def set_hardware(self, hw: HardwareInfo, tier: str) -> None:
        self._hw = hw
        gpu = hw.primary_gpu()
        gpu_txt = gpu.name if gpu else "No detectada"
        self.sys_label.setText(
            f"<b style='color:{theme.TEXT}'>{hw.cpu.name}</b><br>"
            f"{gpu_txt}<br>{fmt_bytes(hw.memory.total_bytes, 0)} RAM · {hw.windows.name} "
            f"{hw.windows.display_version or ''}<br>"
            f"Nivel estimado: <b style='color:{theme.ACCENT_2}'>gama {tier}</b>"
        )

    def update_sample(self, s: ResourceSample) -> None:
        cores = len(s.per_cpu)
        self.cpu_card.set_value(f"{s.cpu_percent:.0f}%", f"{cores} hilos · pico núcleo "
                                f"{max(s.per_cpu, default=0):.0f}%", s.cpu_percent)
        self.ram_card.set_value(f"{s.ram_percent:.0f}%",
                                f"{fmt_bytes(s.ram_used_bytes)} de {fmt_bytes(s.ram_total_bytes)}",
                                s.ram_percent)
        self.disk_card.set_value(fmt_rate(s.disk_read_bps + s.disk_write_bps),
                                 f"L {fmt_rate(s.disk_read_bps)} · E {fmt_rate(s.disk_write_bps)}",
                                 min(100.0, (s.disk_read_bps + s.disk_write_bps) / (2 * 1024**2)))
        g = s.gpu
        if g is not None and g.utilization is not None:
            parts = []
            if g.memory_used_bytes is not None:
                total = g.memory_total_bytes or (self._gpu_vram() or None)
                parts.append(f"VRAM {fmt_bytes(g.memory_used_bytes)}"
                             + (f" / {fmt_bytes(total)}" if total else ""))
            if g.temperature_c is not None:
                parts.append(f"{g.temperature_c:.0f} °C")
            self.gpu_card.set_value(f"{g.utilization:.0f}%", " · ".join(parts) or g.source,
                                    g.utilization)
        else:
            self.gpu_card.set_value("N/D", "Métrica de GPU no disponible", None)
        self.chart.add_value("CPU", s.cpu_percent)
        self.chart.add_value("RAM", s.ram_percent)
        self.chart.add_value("GPU", g.utilization if g else None)

    def _gpu_vram(self) -> int | None:
        gpu = self._hw.primary_gpu() if self._hw else None
        return gpu.vram_bytes if gpu else None

    def update_processes(self, processes: list[ProcessInfo]) -> None:
        while self.proc_grid.count():
            w = self.proc_grid.takeAt(0).widget()
            if w is not None:
                w.deleteLater()
        top = [p for p in processes if p.category not in ("protegido",)][:5]
        flagged = {p.pid for p in heavy_candidates(processes)}
        for row, p in enumerate(top):
            color = theme.WARN if p.pid in flagged else theme.TEXT
            name = QLabel(p.name)
            name.setStyleSheet(f"color: {color};")
            cpu = QLabel(f"{p.cpu_percent:.1f}% CPU")
            cpu.setObjectName("Muted")
            mem = QLabel(fmt_bytes(p.memory_bytes))
            mem.setObjectName("Muted")
            self.proc_grid.addWidget(name, row, 0)
            self.proc_grid.addWidget(cpu, row, 1)
            self.proc_grid.addWidget(mem, row, 2)
