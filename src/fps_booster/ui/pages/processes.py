"""Procesos: solo lectura. FPS BOOSTER no cierra procesos."""

from __future__ import annotations

from PySide6.QtWidgets import QCheckBox, QHBoxLayout, QLabel, QVBoxLayout, QWidget

from fps_booster.core.formatting import fmt_bytes
from fps_booster.core.models import ProcessInfo
from fps_booster.ui import theme
from fps_booster.ui.pages.common import item, make_table, page_header

CATEGORY_LABELS = {
    "protegido": ("Sistema (no cerrar)", theme.MUTED),
    "segundo_plano": ("Segundo plano", theme.ACCENT_2),
    "usuario": ("Aplicación", theme.TEXT),
    "propio": ("FPS BOOSTER", theme.OK),
}


class ProcessesPage(QWidget):
    MAX_ROWS = 150

    def __init__(self, parent: QWidget | None = None):
        super().__init__(parent)
        self.setObjectName("Root")
        self._last: list[ProcessInfo] = []
        root = QVBoxLayout(self)
        root.setContentsMargins(28, 22, 28, 22)
        root.setSpacing(12)
        page_header(root, "Procesos",
                    "FPS BOOSTER solo identifica procesos de alto consumo: nunca los cierra. "
                    "Si querés cerrar alguno, hacelo vos desde la aplicación o el Administrador de tareas.")
        bar = QHBoxLayout()
        self.only_heavy = QCheckBox("Mostrar solo alto consumo")
        self.only_heavy.toggled.connect(lambda _=None: self.update_processes(self._last))
        bar.addWidget(self.only_heavy)
        bar.addStretch(1)
        self.summary = QLabel("")
        self.summary.setObjectName("Muted")
        bar.addWidget(self.summary)
        root.addLayout(bar)
        self.table = make_table(["Proceso", "PID", "CPU", "RAM", "Tipo", "Observación"])
        root.addWidget(self.table, 1)

    def update_processes(self, processes: list[ProcessInfo]) -> None:
        self._last = processes
        rows = [p for p in processes if p.heavy] if self.only_heavy.isChecked() else processes
        rows = rows[: self.MAX_ROWS]
        heavy = sum(1 for p in processes if p.heavy and p.category != "protegido")
        self.summary.setText(f"{len(processes)} procesos · {heavy} con consumo alto")
        self.table.setUpdatesEnabled(False)
        self.table.setRowCount(len(rows))
        for r, p in enumerate(rows):
            label, color = CATEGORY_LABELS.get(p.category, (p.category, theme.TEXT))
            name_color = theme.WARN if p.heavy and p.category != "protegido" else None
            self.table.setItem(r, 0, item(p.name, name_color))
            self.table.setItem(r, 1, item(p.pid, theme.MUTED))
            self.table.setItem(r, 2, item(f"{p.cpu_percent:.1f}%"))
            self.table.setItem(r, 3, item(fmt_bytes(p.memory_bytes)))
            self.table.setItem(r, 4, item(label, color))
            obs = " · ".join(p.reasons)
            if p.category == "protegido" and p.heavy:
                obs += " · proceso de Windows, no se recomienda cerrarlo"
            self.table.setItem(r, 5, item(obs, theme.MUTED))
        self.table.setUpdatesEnabled(True)
