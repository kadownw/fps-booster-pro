"""Botón central circular con anillo animado."""

from __future__ import annotations

import math

from PySide6.QtCore import QPointF, QRectF, Qt, QTimer
from PySide6.QtGui import QColor, QConicalGradient, QFont, QPainter, QPen, QRadialGradient
from PySide6.QtWidgets import QAbstractButton, QWidget

from fps_booster.ui import theme


class OptimizeButton(QAbstractButton):
    def __init__(self, parent: QWidget | None = None, size: int = 230):
        super().__init__(parent)
        self.setFixedSize(size, size)
        self.setCursor(Qt.CursorShape.PointingHandCursor)
        self.setText("OPTIMIZAR")
        self.subtitle = "Perfil Gaming"
        self._angle = 0.0
        self._hover = False
        self._timer = QTimer(self)
        self._timer.timeout.connect(self._tick)
        self._timer.start(40)  # ~25 fps: animación suave con consumo de CPU mínimo

    def set_subtitle(self, text: str) -> None:
        self.subtitle = text
        self.update()

    def _tick(self) -> None:
        if not self.isVisible():
            return
        self._angle = (self._angle + (2.4 if self._hover else 1.0)) % 360
        self.update()

    def enterEvent(self, event) -> None:  # noqa: N802
        self._hover = True
        super().enterEvent(event)

    def leaveEvent(self, event) -> None:  # noqa: N802
        self._hover = False
        super().leaveEvent(event)

    def paintEvent(self, _event) -> None:  # noqa: N802
        p = QPainter(self)
        p.setRenderHint(QPainter.RenderHint.Antialiasing)
        r = QRectF(self.rect())
        c = r.center()
        radius = min(r.width(), r.height()) / 2
        enabled = self.isEnabled()
        accent = QColor(theme.ACCENT if enabled else theme.BORDER)
        accent2 = QColor(theme.ACCENT_2 if enabled else theme.MUTED)

        # Halo exterior
        pulse = 0.5 + 0.5 * math.sin(math.radians(self._angle * 2))
        halo = QRadialGradient(c, radius)
        glow = QColor(accent2)
        glow.setAlpha(int((70 if self._hover else 40) + 30 * pulse) if enabled else 0)
        halo.setColorAt(0.62, glow)
        glow2 = QColor(accent2)
        glow2.setAlpha(0)
        halo.setColorAt(1.0, glow2)
        p.setPen(Qt.PenStyle.NoPen)
        p.setBrush(halo)
        p.drawEllipse(c, radius, radius)

        # Anillo giratorio segmentado
        ring_r = radius * 0.84
        cone = QConicalGradient(c, -self._angle)
        cone.setColorAt(0.0, accent2)
        cone.setColorAt(0.35, QColor(0, 0, 0, 0))
        cone.setColorAt(0.5, accent)
        cone.setColorAt(0.85, QColor(0, 0, 0, 0))
        cone.setColorAt(1.0, accent2)
        pen = QPen(cone, 5)
        pen.setCapStyle(Qt.PenCapStyle.FlatCap)
        p.setPen(pen)
        p.setBrush(Qt.BrushStyle.NoBrush)
        ring = QRectF(c.x() - ring_r, c.y() - ring_r, 2 * ring_r, 2 * ring_r)
        for i in range(12):
            p.drawArc(ring, int((i * 30 + 4) * 16), int(22 * 16))

        # Marcas finas
        p.setPen(QPen(QColor(theme.BORDER), 1))
        inner_tick = radius * 0.74
        for i in range(60):
            a = math.radians(i * 6)
            length = 6 if i % 5 == 0 else 3
            p1 = QPointF(c.x() + math.cos(a) * inner_tick, c.y() + math.sin(a) * inner_tick)
            p2 = QPointF(c.x() + math.cos(a) * (inner_tick - length),
                         c.y() + math.sin(a) * (inner_tick - length))
            p.drawLine(p1, p2)

        # Disco central
        core_r = radius * 0.64
        core = QRadialGradient(QPointF(c.x(), c.y() - core_r * 0.4), core_r * 1.4)
        if self.isDown():
            core.setColorAt(0, QColor(theme.ACCENT_DIM))
            core.setColorAt(1, QColor(theme.SURFACE))
        else:
            core.setColorAt(0, QColor("#16305a") if enabled else QColor(theme.SURFACE_2))
            core.setColorAt(1, QColor(theme.SURFACE))
        p.setPen(QPen(accent, 2))
        p.setBrush(core)
        p.drawEllipse(c, core_r, core_r)

        # Texto
        p.setPen(QColor(theme.TEXT if enabled else theme.MUTED))
        p.setFont(QFont(theme.FONT_DISPLAY, int(radius * 0.13), QFont.Weight.Bold))
        p.drawText(QRectF(r.left(), c.y() - radius * 0.2, r.width(), radius * 0.26),
                   Qt.AlignmentFlag.AlignCenter, self.text())
        p.setPen(accent2)
        p.setFont(QFont(theme.FONT, max(8, int(radius * 0.065))))
        p.drawText(QRectF(r.left(), c.y() + radius * 0.1, r.width(), radius * 0.2),
                   Qt.AlignmentFlag.AlignCenter, self.subtitle)
        p.end()
