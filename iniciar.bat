@echo off
rem Abre FPS BOOSTER en modo simulacion, sin importar desde que carpeta se ejecute.
rem Para modo real:  iniciar.bat --modo-real
cd /d "%~dp0"
set PYTHONPATH=%~dp0src
if not exist "%~dp0.venv\Scripts\python.exe" (
    echo No se encontro el entorno .venv. Crealo con:
    echo   py -3.11 -m venv .venv
    echo   .venv\Scripts\python -m pip install -r requirements.txt
    pause
    exit /b 1
)
start "" "%~dp0.venv\Scripts\pythonw.exe" -m fps_booster %*
