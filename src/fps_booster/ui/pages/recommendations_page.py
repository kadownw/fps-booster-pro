"""Recomendaciones según el hardware."""

from __future__ import annotations

from PySide6.QtCore import Signal
from PySide6.QtWidgets import QHBoxLayout, QLabel, QPushButton, QVBoxLayout, QWidget

from fps_booster.recommendations.recommender import Recommendation
from fps_booster.ui import theme
from fps_booster.ui.pages.common import card, page_header, scroll_page

PRIORITY = {1: ("ALTA", theme.WARN), 2: ("MEDIA", theme.ACCENT_2), 3: ("BAJA", theme.MUTED)}


class RecommendationsPage(QWidget):
    review_requested = Signal()  # aplicar las recomendaciones automáticas

    def __init__(self, parent: QWidget | None = None):
        super().__init__(parent)
        self.setObjectName("Root")
        outer = QVBoxLayout(self)
        outer.setContentsMargins(0, 0, 0, 0)
        area, self._lay = scroll_page()
        outer.addWidget(area)
        page_header(self._lay, "Recomendaciones",
                    "Basadas en tu hardware y en el uso actual. Ninguna se aplica sola.")
        bar = QHBoxLayout()
        self.tier_label = QLabel("Analizando…")
        self.tier_label.setObjectName("Badge")
        bar.addWidget(self.tier_label)
        bar.addStretch(1)
        self.apply_btn = QPushButton("Revisar ajustes automáticos")
        self.apply_btn.setObjectName("Primary")
        self.apply_btn.setEnabled(False)
        self.apply_btn.clicked.connect(self.review_requested.emit)
        bar.addWidget(self.apply_btn)
        self._lay.addLayout(bar)
        self._list = QVBoxLayout()
        self._list.setSpacing(10)
        self._lay.addLayout(self._list)
        self._lay.addStretch(1)
        self.recommendations: list[Recommendation] = []

    def set_recommendations(self, recs: list[Recommendation], tier: str) -> None:
        self.recommendations = recs
        self.tier_label.setText(f"Equipo de gama {tier}")
        while self._list.count():
            w = self._list.takeAt(0).widget()
            if w is not None:
                w.deleteLater()
        for r in recs:
            frame, lay = card()
            label, color = PRIORITY.get(r.priority, ("—", theme.MUTED))
            kind = "Ajuste automático (con tu permiso)" if r.automatic else "Acción manual sugerida"
            head = QLabel(f"<span style='color:{color}; font-size:11px; letter-spacing:1px'>"
                          f"PRIORIDAD {label}</span>  <span style='color:{theme.MUTED}; font-size:11px'>"
                          f"· {kind}</span>")
            title = QLabel(f"<b>{r.title}</b>")
            title.setStyleSheet("font-size: 15px;")
            detail = QLabel(r.detail)
            detail.setWordWrap(True)
            impact = QLabel(r.impact_note)
            impact.setObjectName("Muted")
            impact.setWordWrap(True)
            impact.setStyleSheet("font-size: 11px; font-style: italic;")
            for w in (head, title, detail, impact):
                lay.addWidget(w)
            self._list.addWidget(frame)
        self.apply_btn.setEnabled(any(r.automatic for r in recs))
