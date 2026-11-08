# PyInstaller spec — genera dist\FPSBooster\FPSBooster.exe
# Uso:  pyinstaller packaging\fps_booster.spec --noconfirm
from pathlib import Path

ROOT = Path(SPECPATH).parent

a = Analysis(
    [str(ROOT / "src" / "fps_booster" / "__main__.py")],
    pathex=[str(ROOT / "src")],
    binaries=[],
    datas=[],
    hiddenimports=[],
    excludes=["tkinter", "PySide6.QtWebEngineCore", "PySide6.Qt3DCore", "PySide6.QtQuick"],
    noarchive=False,
)
pyz = PYZ(a.pure)
exe = EXE(
    pyz,
    a.scripts,
    [],
    exclude_binaries=True,
    name="FPSBooster",
    console=False,
    uac_admin=False,  # fase 1: no necesita administrador
)
coll = COLLECT(exe, a.binaries, a.datas, name="FPSBooster")
