; ==============================================================================
; OpenSourceBackup — Lokaler Windows-Installer (NSIS EXE)
; Installiert: Control Plane + Agent + Docker-Compose-Stack
; Alles auf einem Rechner — für Heimanwender und kleine Teams
;
; Build:
;   makensis build\windows\local-installer.nsi
;   → dist\windows\OpenSourceBackup-Setup.exe
;
; Silent-Install:
;   OpenSourceBackup-Setup.exe /S /PORT=8080 /RESTICREPO=C:\Backups
; ==============================================================================

Unicode True

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "nsDialogs.nsh"
!include "WinMessages.nsh"
!include "FileFunc.nsh"
!include "Sections.nsh"

; ── Metadata ──────────────────────────────────────────────────────────────────

Name "OpenSourceBackup"
OutFile "..\..\dist\windows\OpenSourceBackup-Setup.exe"
InstallDir "$PROGRAMDATA\opensourcebackup"
InstallDirRegKey HKLM "Software\OpenSourceBackup" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
BrandingText "Creating backups is easy. Proving recoverability is the difference."

; ── Variables ─────────────────────────────────────────────────────────────────

Var Port
Var ResticRepo
Var ResticPass
Var ResticPassConfirm
Var DbPassword

; GUI-Controls
Var DlgPort
Var DlgRepo
Var DlgPass
Var DlgPassConfirm
Var DlgBrowse

; ── MUI Konfiguration ────────────────────────────────────────────────────────

!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "Willkommen bei OpenSourceBackup"
!define MUI_WELCOMEPAGE_TEXT "OpenSourceBackup installiert folgendes auf Ihrem Computer:$\r$\n$\r$\n  • Control Plane (Web-Dashboard)$\r$\n  • Backup-Agent$\r$\n  • PostgreSQL-Datenbank (via Docker)$\r$\n  • Redis (via Docker)$\r$\n$\r$\nAlle Dienste starten automatisch beim Windows-Boot.$\r$\n$\r$\nVoraussetzung: Docker Desktop muss installiert sein.$\r$\n$\r$\nKlicken Sie auf Weiter um fortzufahren."
!define MUI_FINISHPAGE_RUN_TEXT "Dashboard jetzt öffnen"
!define MUI_FINISHPAGE_LINK "github.com/cerberus8484/opensourcebackup"
!define MUI_FINISHPAGE_LINK_LOCATION "https://github.com/cerberus8484/opensourcebackup"

!insertmacro MUI_PAGE_WELCOME
Page custom RequirementsPage ""
Page custom ConfigPage ConfigPageLeave
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
Page custom FinishPage ""

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "German"

; ── Requirements Check Page ───────────────────────────────────────────────────

Function RequirementsPage
  !insertmacro MUI_HEADER_TEXT "Voraussetzungen" "Docker Desktop wird geprüft."

  nsDialogs::Create 1018
  Pop $0

  ; Docker prüfen
  nsExec::ExecToStack 'docker version --format "{{.Server.Version}}"'
  Pop $0  ; exit code
  Pop $1  ; output

  ${If} $0 == 0
    ${NSD_CreateLabel} 0 0 100% 24u "✓  Docker Desktop läuft (Version $1)"
    Pop $0
    SetCtlColors $0 "00aa44" transparent
  ${Else}
    ${NSD_CreateLabel} 0 0 100% 48u "✗  Docker Desktop ist nicht gestartet oder nicht installiert.$\r$\n$\r$\nBitte Docker Desktop installieren und starten, dann Setup erneut ausführen.$\r$\nhttps://www.docker.com/products/docker-desktop/"
    Pop $0
    SetCtlColors $0 "cc2222" transparent

    ${NSD_CreateButton} 0 56u 140u 16u "Docker Desktop herunterladen"
    Pop $0
    ${NSD_OnClick} $0 OpenDockerDownload
  ${EndIf}

  nsDialogs::Show
FunctionEnd

Function OpenDockerDownload
  ExecShell "open" "https://www.docker.com/products/docker-desktop/"
