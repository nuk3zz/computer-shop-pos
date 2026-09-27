#ifndef MyAppVersion
  #define MyAppVersion "dev"
#endif

#define MyAppName "Computer Shop POS"
#define MyAppPublisher "nuk3zz"
#define MyAppURL "https://github.com/nuk3zz/computer-shop-pos"
#define MyAppExeName "computer-shop-pos.exe"

[Setup]
AppId={{8F6CBEA4-731D-482B-BBDC-F4D8EE76BF45}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf64}\Computer Shop POS
DefaultGroupName={#MyAppName}
AllowNoIcons=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
OutputDir=output
OutputBaseFilename=Computer-Shop-POS-{#MyAppVersion}-Windows-x64-Setup
UninstallDisplayIcon={app}\{#MyAppExeName}

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"
Name: "lanfirewall"; Description: "Allow access from devices on the same private Wi-Fi/LAN"; GroupDescription: "Network access:"; Flags: checkedonce

[Files]
Source: "build\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Open Computer Shop POS"; Filename: "http://localhost:3000"
Name: "{group}\Start Computer Shop POS Server"; Filename: "{app}\{#MyAppExeName}"; Parameters: "--native --data-dir=""{userdocs}\Computer Shop POS"" --port=3000"
Name: "{group}\Uninstall Computer Shop POS"; Filename: "{uninstallexe}"
Name: "{commondesktop}\Computer Shop POS"; Filename: "http://localhost:3000"; Tasks: desktopicon
Name: "{commonstartup}\Computer Shop POS Server"; Filename: "{app}\{#MyAppExeName}"; Parameters: "--native --data-dir=""{userdocs}\Computer Shop POS"" --port=3000"; WorkingDir: "{app}"

[Run]
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""Computer Shop POS (Private LAN)"""; Flags: runhidden; Tasks: lanfirewall
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall add rule name=""Computer Shop POS (Private LAN)"" dir=in action=allow program=""{app}\{#MyAppExeName}"" enable=yes profile=private protocol=TCP localport=3000"; Flags: runhidden; Tasks: lanfirewall
Filename: "{app}\{#MyAppExeName}"; Parameters: "--native --data-dir=""{userdocs}\Computer Shop POS"" --port=3000"; WorkingDir: "{app}"; Flags: nowait postinstall skipifsilent; Description: "Start Computer Shop POS"
Filename: "http://localhost:3000/setup"; Flags: shellexec nowait postinstall skipifsilent; Description: "Open first-time setup"

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/IM {#MyAppExeName} /T /F"; Flags: runhidden; RunOnceId: "StopServer"
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""Computer Shop POS (Private LAN)"""; Flags: runhidden; RunOnceId: "RemoveFirewallRule"

[Code]
function InitializeSetup(): Boolean;
var
  ResultCode: Integer;
begin
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/IM {#MyAppExeName} /T /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := True;
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    ForceDirectories(ExpandConstant('{userdocs}\Computer Shop POS'));
end;
