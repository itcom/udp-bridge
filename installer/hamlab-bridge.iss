#define AppName "HAMLAB Bridge"
#define AppVersion "0.1.0"
#define AppExeName "hamlab-bridge.exe"
#define AppPublisher "itcom"
#define AppURL "https://github.com/itcom/udp-bridge"

[Setup]
AppId={{E5E6D1E3-6C89-4D7C-B8E1-0A2E7C8A0001}}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
DefaultDirName={autopf}\HAMLAB Bridge
DefaultGroupName=HAMLAB Bridge
OutputDir=dist
Compression=lzma
SolidCompression=yes
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64os
OutputBaseFilename=hamlab-bridge-Setup-x64

[Files]
Source: "..\hamlab-bridge.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.txt"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\HAMLAB Bridge"; Filename: "{app}\{#AppExeName}"
Name: "{group}\Uninstall HAMLAB Bridge"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\{#AppExeName}"; \
  Description: "HAMLAB Bridge を起動"; \
  Flags: nowait postinstall skipifsilent