FunctionEnd

; ── Config Page ───────────────────────────────────────────────────────────────

Function ConfigPage
  !insertmacro MUI_HEADER_TEXT "Konfiguration" "Backup-Ziel und Passwort festlegen."

  nsDialogs::Create 1018
  Pop $0

  ; Port
  ${NSD_CreateLabel}  0   0  100% 10u "Web-Dashboard Port:"
  Pop $0
  ${NSD_CreateNumber} 0  12u  60u 14u "$Port"
  Pop $DlgPort

  ; Backup-Ziel
  ${NSD_CreateLabel}  0  34u 100% 10u "Backup-Ziel: *"
  Pop $0
  ${NSD_CreateText}   0  46u  74% 14u "$ResticRepo"
  Pop $DlgRepo
  ${NSD_CreateBrowseButton} 76% 46u 24% 14u "Durchsuchen..."
  Pop $DlgBrowse
  ${NSD_OnClick} $DlgBrowse BrowseForFolder

  ; Passwort
  ${NSD_CreateLabel}    0  68u 100% 10u "Backup-Verschlüsselungspasswort: *"
  Pop $0
  ${NSD_CreatePassword} 0  80u 100% 14u ""
  Pop $DlgPass

  ${NSD_CreateLabel}    0 102u 100% 10u "Passwort wiederholen: *"
  Pop $0
  ${NSD_CreatePassword} 0 114u 100% 14u ""
  Pop $DlgPassConfirm

  ; Hinweis
  ${NSD_CreateLabel} 0 134u 100% 22u "⚠ Das Passwort verschlüsselt alle Ihre Backups. Ohne dieses Passwort können keine Daten wiederhergestellt werden. Bitte sicher aufbewahren!"
  Pop $0
  SetCtlColors $0 "aa8800" transparent

  nsDialogs::Show
FunctionEnd

Function BrowseForFolder
  nsDialogs::SelectFolderDialog "Backup-Ziel wählen" "$DOCUMENTS"
  Pop $0
  ${If} $0 != error
    ${NSD_SetText} $DlgRepo $0
  ${EndIf}
FunctionEnd

Function ConfigPageLeave
  ${NSD_GetText} $DlgPort        $Port
  ${NSD_GetText} $DlgRepo        $ResticRepo
  ${NSD_GetText} $DlgPass        $ResticPass
  ${NSD_GetText} $DlgPassConfirm $ResticPassConfirm

  ${If} $Port == ""
    StrCpy $Port "8080"
  ${EndIf}

  ${If} $ResticRepo == ""
    MessageBox MB_OK|MB_ICONEXCLAMATION "Bitte ein Backup-Ziel angeben!"
    Abort
  ${EndIf}

  ${If} $ResticPass == ""
    MessageBox MB_OK|MB_ICONEXCLAMATION "Bitte ein Backup-Passwort eingeben!"
    Abort
  ${EndIf}

  ${If} $ResticPass != $ResticPassConfirm
    MessageBox MB_OK|MB_ICONEXCLAMATION "Die Passwörter stimmen nicht überein!"
    Abort
  ${EndIf}

  StrLen $0 $ResticPass
  ${If} $0 < 8
    MessageBox MB_OK|MB_ICONEXCLAMATION "Das Passwort muss mindestens 8 Zeichen lang sein!"
    Abort
  ${EndIf}
FunctionEnd

; ── Installer Init ─────────────────────────────────────────────────────────────

Function .onInit
  StrCpy $Port      "8080"
  StrCpy $ResticRepo "$DOCUMENTS\OpenSourceBackup-Backups"

  ; Kommandozeilen-Overrides für Silent-Install
  ${GetOptions} $CMDLINE "/PORT="   $0
  ${If} $0 != ""
    StrCpy $Port $0
  ${EndIf}
  ${GetOptions} $CMDLINE "/RESTICREPO=" $0
  ${If} $0 != ""
    StrCpy $ResticRepo $0
  ${EndIf}
  ${GetOptions} $CMDLINE "/PASSWORD=" $0
  ${If} $0 != ""
    StrCpy $ResticPass $0
    StrCpy $ResticPassConfirm $0
  ${EndIf}
