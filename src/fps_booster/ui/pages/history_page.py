"""Historial de cambios y reversión."""

from __future__ import annotations

from PySide6.QtCore import Signal
from PySide6.QtWidgets import QHBoxLayout, QLabel, QPushButton, QVBoxLayout, QWidget

from fps_booster.ui import theme
from fps_booster.ui.pages.common import card, item, make_table, page_header, section

ACTION_LABELS = {
    "apply": ("Aplicado", theme.OK),
    "revert": ("Revertido", theme.ACCENT_2),
    "rollback": ("Deshecho por error", theme.WARN),
    "error": ("Error", theme.DANGER),
    "info": ("Info", theme.MUTED),
}


class HistoryPage(QWidget):
    revert_requested = Signal()

    def __init__(self, parent: QWidget | None = None):
        super().__init__(parent)
        self.setObjectName("Root")
        root = QVBoxLayout(self)
        root.setContentsMargins(28, 22, 28, 22)
        root.setSpacing(12)
        page_header(root, "Historial", "Registro de cada cambio y valores originales guardados.")

        frame, lay = card(accent=True)
        top = QHBoxLayout()
        top.addWidget(section("Valores originales guardados"))
        top.addStretch(1)
        self.revert_btn = QPushButton("Revertir todo a los valores originales")
        self.revert_btn.setObjectName("Danger")
        self.revert_btn.clicked.connect(self.revert_requested.emit)
        top.addWidget(self.revert_btn)
        lay.addLayout(top)
        self.pending_label = QLabel("No hay cambios pendientes de revertir.")
        self.pending_label.setWordWrap(True)
        lay.addWidget(self.pending_label)
        root.addWidget(frame)

        root.addWidget(section("Registro"))
        self.table = make_table(["Fecha", "Modo", "Acción", "Ajuste", "Detalle"])
        root.addWidget(self.table, 1)
        self.log_path = QLabel("")
        self.log_path.setObjectName("Muted")
        root.addWidget(self.log_path)

    def set_pending(self, pending: list[dict]) -> None:
        self.revert_btn.setEnabled(bool(pending))
        if not pending:
            self.pending_label.setText("No hay cambios pendientes de revertir.")
            return
        lines = [
            f"<b>{p['title']}</b>: original <span style='color:{theme.ACCENT_2}'>{p['original_label']}</span>"
            f" · actual {p['current_label']}" for p in pending
        ]
        self.pending_label.setText("<br>".join(lines))

    def set_entries(self, entries: list[dict], path: str) -> None:
        self.log_path.setText(f"Archivo: {path}")
        self.table.setRowCount(len(entries))
        for r, e in enumerate(entries):
            label, color = ACTION_LABELS.get(e.get("action"), (e.get("action"), theme.TEXT))
            self.table.setItem(r, 0, item(str(e.get("timestamp", ""))[:19].replace("T", " "), theme.MUTED))
            self.table.setItem(r, 1, item(e.get("mode", "")))
            self.table.setItem(r, 2, item(label, color))
            self.table.setItem(r, 3, item(e.get("optimization_id") or "—"))
            self.table.setItem(r, 4, item(e.get("message", "")))
