; ==============================================================================
; OpenSourceBackup Agent — NSIS Installer (EXE)
; Builds a self-contained Windows installer EXE
;
; Build:
;   makensis agent-installer.nsi
;   → dist/windows/OpenSourceBackup-Agent-Setup.exe
;
; Silent install:
;   OpenSourceBackup-Agent-Setup.exe /S
;     /CONTROL_PLANE_URL=http://192.168.1.10:8080
;     /ENROLLMENT_TOKEN=abc123
;     /RESTIC_PASSWORD=mypassword
;     /RESTIC_REPO=Z:\Backups
; ==============================================================================

Unicode True

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "nsDialogs.nsh"
!include "WinMessages.nsh"

; ── Metadata ──────────────────────────────────────────────────────────────────

Name "OpenSourceBackup Agent"
OutFile "..\..\dist\windows\OpenSourceBackup-Agent-Setup.exe"
InstallDir "$PROGRAMDATA\opensourcebackup"
InstallDirRegKey HKLM "Software\OpenSourceBackup\Agent" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
BrandingText "OpenSourceBackup — Creating backups is easy. Proving recoverability is the difference."

; ── Variables ─────────────────────────────────────────────────────────────────

Var Dialog
Var LabelURL
Var TextURL
Var LabelToken
Var TextToken
Var LabelPass
Var TextPass
Var LabelRepo
Var TextRepo

Var ControlPlaneURL
Var EnrollmentToken
Var ResticPassword
Var ResticRepo

; Command-line overrides (/KEY=VALUE)
!macro GetArg KEY VAR
  ${GetOptions} $CMDLINE "/${KEY}=" $0
  ${If} $0 != ""
    StrCpy ${VAR} $0
  ${EndIf}
!macroend

; ── MUI Pages ─────────────────────────────────────────────────────────────────

!define MUI_ABORTWARNING
!define MUI_ICON "..\..\.idea\icon.ico"
!define MUI_UNICON "..\..\.idea\icon.ico"
!define MUI_WELCOMEPAGE_TITLE "OpenSourceBackup Agent Setup"
!define MUI_WELCOMEPAGE_TEXT "Dieser Assistent installiert den OpenSourceBackup Agent als Windows-Dienst auf diesem Computer.$\r$\n$\r$\nDer Agent startet automatisch beim Systemstart und verbindet sich mit dem Control Plane.$\r$\n$\r$\nKlicken Sie auf Weiter um fortzufahren."

!insertmacro MUI_PAGE_WELCOME
Page custom ConnectionPage ConnectionPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "German"
!insertmacro MUI_LANGUAGE "English"

; ── Connection Configuration Page ─────────────────────────────────────────────

Function ConnectionPage
  !insertmacro MUI_HEADER_TEXT "Control Plane verbinden" "Geben Sie die Verbindungsdaten ein."

  nsDialogs::Create 1018
  Pop $Dialog
  ${If} $Dialog == error
    Abort
  ${EndIf}

  ; Control Plane URL
  ${NSD_CreateLabel} 0 0 100% 12u "Control Plane URL: *"
  Pop $LabelURL
  ${NSD_CreateText} 0 14u 100% 14u "$ControlPlaneURL"
  Pop $TextURL

  ; Enrollment Token
  ${NSD_CreateLabel} 0 36u 100% 12u "Enrollment Token: *"
  Pop $LabelToken
  ${NSD_CreateText} 0 50u 100% 14u "$EnrollmentToken"
  Pop $TextToken

  ; Backup Password
  ${NSD_CreateLabel} 0 72u 100% 12u "Backup Passwort (Verschlüsselung): *"
  Pop $LabelPass
  ${NSD_CreatePassword} 0 86u 100% 14u ""
  Pop $TextPass

  ; Restic Repo
  ${NSD_CreateLabel} 0 108u 100% 12u "Backup-Ziel (Pfad oder S3-URL):"
  Pop $LabelRepo
  ${NSD_CreateText} 0 122u 100% 14u "$ResticRepo"
  Pop $TextRepo

  nsDialogs::Show
FunctionEnd

Function ConnectionPageLeave
  ${NSD_GetText} $TextURL   $ControlPlaneURL
  ${NSD_GetText} $TextToken $EnrollmentToken
  ${NSD_GetText} $TextPass  $ResticPassword
  ${NSD_GetText} $TextRepo  $ResticRepo

  ${If} $ControlPlaneURL == ""
    MessageBox MB_OK|MB_ICONEXCLAMATION "Bitte Control Plane URL eingeben!"
    Abort
  ${EndIf}
  ${If} $EnrollmentToken == ""
    MessageBox MB_OK|MB_ICONEXCLAMATION "Bitte Enrollment Token eingeben!"
    Abort
  ${EndIf}
  ${If} $ResticPassword == ""
    MessageBox MB_OK|MB_ICONEXCLAMATION "Bitte Backup-Passwort eingeben!"
    Abort
  ${EndIf}