FunctionEnd

; ── Install Section ────────────────────────────────────────────────────────────

Section "OpenSourceBackup" SecMain
  SectionIn RO

  SetOutPath "$INSTDIR"
  SetOverwrite on

  ; Verzeichnisse
  CreateDirectory "$INSTDIR\server"
  CreateDirectory "$INSTDIR\agent"
  CreateDirectory "$INSTDIR\data\postgres"
  CreateDirectory "$INSTDIR\data\redis"
  CreateDirectory "$INSTDIR\restore-tests"

  ; Binaries
  File /oname=server\opensourcebackup-server.exe "..\..\dist\server\v0.1.0\opensourcebackup-server-windows-amd64.exe"
  File /oname=agent\opensourcebackup-agent.exe   "..\..\dist\agent\v0.1.0\opensourcebackup-agent-windows-amd64.exe"
  File /oname=agent\restic.exe                   "..\..\dist\tools\restic.exe"

  ; Docker Compose schreiben
  FileOpen $0 "$INSTDIR\docker-compose.yml" w
  FileWrite $0 'services:$\r$\n'
  FileWrite $0 '  postgres:$\r$\n'
  FileWrite $0 '    image: postgres:16-alpine$\r$\n'
  FileWrite $0 '    restart: always$\r$\n'
  FileWrite $0 '    environment:$\r$\n'
  FileWrite $0 '      POSTGRES_USER: opensourcebackup$\r$\n'
  FileWrite $0 '      POSTGRES_PASSWORD: "$DbPassword"$\r$\n'
  FileWrite $0 '      POSTGRES_DB: opensourcebackup$\r$\n'
  FileWrite $0 '    volumes:$\r$\n'
  FileWrite $0 '      - $INSTDIR\data\postgres:/var/lib/postgresql/data$\r$\n'
  FileWrite $0 '    ports:$\r$\n'
  FileWrite $0 '      - "127.0.0.1:5432:5432"$\r$\n'
  FileWrite $0 '  redis:$\r$\n'
  FileWrite $0 '    image: redis:7-alpine$\r$\n'
  FileWrite $0 '    restart: always$\r$\n'
  FileWrite $0 '    ports:$\r$\n'
  FileWrite $0 '      - "127.0.0.1:6379:6379"$\r$\n'
  FileClose $0

  ; Alles per PowerShell-Script aufsetzen (Dienste, DB, Enrollment)
  DetailPrint "Starte PostgreSQL + Redis..."
  nsExec::ExecToLog 'docker compose -f "$INSTDIR\docker-compose.yml" up -d'

  DetailPrint "Warte auf PostgreSQL (30s)..."
  Sleep 15000
  nsExec::ExecToLog 'docker exec opensourcebackup-postgres-1 pg_isready -U opensourcebackup'

  ; Env-Variablen setzen
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "DATABASE_URL"    "postgres://opensourcebackup:$DbPassword@127.0.0.1:5432/opensourcebackup?sslmode=disable"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "LISTEN_ADDR"     ":$Port"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CORS_ORIGIN"     "*"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTIC_PASSWORD" "$ResticPass"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTIC_REPO"     "$ResticRepo"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTIC_BIN"      "$INSTDIR\agent\restic.exe"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "AGENT_TOKEN_FILE" "$INSTDIR\agent\agent-token"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTORE_TEST_ROOT" "$INSTDIR\restore-tests"

  ; Control Plane Dienst
  DetailPrint "Registriere Control Plane Dienst..."
  nsExec::ExecToLog '"$INSTDIR\server\opensourcebackup-server.exe" install'
  nsExec::ExecToLog '"$INSTDIR\server\opensourcebackup-server.exe" start'
  Sleep 5000

  ; Agent Dienst (ohne Enrollment-Token — wird später im Dashboard gemacht)
  DetailPrint "Registriere Agent Dienst..."
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CONTROL_PLANE_URL" "http://127.0.0.1:$Port"
  nsExec::ExecToLog '"$INSTDIR\agent\opensourcebackup-agent.exe" install'

  ; Add/Remove Programs Eintrag
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup" "DisplayName"     "OpenSourceBackup"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup" "DisplayVersion"  "0.1.0"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup" "Publisher"       "OpenSourceBackup"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup" "URLInfoAbout"    "https://github.com/cerberus8484/opensourcebackup"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup" "DisplayIcon"     "$INSTDIR\server\opensourcebackup-server.exe"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup" "NoModify" 1
  WriteRegStr HKLM "Software\OpenSourceBackup" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\OpenSourceBackup" "Port"       "$Port"

  WriteUninstaller "$INSTDIR\uninstall.exe"

  DetailPrint ""
  DetailPrint "✓ OpenSourceBackup installiert!"
  DetailPrint "  Dashboard: http://localhost:$Port/ui/"

