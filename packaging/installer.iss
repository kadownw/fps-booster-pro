; Inno Setup — instalador de FPS BOOSTER (requiere haber corrido PyInstaller antes)
#define AppName "FPS BOOSTER"
#define AppVersion "0.1.0"

[Setup]
AppId={{6E2C7B1A-4F2D-4C5B-9A3E-FB00570A0001}
AppName={#AppName}
AppVersion={#AppVersion}
DefaultDirName={autopf}\FPSBooster
DefaultGroupName={#AppName}
OutputDir=..\dist\installer
OutputBaseFilename=FPSBooster-Setup-{#AppVersion}
Compression=lzma2
SolidCompression=yes
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.10240
PrivilegesRequired=lowest
WizardStyle=modern

[Files]
Source: "..\dist\FPSBooster\*"; DestDir: "{app}"; Flags: recursesubdirs ignoreversion

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\FPSBooster.exe"
Name: "{group}\{#AppName} (modo real)"; Filename: "{app}\FPSBooster.exe"; Parameters: "--modo-real"
Name: "{userdesktop}\{#AppName}"; Filename: "{app}\FPSBooster.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Crear acceso directo en el escritorio"; Flags: unchecked

; Nota: desinstalar NO revierte ajustes automáticamente. Usar "Historial > Revertir todo" antes.
