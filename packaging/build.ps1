# Genera el ejecutable y (si Inno Setup está instalado) el instalador.
$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
Set-Location $root
.\.venv\Scripts\python.exe -m pytest -q
.\.venv\Scripts\python.exe -m PyInstaller packaging\fps_booster.spec --noconfirm --distpath dist --workpath build
$iscc = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe"
if (Test-Path $iscc) { & $iscc packaging\installer.iss } else { Write-Host "Inno Setup no encontrado: solo se generó dist\FPSBooster" }
