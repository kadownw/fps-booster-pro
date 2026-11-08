"""Perfiles Gaming / Equilibrado / Ahorro."""

from __future__ import annotations

from PySide6.QtCore import Signal
from PySide6.QtWidgets import QHBoxLayout, QLabel, QPushButton, QVBoxLayout, QWidget

from fps_booster.optimizations.catalog import default_catalog
from fps_booster.profiles.profiles import PROFILES
from fps_booster.ui import theme
from fps_booster.ui.pages.common import card, page_header, section

TARGET_LABELS = {
    "high_performance": "Alto rendimiento", "balanced": "Equilibrado", "power_saver": "Economizador",
    "on": "activar", "off": "desactivar",
}


class ProfilesPage(QWidget):
    review_requested = Signal(str)

    def __init__(self, parent: QWidget | None = None):
        super().__init__(parent)
        self.setObjectName("Root")
        root = QVBoxLayout(self)
        root.setContentsMargins(28, 22, 28, 22)
        root.setSpacing(14)
        page_header(root, "Perfiles",
                    "Un perfil es un conjunto de ajustes. Al elegirlo vas a ver cada cambio y decidir "
                    "cuáles aplicar. Todos se pueden revertir desde Historial.")
        row = QHBoxLayout()
        row.setSpacing(14)
        catalog = default_catalog()
        for profile in PROFILES.values():
            frame, lay = card(accent=profile.id == "gaming")
            lay.addWidget(section(profile.name))
            tag = QLabel(profile.tagline)
            tag.setStyleSheet(f"font-size: 18px; font-weight: 600; color: {theme.TEXT};")
            lay.addWidget(tag)
            desc = QLabel(profile.description)
            desc.setObjectName("Muted")
            desc.setWordWrap(True)
            lay.addWidget(desc)
            lay.addSpacing(6)
            for s in profile.settings:
                opt = catalog[s.optimization_id]
                extra = " <span style='color:%s'>(opcional)</span>" % theme.MUTED if s.optional else ""
                lbl = QLabel(f"<span style='color:{theme.ACCENT_2}'>▸</span> {opt.title}: "
                             f"<b>{TARGET_LABELS.get(s.target, s.target)}</b>{extra}")
                lbl.setWordWrap(True)
                lay.addWidget(lbl)
            lay.addStretch(1)
            btn = QPushButton("Revisar y aplicar")
            btn.setObjectName("Primary" if profile.id == "gaming" else "")
            btn.clicked.connect(lambda _=False, pid=profile.id: self.review_requested.emit(pid))
            lay.addWidget(btn)
            row.addWidget(frame, 1)
        root.addLayout(row)
        note = QLabel("FPS BOOSTER no promete aumentos de FPS: el efecto depende de cada equipo y juego. "
                      "La medición de FPS antes/después llega en la fase 2.")
        note.setObjectName("Muted")
        note.setWordWrap(True)
        root.addWidget(note)
        root.addStretch(1)
