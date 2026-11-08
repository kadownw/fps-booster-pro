"""Diálogo de consentimiento: el único camino de la UI para crear un ``Consent``."""

from __future__ import annotations

from PySide6.QtCore import Qt
from PySide6.QtWidgets import (
    QCheckBox,
    QDialog,
    QFrame,
    QHBoxLayout,
    QLabel,
    QPushButton,
    QScrollArea,
    QVBoxLayout,
    QWidget,
)

from fps_booster.optimizations.engine import Consent, PlannedChange
from fps_booster.ui import theme


class ConsentDialog(QDialog):
    def __init__(self, title: str, plan: list[PlannedChange], simulated: bool,
                 parent: QWidget | None = None):
        super().__init__(parent)
        self.setWindowTitle("Revisar cambios — FPS BOOSTER")
        self.setMinimumSize(720, 520)
        self.plan = plan
        self._boxes: dict[str, QCheckBox] = {}

        root = QVBoxLayout(self)
        root.setContentsMargins(24, 20, 24, 20)
        root.setSpacing(12)
        heading = QLabel(title)
        heading.setObjectName("PageTitle")
        root.addWidget(heading)

        banner = QLabel(
            "MODO SIMULACIÓN — los cambios se aplican a un sistema virtual. Tu Windows NO se modifica."
            if simulated else
            "MODO REAL — estos cambios modificarán la configuración de Windows."
        )
        banner.setObjectName("ModeSim" if simulated else "ModeReal")
        banner.setWordWrap(True)
        root.addWidget(banner)

        info = QLabel("Elegí qué cambios aplicar. Antes de cada uno se guarda el valor original "
                      "para poder revertirlo desde Historial.")
        info.setObjectName("Muted")
        info.setWordWrap(True)
        root.addWidget(info)

        scroll = QScrollArea()
        scroll.setWidgetResizable(True)
        holder = QWidget()
        rows = QVBoxLayout(holder)
        rows.setSpacing(8)
        rows.setContentsMargins(0, 0, 8, 0)
        for change in plan:
            rows.addWidget(self._row(change))
        rows.addStretch(1)
        scroll.setWidget(holder)
        root.addWidget(scroll, 1)

        self.ack = QCheckBox("Entiendo estos cambios y autorizo aplicarlos.")
        self.ack.toggled.connect(self._update_state)
        root.addWidget(self.ack)

        buttons = QHBoxLayout()
        buttons.addStretch(1)
        cancel = QPushButton("Cancelar")
        cancel.clicked.connect(self.reject)
        self.apply_btn = QPushButton("Aplicar cambios")
        self.apply_btn.setObjectName("Primary")
        self.apply_btn.clicked.connect(self.accept)
        buttons.addWidget(cancel)
        buttons.addWidget(self.apply_btn)
        root.addLayout(buttons)
        self._update_state()

    def _row(self, change: PlannedChange) -> QFrame:
        frame = QFrame()
        frame.setObjectName("Card")
        lay = QHBoxLayout(frame)
        lay.setContentsMargins(14, 10, 14, 10)
        box = QCheckBox()
        box.setChecked(change.selected)
        box.setEnabled(change.actionable)
        box.toggled.connect(self._update_state)
        self._boxes[change.id] = box
        lay.addWidget(box, 0, Qt.AlignmentFlag.AlignTop)

        text = QVBoxLayout()
        opt = change.optimization
        name = QLabel(f"<b>{opt.title}</b>" + ("  <span style='color:%s'>(opcional)</span>" % theme.MUTED
                                                if change.optional else ""))
        text.addWidget(name)
        arrow = QLabel(
            f"<span style='color:{theme.MUTED}'>Actual:</span> {change.current_label}"
            f"  <span style='color:{theme.ACCENT_2}'>→</span>  "
            f"<span style='color:{theme.MUTED}'>Nuevo:</span> <b>{change.target_label}</b>"
        )
        text.addWidget(arrow)
        desc = QLabel(opt.description)
        desc.setObjectName("Muted")
        desc.setWordWrap(True)
        text.addWidget(desc)
        flags = [f"Riesgo {opt.risk}", "Reversible" if opt.reversible else "NO reversible"]
        if opt.requires_restart:
            flags.append("Requiere cerrar sesión")
        if opt.requires_admin:
            flags.append("Requiere administrador")
        color = theme.WARN if not change.available else theme.MUTED
        note = f" · {change.note}" if change.note else ""
        meta = QLabel(f"<span style='color:{color}'>{' · '.join(flags)}{note}</span>")
        meta.setWordWrap(True)
        text.addWidget(meta)
        lay.addLayout(text, 1)
        return frame

    def selected_ids(self) -> list[str]:
        return [cid for cid, box in self._boxes.items() if box.isChecked() and box.isEnabled()]

    def _update_state(self) -> None:
        self.apply_btn.setEnabled(self.ack.isChecked() and bool(self.selected_ids()))

    def result_consent(self) -> Consent | None:
        """Devuelve el consentimiento solo si el usuario aceptó y marcó la confirmación."""
        if self.result() != QDialog.DialogCode.Accepted or not self.ack.isChecked():
            return None
        ids = self.selected_ids()
        if not ids:
            return None
        for change in self.plan:
            change.selected = change.id in ids
        return Consent.grant(ids)
