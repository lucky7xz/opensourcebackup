# ==============================================================================
# OpenSourceBackup — Lokaler Windows-Installer
# Installiert alles auf EINEM Rechner:
#   Control Plane + PostgreSQL + Redis (Docker) + Agent
#   alles als Windows-Dienste — startet automatisch beim Booten
#
# Voraussetzungen: Docker Desktop muss installiert und gestartet sein
#
# Usage (PowerShell als Administrator):
#   Invoke-WebRequest http://<server>/scripts/install-local.ps1 | iex
#
# Oder lokal:
#   .\install-local.ps1
#   .\install-local.ps1 -Port 8080 -ResticRepo "Z:\Backups"
# ==============================================================================

#Requires -RunAsAdministrator

param(
    [string]$Version      = "v0.1.0",
    [string]$InstallDir   = "C:\ProgramData\opensourcebackup",
    [int]   $Port         = 8080,
    [string]$ResticRepo   = "",
    [string]$ResticPass   = "",
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"

# ── Farben & Ausgabe ──────────────────────────────────────────────────────────

function Write-Banner {
    Write-Host ""
    Write-Host "  ╔═══════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "  ║   OpenSourceBackup — Lokale Installation          ║" -ForegroundColor Cyan
    Write-Host "  ║   Creating backups is easy.                       ║" -ForegroundColor Cyan
    Write-Host "  ║   Proving recoverability is the difference.       ║" -ForegroundColor Cyan
    Write-Host "  ╚═══════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Step  { param($n,$t,$m) Write-Host "`n  ── Schritt $n/$t : $m" -ForegroundColor Cyan }
function Write-Ok    { param($m) Write-Host "  [✓]  $m" -ForegroundColor Green }
function Write-Info  { param($m) Write-Host "  [→]  $m" -ForegroundColor DarkCyan }
function Write-Warn  { param($m) Write-Host "  [!]  $m" -ForegroundColor Yellow }
function Write-Fail  { param($m) Write-Host "  [✗]  $m" -ForegroundColor Red; exit 1 }

Write-Banner

# ── Deinstallation ─────────────────────────────────────────────────────────────

if ($Uninstall) {
    Write-Info "Stoppe und entferne alle OpenSourceBackup-Dienste..."

    foreach ($svc in @("OpenSourceBackupServer","OpenSourceBackupAgent")) {
        $bin = if ($svc -eq "OpenSourceBackupServer") {
            "$InstallDir\server\opensourcebackup-server.exe"
        } else {
            "$InstallDir\agent\opensourcebackup-agent.exe"
        }
        if (Test-Path $bin) {
            & $bin stop    2>$null; Start-Sleep 1
            & $bin uninstall 2>$null
            Write-Ok "Dienst $svc entfernt"
        }
    }

    if (Test-Path "$InstallDir\docker-compose.yml") {
        Write-Info "Stoppe Docker-Container..."
        docker compose -f "$InstallDir\docker-compose.yml" down 2>$null
        Write-Ok "Docker-Container gestoppt"
    }

    $keep = Read-Host "  Datenbankdaten in $InstallDir\data behalten? [J/n]"
    if ($keep -match "^[Nn]") {
        Remove-Item -Recurse -Force "$InstallDir\data" -ErrorAction SilentlyContinue
        Write-Ok "Daten gelöscht"
    } else {
        Write-Ok "Daten behalten unter $InstallDir\data"
    }

    Write-Host ""
    Write-Ok "OpenSourceBackup deinstalliert."
    exit 0
}

# ── Schritt 1: Voraussetzungen prüfen ─────────────────────────────────────────

Write-Step 1 7 "Voraussetzungen prüfen"

# Docker
try {
    $dockerVer = (docker version --format "{{.Server.Version}}" 2>$null)
    if (-not $dockerVer) { throw "Docker nicht erreichbar" }
    Write-Ok "Docker $dockerVer läuft"
} catch {
    Write-Warn "Docker Desktop nicht gefunden oder nicht gestartet."
    Write-Host ""
    Write-Host "  Docker Desktop ist Voraussetzung für PostgreSQL + Redis." -ForegroundColor Yellow
    Write-Host "  Download: https://www.docker.com/products/docker-desktop/" -ForegroundColor Cyan
    Write-Host ""
    $install = Read-Host "  Docker Desktop jetzt herunterladen und installieren? [J/n]"
    if ($install -notmatch "^[Nn]") {
        Start-Process "https://www.docker.com/products/docker-desktop/"
        Write-Fail "Bitte Docker Desktop installieren, dann dieses Script erneut ausführen."
    } else {
        Write-Fail "Docker Desktop erforderlich."
    }
}

# ── Schritt 2: Konfiguration abfragen ─────────────────────────────────────────

Write-Step 2 7 "Konfiguration"

# Backup-Passwort
if (-not $ResticPass) {
    Write-Host ""
    Write-Host "  Das Backup-Passwort verschlüsselt alle Backups." -ForegroundColor Yellow
    Write-Host "  Bitte sicher aufbewahren — ohne es sind keine Restores möglich!" -ForegroundColor Yellow
    Write-Host ""
    $p1 = Read-Host "  Backup-Passwort eingeben" -AsSecureString
    $p2 = Read-Host "  Passwort wiederholen    " -AsSecureString
    $p1c = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($p1))
    $p2c = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($p2))
    if ($p1c -ne $p2c) { Write-Fail "Passwörter stimmen nicht überein." }
    if ($p1c.Length -lt 8) { Write-Fail "Passwort muss mindestens 8 Zeichen lang sein." }
    $ResticPass = $p1c
}

# Backup-Ziel
if (-not $ResticRepo) {
    Write-Host ""
    Write-Host "  Wo sollen Backups gespeichert werden?" -ForegroundColor Yellow
    Write-Host "  Beispiele: C:\Backups  oder  Z:\NAS\Backups  oder  s3:bucket/path" -ForegroundColor DarkCyan
    $ResticRepo = Read-Host "  Backup-Ziel (Enter für Standard: $InstallDir\restic-repo)"
    if (-not $ResticRepo) { $ResticRepo = "$InstallDir\restic-repo" }
}

Write-Ok "Port:        $Port"
Write-Ok "Backup-Ziel: $ResticRepo"
Write-Ok "Install-Dir: $InstallDir"

# ── Schritt 3: Verzeichnisse + Passwörter ─────────────────────────────────────

Write-Step 3 7 "Verzeichnisse anlegen"

$Dirs = @(
    "$InstallDir\server",
    "$InstallDir\agent",
    "$InstallDir\data\postgres",
    "$InstallDir\data\redis",
    "$InstallDir\restore-tests",
    "$InstallDir\restic-repo"
)
foreach ($d in $Dirs) { New-Item -ItemType Directory -Force -Path $d | Out-Null }
Write-Ok "Verzeichnisse angelegt"

# Datenbank-Passwort: einmal generieren und dauerhaft speichern
$DbPassFile = "$InstallDir\data\.db_password"
if (Test-Path $DbPassFile) {
    $DbPassword = Get-Content $DbPassFile -Raw
    Write-Ok "Datenbank-Passwort wiederverwendet (Neuinstallation erkannt)"
} else {
    $DbPassword = [System.Web.Security.Membership]::GeneratePassword(32, 4)
    Set-Content $DbPassFile $DbPassword -Encoding UTF8 -NoNewline
    (Get-Item $DbPassFile).Attributes = "Hidden"
    Write-Ok "Neues Datenbank-Passwort generiert"
}

$DbUrl = "postgres://opensourcebackup:${DbPassword}@127.0.0.1:5432/opensourcebackup?sslmode=disable"

# ── Schritt 4: Docker — PostgreSQL + Redis ────────────────────────────────────

Write-Step 4 7 "PostgreSQL + Redis starten"

$ComposeFile = "$InstallDir\docker-compose.yml"
@"
services:
  postgres:
    image: postgres:16-alpine
    restart: always
    environment:
      POSTGRES_USER: opensourcebackup
      POSTGRES_PASSWORD: ${DbPassword}
      POSTGRES_DB: opensourcebackup
    volumes:
      - $($InstallDir -replace '\\','/')/data/postgres:/var/lib/postgresql/data
    ports:
      - "127.0.0.1:5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U opensourcebackup"]
      interval: 5s
      timeout: 5s
      retries: 20

  redis:
    image: redis:7-alpine
    restart: always
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - $($InstallDir -replace '\\','/')/data/redis:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 10
"@ | Set-Content $ComposeFile -Encoding UTF8

docker compose -f $ComposeFile up -d

Write-Info "Warte auf PostgreSQL (bis zu 60 Sekunden)..."
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep 2
    try {
        $r = docker exec opensourcebackup-postgres-1 pg_isready -U opensourcebackup 2>$null
        if ($r -match "accepting connections") { $ready = $true; break }
    } catch {}
    Write-Host "." -NoNewline
}
Write-Host ""
if (-not $ready) { Write-Fail "PostgreSQL ist nicht bereit. Prüfe: docker compose -f $ComposeFile logs" }
Write-Ok "PostgreSQL + Redis laufen"

# ── Schritt 5: Binaries herunterladen ─────────────────────────────────────────

Write-Step 5 7 "OpenSourceBackup herunterladen"

$BaseUrl = "https://github.com/cerberus8484/opensourcebackup/releases/download/$Version"

# Server
$ServerBin = "$InstallDir\server\opensourcebackup-server.exe"
Write-Info "Lade Control Plane herunter..."
Invoke-WebRequest "$BaseUrl/opensourcebackup-server-windows-amd64.exe" `
    -OutFile $ServerBin -UseBasicParsing
Write-Ok "Server: $ServerBin"

# Agent
$AgentBin = "$InstallDir\agent\opensourcebackup-agent.exe"
Write-Info "Lade Agent herunter..."
Invoke-WebRequest "$BaseUrl/opensourcebackup-agent-windows-amd64.exe" `
    -OutFile $AgentBin -UseBasicParsing
Write-Ok "Agent: $AgentBin"

# Restic
$ResticBin = "$InstallDir\agent\restic.exe"
if (-not (Test-Path $ResticBin)) {
    Write-Info "Lade restic herunter..."
    $rzip = "$env:TEMP\restic.zip"
    Invoke-WebRequest "https://github.com/restic/restic/releases/download/v0.17.3/restic_0.17.3_windows_amd64.zip" `
        -OutFile $rzip -UseBasicParsing
    Expand-Archive -Path $rzip -DestinationPath "$InstallDir\agent" -Force
    $extracted = Get-ChildItem "$InstallDir\agent\restic_*.exe" | Select-Object -First 1
    if ($extracted) { Move-Item $extracted.FullName $ResticBin -Force }
    Remove-Item $rzip -Force
    Write-Ok "restic: $ResticBin"
}

# ── Schritt 6: Control Plane als Windows-Dienst ───────────────────────────────

Write-Step 6 7 "Control Plane als Windows-Dienst installieren"

$LocalIP = (Get-NetIPAddress -AddressFamily IPv4 |
    Where-Object { $_.InterfaceAlias -notmatch 'Loopback|vEthernet' } |
    Select-Object -First 1).IPAddress ?? "localhost"

# Umgebungsvariablen für den Server-Dienst (Machine-Scope)
[Environment]::SetEnvironmentVariable("DATABASE_URL",  $DbUrl,              "Machine")
[Environment]::SetEnvironmentVariable("LISTEN_ADDR",   ":$Port",            "Machine")
[Environment]::SetEnvironmentVariable("CORS_ORIGIN",   "*",                 "Machine")
[Environment]::SetEnvironmentVariable("WEB_UI_DIR",    "",                  "Machine")

# Auch für aktuelle Session
$env:DATABASE_URL = $DbUrl
$env:LISTEN_ADDR  = ":$Port"
$env:CORS_ORIGIN  = "*"

# Datenbank-Migrationen ausführen
Write-Info "Führe Datenbank-Migrationen aus..."
& $ServerBin migrate 2>$null
if ($LASTEXITCODE -ne 0) {
    # Fallback: Server startet und migriert selbst beim ersten Start
    Write-Warn "Migrationen werden beim ersten Start ausgeführt"
}

# Server als Dienst registrieren
Write-Info "Registriere Control Plane Dienst..."
& $ServerBin install
& $ServerBin start
Start-Sleep 3

$svcStatus = & $ServerBin status 2>&1
Write-Ok "Control Plane Dienst läuft (Port $Port)"

# ── Schritt 7: Agent auf diesem Rechner ───────────────────────────────────────

Write-Step 7 7 "Agent für diesen Rechner installieren"

# Enrollment-Token vom laufenden Server holen
Write-Info "Registriere diesen Rechner als Backup-System..."
Start-Sleep 2

$hostname = $env:COMPUTERNAME.ToLower()

# System registrieren
try {
    $sysResp = Invoke-RestMethod `
        -Uri "http://127.0.0.1:$Port/v1/systems" `
        -Method POST `
        -ContentType "application/json" `
        -Body (ConvertTo-Json @{ Hostname = $hostname; RiskClass = "standard" })
    $systemId = $sysResp.ID
    Write-Ok "System '$hostname' registriert (ID: $($systemId.Substring(0,8))…)"
} catch {
    Write-Warn "System-Registrierung fehlgeschlagen — Control Plane noch nicht bereit"
    Write-Warn "Agent kann später über das Dashboard eingerichtet werden"
    $systemId = $null
}

if ($systemId) {
    # Enrollment-Token holen
    try {
        $tokenResp = Invoke-RestMethod `
            -Uri "http://127.0.0.1:$Port/v1/systems/$systemId/enrollment-token" `
            -Method POST `
            -ContentType "application/json"
        $enrollToken = $tokenResp.token

        # Agent-Umgebungsvariablen setzen
        [Environment]::SetEnvironmentVariable("CONTROL_PLANE_URL",  "http://127.0.0.1:$Port", "Machine")
        [Environment]::SetEnvironmentVariable("ENROLLMENT_TOKEN",   $enrollToken,              "Machine")
        [Environment]::SetEnvironmentVariable("RESTIC_PASSWORD",    $ResticPass,               "Machine")
        [Environment]::SetEnvironmentVariable("RESTIC_REPO",        $ResticRepo,               "Machine")
        [Environment]::SetEnvironmentVariable("RESTIC_BIN",         $ResticBin,                "Machine")
        [Environment]::SetEnvironmentVariable("AGENT_TOKEN_FILE",   "$InstallDir\agent\agent-token", "Machine")
        [Environment]::SetEnvironmentVariable("RESTORE_TEST_ROOT",  "$InstallDir\restore-tests",     "Machine")

        $env:CONTROL_PLANE_URL = "http://127.0.0.1:$Port"
        $env:ENROLLMENT_TOKEN  = $enrollToken
        $env:RESTIC_PASSWORD   = $ResticPass
        $env:RESTIC_REPO       = $ResticRepo
        $env:RESTIC_BIN        = $ResticBin
        $env:AGENT_TOKEN_FILE  = "$InstallDir\agent\agent-token"
        $env:RESTORE_TEST_ROOT = "$InstallDir\restore-tests"

        # Agent als Dienst registrieren
        Write-Info "Installiere Agent als Windows-Dienst..."
        & $AgentBin install
        & $AgentBin start
        Start-Sleep 3
        & $AgentBin status
        Write-Ok "Agent läuft als Windows-Dienst"

    } catch {
        Write-Warn "Enrollment fehlgeschlagen — Agent kann später über das Dashboard eingerichtet werden"
    }
}

# ── Zugangsdaten speichern ────────────────────────────────────────────────────

$CredsFile = "$InstallDir\credentials.txt"
@"
=======================================================
  OpenSourceBackup — Lokale Installation
  Installiert: $(Get-Date -Format "dd.MM.yyyy HH:mm")
  DIESE DATEI SICHER AUFBEWAHREN
=======================================================

  Web Dashboard  : http://localhost:$Port/ui/
  Netzwerk-URL   : http://${LocalIP}:$Port/ui/

  Datenbank
  Host     : 127.0.0.1:5432
  User     : opensourcebackup
  Passwort : $DbPassword
  DB-Name  : opensourcebackup

  Backup-Ziel    : $ResticRepo
  Backup-Passwort: (separat gespeichert — NICHT hier ablegen!)

  Dienste (services.msc):
  - OpenSourceBackup Server  (Control Plane)
  - OpenSourceBackup Agent   (Backup-Agent)

=======================================================
"@ | Set-Content $CredsFile -Encoding UTF8
(Get-Item $CredsFile).Attributes = "Hidden"

# ── Fertig ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  ╔═══════════════════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "  ║   ✓  OpenSourceBackup erfolgreich installiert!               ║" -ForegroundColor Green
Write-Host "  ╠═══════════════════════════════════════════════════════════════╣" -ForegroundColor Green
Write-Host "  ║                                                               ║" -ForegroundColor Green
Write-Host "  ║   🌐  WEB DASHBOARD                                           ║" -ForegroundColor Green
Write-Host "  ║       http://localhost:$Port/ui/                              ║" -ForegroundColor Cyan
Write-Host "  ║       http://${LocalIP}:$Port/ui/  (Netzwerk)                 ║" -ForegroundColor Cyan
Write-Host "  ║                                                               ║" -ForegroundColor Green
Write-Host "  ║   ⚙   DIENSTE  (services.msc)                                 ║" -ForegroundColor Green
Write-Host "  ║       OpenSourceBackup Server  — Control Plane               ║" -ForegroundColor Cyan
Write-Host "  ║       OpenSourceBackup Agent   — Backup-Agent                ║" -ForegroundColor Cyan
Write-Host "  ║                                                               ║" -ForegroundColor Green
Write-Host "  ║   💾  BACKUP-ZIEL                                             ║" -ForegroundColor Green
Write-Host "  ║       $ResticRepo" -ForegroundColor Cyan
Write-Host "  ║                                                               ║" -ForegroundColor Green
Write-Host "  ║   📄  ZUGANGSDATEN gespeichert unter:                         ║" -ForegroundColor Green
Write-Host "  ║       $CredsFile" -ForegroundColor Cyan
Write-Host "  ║                                                               ║" -ForegroundColor Green
Write-Host "  ╠═══════════════════════════════════════════════════════════════╣" -ForegroundColor Green
Write-Host "  ║   ⚠   Dienste starten automatisch bei jedem Windows-Boot.    ║" -ForegroundColor Yellow
Write-Host "  ║   ⚠   Dashboard ist nur im lokalen Netzwerk erreichbar.      ║" -ForegroundColor Yellow
Write-Host "  ╚═══════════════════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "  Dashboard jetzt öffnen? " -ForegroundColor Yellow -NoNewline
$open = Read-Host "[J/n]"
if ($open -notmatch "^[Nn]") {
    Start-Process "http://localhost:$Port/ui/"
}
Write-Host ""
