"""Tarjeta de métrica: título, valor grande, subtítulo y sparkline."""

from __future__ import annotations

from PySide6.QtWidgets import QFrame, QLabel, QVBoxLayout, QWidget

from fps_booster.ui.widgets.line_chart import LiveChart


class StatCard(QFrame):
    def __init__(self, title: str, color: str, parent: QWidget | None = None):
        super().__init__(parent)
        self.setObjectName("Card")
        self.setMinimumHeight(150)
        layout = QVBoxLayout(self)
        layout.setContentsMargins(18, 14, 18, 10)
        layout.setSpacing(2)
        self.title_label = QLabel(title.upper())
        self.title_label.setObjectName("CardTitle")
        self.value_label = QLabel("—")
        self.value_label.setObjectName("CardValue")
        self.value_label.setStyleSheet(f"color: {color};")
        self.sub_label = QLabel("")
        self.sub_label.setObjectName("CardSub")
        self.sub_label.setWordWrap(True)
        self.chart = LiveChart(compact=True, max_points=40)
        self.chart.add_series("v", color)
        layout.addWidget(self.title_label)
        layout.addWidget(self.value_label)
        layout.addWidget(self.sub_label)
        layout.addWidget(self.chart, 1)

    def set_value(self, text: str, sub: str = "", chart_value: float | None = None) -> None:
        self.value_label.setText(text)
        self.sub_label.setText(sub)
        self.chart.add_value("v", chart_value)