SectionEnd

; ── Finish Page ───────────────────────────────────────────────────────────────

Function FinishPage
  !insertmacro MUI_HEADER_TEXT "Installation abgeschlossen" "OpenSourceBackup läuft als Windows-Dienst."

  nsDialogs::Create 1018
  Pop $0

  ${NSD_CreateLabel} 0 0 100% 14u "✓  OpenSourceBackup wurde erfolgreich installiert!"
  Pop $0
  SetCtlColors $0 "00aa44" transparent

  ${NSD_CreateLabel} 0 20u 100% 10u "Web-Dashboard:"
  Pop $0
  ${NSD_CreateLink}  0 32u 100% 14u "http://localhost:$Port/ui/"
  Pop $0
  ${NSD_OnClick} $0 OpenDashboard

  ${NSD_CreateLabel} 0 54u 100% 40u "Dienste laufen im Hintergrund und starten automatisch mit Windows.$\r$\nDiensteverwaltung: Windows-Taste → services.msc$\r$\n$\r$\nNächster Schritt: Im Dashboard unter 'Agents' den Agent enrollen,$\r$\ndann eine Policy und einen Job anlegen."
  Pop $0

  nsDialogs::Show
FunctionEnd

Function OpenDashboard
  ExecShell "open" "http://localhost:$Port/ui/"
FunctionEnd

; ── Uninstall ─────────────────────────────────────────────────────────────────

Section "Uninstall"

  nsExec::ExecToLog '"$INSTDIR\server\opensourcebackup-server.exe" stop'
  nsExec::ExecToLog '"$INSTDIR\server\opensourcebackup-server.exe" uninstall'
  nsExec::ExecToLog '"$INSTDIR\agent\opensourcebackup-agent.exe" stop'
  nsExec::ExecToLog '"$INSTDIR\agent\opensourcebackup-agent.exe" uninstall'
  nsExec::ExecToLog 'docker compose -f "$INSTDIR\docker-compose.yml" down'

  ; Env-Variablen entfernen
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "DATABASE_URL"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "LISTEN_ADDR"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CONTROL_PLANE_URL"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTIC_BIN"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "AGENT_TOKEN_FILE"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTORE_TEST_ROOT"

  MessageBox MB_YESNO "Backup-Daten und Datenbank in $INSTDIR\data löschen?" IDYES DeleteData IDNO KeepData
  DeleteData:
    RMDir /r "$INSTDIR\data"
  KeepData:

  Delete "$INSTDIR\server\opensourcebackup-server.exe"
  Delete "$INSTDIR\agent\opensourcebackup-agent.exe"
  Delete "$INSTDIR\agent\restic.exe"
  Delete "$INSTDIR\docker-compose.yml"
  Delete "$INSTDIR\uninstall.exe"

  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackup"
  DeleteRegKey HKLM "Software\OpenSourceBackup"

  MessageBox MB_OK "OpenSourceBackup wurde deinstalliert.$\r$\nDaten unter $INSTDIR wurden nach Ihrer Wahl behandelt."

SectionEnd
