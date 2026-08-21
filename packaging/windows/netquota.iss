#ifndef AppVersion
#define AppVersion "0.1.0"
#endif

#ifndef AppBinaryVersion
#define AppBinaryVersion "0.1.0"
#endif

#ifndef SourceDir
#define SourceDir "release\netquota-windows-amd64"
#endif

[Setup]
AppId={{9B5A5C67-54F6-4B43-9E1A-9F3E8E4A2D31}
AppName=NetQuota
AppVersion={#AppVersion}
AppVerName=NetQuota {#AppVersion}
AppPublisher=KageRyo
AppPublisherURL=https://github.com/KageRyo/netquota
AppSupportURL=https://github.com/KageRyo/netquota/issues
AppUpdatesURL=https://github.com/KageRyo/netquota/releases/latest
LicenseFile="{#SourceDir}\LICENSE.txt"
SetupIconFile="{#SourceDir}\icon.ico"
DefaultDirName={localappdata}\Programs\NetQuota
DefaultGroupName=NetQuota
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=commandline
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon="{app}\icon.ico"
OutputBaseFilename=netquota-windows-amd64-setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
ChangesAssociations=no
VersionInfoCompany=KageRyo
VersionInfoDescription=NetQuota Windows installer
VersionInfoProductName=NetQuota
VersionInfoVersion={#AppBinaryVersion}
VersionInfoProductVersion={#AppBinaryVersion}
VersionInfoTextVersion={#AppVersion}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional icons:"; Flags: unchecked

[Files]
Source: "{#SourceDir}\netquota.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\netquota-console.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\icon.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\README.md"; DestDir: "{app}"; Flags: isreadme
Source: "{#SourceDir}\PRIVACY.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceDir}\LICENSE.txt"; DestDir: "{app}"; DestName: "LICENSE"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\NetQuota"; Filename: "{app}\netquota.exe"; WorkingDir: "{app}"; IconFilename: "{app}\icon.ico"; IconIndex: 0
Name: "{autodesktop}\NetQuota"; Filename: "{app}\netquota.exe"; WorkingDir: "{app}"; IconFilename: "{app}\icon.ico"; IconIndex: 0; Tasks: desktopicon

[Run]
Filename: "{app}\netquota.exe"; Description: "Launch NetQuota"; Flags: nowait postinstall skipifsilent
