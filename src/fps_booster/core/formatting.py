"""Helpers de formato para la UI."""

from __future__ import annotations


def fmt_bytes(value: float | None, precision: int = 1) -> str:
    if value is None:
        return "N/D"
    units = ["B", "KB", "MB", "GB", "TB"]
    size = float(value)
    for unit in units:
        if abs(size) < 1024 or unit == units[-1]:
            return f"{size:.0f} {unit}" if unit == "B" else f"{size:.{precision}f} {unit}"
        size /= 1024
    return f"{size:.{precision}f} TB"


def fmt_rate(bps: float | None) -> str:
    return "N/D" if bps is None else f"{fmt_bytes(bps)}/s"


def fmt_percent(value: float | None) -> str:
    return "N/D" if value is None else f"{value:.0f}%"
