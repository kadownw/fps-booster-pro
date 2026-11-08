"""Tema oscuro futurista con acentos azules."""

from __future__ import annotations

BG = "#060a13"
SURFACE = "#0c1322"
SURFACE_2 = "#111b2f"
BORDER = "#1b2a47"
ACCENT = "#1e8fff"
ACCENT_2 = "#00d8ff"
ACCENT_DIM = "#0f3d73"
TEXT = "#e7f0ff"
MUTED = "#7d8bad"
OK = "#2ee6a6"
WARN = "#ffb547"
DANGER = "#ff4d6d"
PURPLE = "#8b7bff"

FONT = "Segoe UI"
FONT_DISPLAY = "Bahnschrift"

QSS = f"""
* {{ font-family: "{FONT}"; color: {TEXT}; }}
QMainWindow, QDialog {{ background: {BG}; }}
QWidget#Root {{ background: {BG}; }}
QToolTip {{ background: {SURFACE_2}; color: {TEXT}; border: 1px solid {ACCENT}; padding: 6px; }}

QFrame#Sidebar {{ background: {SURFACE}; border-right: 1px solid {BORDER}; }}
QLabel#Logo {{ font-family: "{FONT_DISPLAY}"; font-size: 22px; font-weight: 700;
               color: {TEXT}; letter-spacing: 3px; }}
QLabel#LogoAccent {{ font-family: "{FONT_DISPLAY}"; font-size: 11px; color: {ACCENT_2};
                     letter-spacing: 5px; }}
QPushButton#NavButton {{ background: transparent; border: none; border-left: 3px solid transparent;
    text-align: left; padding: 12px 18px; font-size: 14px; color: {MUTED}; }}
QPushButton#NavButton:hover {{ color: {TEXT}; background: {SURFACE_2}; }}
QPushButton#NavButton:checked {{ color: {TEXT}; border-left: 3px solid {ACCENT_2};
    background: qlineargradient(x1:0,y1:0,x2:1,y2:0, stop:0 {ACCENT_DIM}, stop:1 transparent); }}

QLabel#PageTitle {{ font-family: "{FONT_DISPLAY}"; font-size: 24px; font-weight: 600; }}
QLabel#PageSubtitle {{ color: {MUTED}; font-size: 13px; }}
QLabel#SectionTitle {{ font-family: "{FONT_DISPLAY}"; font-size: 15px; font-weight: 600;
                       color: {ACCENT_2}; letter-spacing: 1px; }}
QLabel#Muted {{ color: {MUTED}; }}
QLabel#ModeSim {{ background: {ACCENT_DIM}; color: {ACCENT_2}; border: 1px solid {ACCENT};
    border-radius: 11px; padding: 4px 12px; font-weight: 600; font-size: 11px; letter-spacing: 1px; }}
QLabel#ModeReal {{ background: #3d1220; color: {DANGER}; border: 1px solid {DANGER};
    border-radius: 11px; padding: 4px 12px; font-weight: 600; font-size: 11px; letter-spacing: 1px; }}
QLabel#Badge {{ background: {SURFACE_2}; border: 1px solid {BORDER}; border-radius: 9px;
    padding: 2px 10px; font-size: 11px; color: {MUTED}; }}

QFrame#Card {{ background: {SURFACE}; border: 1px solid {BORDER}; border-radius: 14px; }}
QFrame#Card:hover {{ border: 1px solid {ACCENT_DIM}; }}
QFrame#CardAccent {{ background: qlineargradient(x1:0,y1:0,x2:1,y2:1, stop:0 {SURFACE_2}, stop:1 {SURFACE});
    border: 1px solid {ACCENT}; border-radius: 14px; }}
QLabel#CardTitle {{ color: {MUTED}; font-size: 12px; letter-spacing: 2px; font-weight: 600; }}
QLabel#CardValue {{ font-family: "{FONT_DISPLAY}"; font-size: 30px; font-weight: 600; }}
QLabel#CardSub {{ color: {MUTED}; font-size: 11px; }}

QPushButton {{ background: {SURFACE_2}; border: 1px solid {BORDER}; border-radius: 8px;
    padding: 8px 16px; font-size: 13px; }}
QPushButton:hover {{ border: 1px solid {ACCENT}; }}
QPushButton:disabled {{ color: #4a5675; border-color: {SURFACE_2}; }}
QPushButton#Primary {{ background: qlineargradient(x1:0,y1:0,x2:1,y2:0, stop:0 {ACCENT}, stop:1 {ACCENT_2});
    border: none; color: #031024; font-weight: 700; }}
QPushButton#Primary:hover {{ background: {ACCENT_2}; }}
QPushButton#Primary:disabled {{ background: {SURFACE_2}; color: #4a5675; }}
QPushButton#Danger {{ border: 1px solid {DANGER}; color: {DANGER}; background: transparent; }}
QPushButton#Danger:hover {{ background: #3d1220; }}

QComboBox {{ background: {SURFACE_2}; border: 1px solid {BORDER}; border-radius: 8px; padding: 6px 12px; }}
QComboBox:hover {{ border-color: {ACCENT}; }}
QComboBox QAbstractItemView {{ background: {SURFACE_2}; border: 1px solid {ACCENT};
    selection-background-color: {ACCENT_DIM}; }}

QCheckBox {{ spacing: 8px; }}
QCheckBox::indicator {{ width: 18px; height: 18px; border-radius: 5px; border: 1px solid {MUTED};
    background: {SURFACE}; }}
QCheckBox::indicator:checked {{ background: {ACCENT}; border-color: {ACCENT_2}; }}
QCheckBox::indicator:disabled {{ border-color: {BORDER}; background: {SURFACE_2}; }}

QTableWidget {{ background: {SURFACE}; border: 1px solid {BORDER}; border-radius: 10px;
    gridline-color: {BORDER}; alternate-background-color: #0e1728; selection-background-color: {ACCENT_DIM}; }}
QHeaderView::section {{ background: {SURFACE_2}; color: {MUTED}; border: none;
    border-bottom: 1px solid {BORDER}; padding: 8px; font-weight: 600; font-size: 12px; }}
QTableCornerButton::section {{ background: {SURFACE_2}; border: none; }}

QScrollArea {{ border: none; background: transparent; }}
QScrollBar:vertical {{ background: transparent; width: 10px; margin: 2px; }}
QScrollBar::handle:vertical {{ background: {BORDER}; border-radius: 4px; min-height: 30px; }}
QScrollBar::handle:vertical:hover {{ background: {ACCENT_DIM}; }}
QScrollBar::add-line, QScrollBar::sub-line {{ height: 0; }}
QScrollBar:horizontal {{ height: 0; }}

QMessageBox {{ background: {SURFACE}; }}
QMessageBox QLabel {{ color: {TEXT}; }}
"""
