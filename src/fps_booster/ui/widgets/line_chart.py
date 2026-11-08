"""Gráfico de líneas en tiempo real dibujado con QPainter (liviano, sin QtCharts)."""

from __future__ import annotations

from collections import deque

from PySide6.QtCore import QPointF, QRectF, Qt
from PySide6.QtGui import QBrush, QColor, QFont, QLinearGradient, QPainter, QPainterPath, QPen
from PySide6.QtWidgets import QSizePolicy, QWidget

from fps_booster.ui import theme


class LiveChart(QWidget):
    def __init__(self, title: str = "", max_points: int = 60, max_value: float = 100.0,
                 unit: str = "%", compact: bool = False, parent: QWidget | None = None):
        super().__init__(parent)
        self.title = title
        self.max_points = max_points
        self.max_value = max_value
        self.unit = unit
        self.compact = compact
        self._series: dict[str, tuple[QColor, deque]] = {}
        self.setSizePolicy(QSizePolicy.Policy.Expanding, QSizePolicy.Policy.Expanding)
        self.setMinimumHeight(48 if compact else 200)

    def add_series(self, name: str, color: str) -> None:
        self._series[name] = (QColor(color), deque(maxlen=self.max_points))

    def add_value(self, name: str, value: float | None) -> None:
        if name in self._series:
            self._series[name][1].append(value)
            self.update()

    def values(self, name: str) -> list[float | None]:
        return list(self._series[name][1])

    def paintEvent(self, _event) -> None:  # noqa: N802 - API de Qt
        p = QPainter(self)
        p.setRenderHint(QPainter.RenderHint.Antialiasing)
        rect = QRectF(self.rect())
        pad_l, pad_t, pad_r, pad_b = (2, 2, 2, 2) if self.compact else (44, 36, 16, 20)
        plot = rect.adjusted(pad_l, pad_t, -pad_r, -pad_b)

        if not self.compact:
            p.setPen(QPen(QColor(theme.BORDER), 1))
            p.setBrush(QColor(theme.SURFACE))
            p.drawRoundedRect(rect.adjusted(0.5, 0.5, -0.5, -0.5), 14, 14)
            p.setFont(QFont(theme.FONT_DISPLAY, 11, QFont.Weight.DemiBold))
            p.setPen(QColor(theme.ACCENT_2))
            p.drawText(QRectF(16, 8, rect.width() / 2, 24), Qt.AlignmentFlag.AlignVCenter, self.title)
            self._draw_legend(p, rect)
            grid = QPen(QColor(theme.BORDER), 1, Qt.PenStyle.DashLine)
            p.setFont(QFont(theme.FONT, 8))
            for i in range(5):
                y = plot.top() + plot.height() * i / 4
                p.setPen(grid)
                p.drawLine(QPointF(plot.left(), y), QPointF(plot.right(), y))
                p.setPen(QColor(theme.MUTED))
                label = f"{self.max_value * (4 - i) / 4:.0f}{self.unit}"
                p.drawText(QRectF(0, y - 8, pad_l - 6, 16),
                           Qt.AlignmentFlag.AlignRight | Qt.AlignmentFlag.AlignVCenter, label)

        step = plot.width() / max(1, self.max_points - 1)
        for color, data in self._series.values():
            pts = [
                QPointF(plot.right() - (len(data) - 1 - i) * step,
                        plot.bottom() - plot.height() * min(1.0, max(0.0, v / self.max_value)))
                for i, v in enumerate(data) if v is not None
            ]
            if len(pts) < 2:
                continue
            line = QPainterPath(pts[0])
            for pt in pts[1:]:
                line.lineTo(pt)
            fill = QPainterPath(line)
            fill.lineTo(pts[-1].x(), plot.bottom())
            fill.lineTo(pts[0].x(), plot.bottom())
            fill.closeSubpath()
            grad = QLinearGradient(0, plot.top(), 0, plot.bottom())
            top = QColor(color)
            top.setAlpha(110 if self.compact else 90)
            bottom = QColor(color)
            bottom.setAlpha(0)
            grad.setColorAt(0, top)
            grad.setColorAt(1, bottom)
            p.fillPath(fill, QBrush(grad))
            glow = QColor(color)
            glow.setAlpha(60)
            p.setPen(QPen(glow, 5 if not self.compact else 3))
            p.drawPath(line)
            p.setPen(QPen(color, 2 if not self.compact else 1.5))
            p.drawPath(line)
            if not self.compact:
                p.setBrush(color)
                p.drawEllipse(pts[-1], 3.5, 3.5)
        p.end()

    def _draw_legend(self, p: QPainter, rect: QRectF) -> None:
        x = rect.right() - 16
        p.setFont(QFont(theme.FONT, 9))
        for name, (color, data) in reversed(list(self._series.items())):
            last = next((v for v in reversed(data) if v is not None), None)
            text = f"{name} {last:.0f}{self.unit}" if last is not None else f"{name} N/D"
            width = p.fontMetrics().horizontalAdvance(text)
            x -= width
            p.setPen(QColor(theme.TEXT))
            p.drawText(QRectF(x, 8, width, 24), Qt.AlignmentFlag.AlignVCenter, text)
            x -= 14
            p.setPen(Qt.PenStyle.NoPen)
            p.setBrush(color)
            p.drawEllipse(QPointF(x + 5, 20), 4, 4)
            x -= 14