FunctionEnd

; ── Installer Init ─────────────────────────────────────────────────────────────

Function .onInit
  ; Set defaults
  StrCpy $ControlPlaneURL "http://192.168.1.10:8080"
  StrCpy $EnrollmentToken ""
  StrCpy $ResticPassword  ""
  StrCpy $ResticRepo      "C:\ProgramData\opensourcebackup\restic-repo"

  ; Read command-line overrides for silent mode
  !insertmacro GetArg "CONTROL_PLANE_URL" $ControlPlaneURL
  !insertmacro GetArg "ENROLLMENT_TOKEN"  $EnrollmentToken
  !insertmacro GetArg "RESTIC_PASSWORD"   $ResticPassword
  !insertmacro GetArg "RESTIC_REPO"       $ResticRepo

  ; In silent mode skip the GUI
  IfSilent 0 +2
  Goto done
  done:
FunctionEnd

; ── Install Section ────────────────────────────────────────────────────────────

Section "OpenSourceBackup Agent" SecMain
  SectionIn RO  ; always install

  SetOutPath "$INSTDIR"
  SetOverwrite on

  ; Write agent binary
  File /oname=opensourcebackup-agent.exe "..\..\dist\agent\v0.1.0\opensourcebackup-agent-windows-amd64.exe"

  ; Write restic binary
  File /oname=restic.exe "..\..\dist\tools\restic.exe"

  ; Create subdirectories
  CreateDirectory "$INSTDIR\restore-tests"

  ; ── Set Machine environment variables ──
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "CONTROL_PLANE_URL"   "$ControlPlaneURL"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "ENROLLMENT_TOKEN"    "$EnrollmentToken"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "RESTIC_PASSWORD"     "$ResticPassword"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "RESTIC_REPO"         "$ResticRepo"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "RESTIC_BIN"          "$INSTDIR\restic.exe"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "AGENT_TOKEN_FILE"    "$INSTDIR\agent-token"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" \
    "RESTORE_TEST_ROOT"   "$INSTDIR\restore-tests"

  ; ── Register and start Windows Service ──
  DetailPrint "Registriere Windows-Dienst..."
  nsExec::ExecToLog '"$INSTDIR\opensourcebackup-agent.exe" install'
  DetailPrint "Starte Dienst..."
  nsExec::ExecToLog '"$INSTDIR\opensourcebackup-agent.exe" start'

  ; ── Registry for Add/Remove Programs ──
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent" \
    "DisplayName"      "OpenSourceBackup Agent"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent" \
    "UninstallString"  '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent" \
    "DisplayVersion"   "0.1.0"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent" \
    "Publisher"        "OpenSourceBackup"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent" \
    "URLInfoAbout"     "https://github.com/cerberus8484/opensourcebackup"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent" \
    "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent" \
    "NoRepair" 1

  ; Save install dir
  WriteRegStr HKLM "Software\OpenSourceBackup\Agent" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\OpenSourceBackup\Agent" "Version"    "0.1.0"

  ; Write uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  DetailPrint ""
  DetailPrint "✓ OpenSourceBackup Agent erfolgreich installiert!"
  DetailPrint "  Dienst läuft unter: Dienste → OpenSourceBackup Agent"
  DetailPrint "  Verzeichnis: $INSTDIR"

SectionEnd

; ── Uninstall Section ──────────────────────────────────────────────────────────

Section "Uninstall"

  ; Stop and remove service
  nsExec::ExecToLog '"$INSTDIR\opensourcebackup-agent.exe" stop'
  nsExec::ExecToLog '"$INSTDIR\opensourcebackup-agent.exe" uninstall'

  ; Remove env vars (optionally — keep RESTIC_PASSWORD for re-install)
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CONTROL_PLANE_URL"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "ENROLLMENT_TOKEN"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTIC_BIN"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "AGENT_TOKEN_FILE"
  DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "RESTORE_TEST_ROOT"

  ; Remove files (keep agent-token and restic-repo for data safety)
  Delete "$INSTDIR\opensourcebackup-agent.exe"
  Delete "$INSTDIR\restic.exe"
  Delete "$INSTDIR\uninstall.exe"

  ; Remove registry
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\OpenSourceBackupAgent"
  DeleteRegKey HKLM "Software\OpenSourceBackup\Agent"

  ; Note: data directory kept intentionally for backup data safety
  MessageBox MB_OK "OpenSourceBackup Agent wurde deinstalliert.$\r$\nDaten in $INSTDIR wurden behalten."

SectionEnd
