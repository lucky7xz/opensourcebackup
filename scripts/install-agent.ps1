# ==============================================================================
# OpenSourceBackup — Agent Install Script (Windows)
# Installs the agent as a permanent Windows Service (auto-start on boot)
#
# Usage (PowerShell as Administrator):
#   $env:CONTROL_PLANE_URL = "http://192.168.1.10:8080"
#   $env:ENROLLMENT_TOKEN  = "<token from wizard>"
#   $env:RESTIC_PASSWORD   = "<your backup password>"
#   $env:RESTIC_REPO       = "Z:\OpenSourceBackup"
#   .\install-agent.ps1
#
# Or one-liner (set vars first):
#   irm http://<server>:8080/scripts/install-agent.ps1 | iex
# ==============================================================================

#Requires -RunAsAdministrator

param(
    [string]$ControlPlaneUrl  = $env:CONTROL_PLANE_URL,
    [string]$EnrollmentToken  = $env:ENROLLMENT_TOKEN,
    [string]$ResticPassword   = $env:RESTIC_PASSWORD,
    [string]$ResticRepo       = $env:RESTIC_REPO,
    [string]$ResticBin        = $env:RESTIC_BIN,
    [string]$PollInterval     = ($env:AGENT_POLL_INTERVAL ?? "30s"),
    [string]$OsbVersion       = ($env:OSB_VERSION ?? "v0.1.0"),
    [string]$InstallDir       = "C:\ProgramData\opensourcebackup",
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"

function Write-Ok   { param($msg) Write-Host "  [OK]  $msg" -ForegroundColor Green }
function Write-Info { param($msg) Write-Host "  [-->] $msg" -ForegroundColor Cyan }
function Write-Fail { param($msg) Write-Host "  [ERR] $msg" -ForegroundColor Red; exit 1 }

$ServiceName = "OpenSourceBackupAgent"

Write-Host ""
Write-Host "  OpenSourceBackup — Agent Installer (Windows)" -ForegroundColor Cyan
Write-Host ""

# ── Uninstall mode ────────────────────────────────────────────────────────────

if ($Uninstall) {
    Write-Info "Stopping and uninstalling service..."
    $agentBin = Join-Path $InstallDir "opensourcebackup-agent.exe"
    if (Test-Path $agentBin) {
        & $agentBin stop    2>$null
        & $agentBin uninstall
        Write-Ok "Service removed"
    } else {
        sc.exe stop $ServiceName    2>$null
        sc.exe delete $ServiceName  2>$null
        Write-Ok "Service removed via sc.exe"
    }
    exit 0
}

# ── Validate required parameters ─────────────────────────────────────────────

if (-not $ControlPlaneUrl) { Write-Fail "CONTROL_PLANE_URL is required" }
if (-not $EnrollmentToken) { Write-Fail "ENROLLMENT_TOKEN is required" }
if (-not $ResticPassword)  { Write-Fail "RESTIC_PASSWORD is required" }
if (-not $ResticRepo)      { Write-Fail "RESTIC_REPO is required" }

# ── Step 1: Create directories ────────────────────────────────────────────────

Write-Info "Creating directories..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\restore-tests" | Out-Null
Write-Ok "Install dir: $InstallDir"

# ── Step 2: Download agent binary ─────────────────────────────────────────────

$AgentBin = Join-Path $InstallDir "opensourcebackup-agent.exe"
Write-Info "Downloading agent (windows-amd64, $OsbVersion)..."
Invoke-WebRequest `
  -Uri "$ControlPlaneUrl/downloads/agent/$OsbVersion/windows-amd64" `
  -OutFile $AgentBin `
  -UseBasicParsing
Write-Ok "Agent: $AgentBin"

# ── Step 3: Download restic ───────────────────────────────────────────────────

$ResticExe = Join-Path $InstallDir "restic.exe"
if (-not (Test-Path $ResticExe)) {
    Write-Info "Downloading restic..."
    $ResticVersion = "0.17.3"
    $ResticZip = Join-Path $env:TEMP "restic.zip"
    Invoke-WebRequest `
      -Uri "https://github.com/restic/restic/releases/download/v${ResticVersion}/restic_${ResticVersion}_windows_amd64.zip" `
      -OutFile $ResticZip `
      -UseBasicParsing
    Expand-Archive -Path $ResticZip -DestinationPath $InstallDir -Force
    # Rename extracted binary
    $extracted = Get-ChildItem "$InstallDir\restic_*.exe" | Select-Object -First 1
    if ($extracted) { Move-Item $extracted.FullName $ResticExe -Force }
    Remove-Item $ResticZip -Force
    Write-Ok "restic: $ResticExe"
} else {
    Write-Ok "restic already present: $ResticExe"
}

if (-not $ResticBin) { $ResticBin = $ResticExe }

# ── Step 4: Set environment variables for the service ────────────────────────

Write-Info "Configuring service environment..."

# These are stored as machine-wide env vars so the Windows Service can read them.
# The agent binary reads them when started as a service.
[System.Environment]::SetEnvironmentVariable("CONTROL_PLANE_URL",  $ControlPlaneUrl, "Machine")
[System.Environment]::SetEnvironmentVariable("ENROLLMENT_TOKEN",   $EnrollmentToken, "Machine")
[System.Environment]::SetEnvironmentVariable("RESTIC_PASSWORD",    $ResticPassword,  "Machine")
[System.Environment]::SetEnvironmentVariable("RESTIC_REPO",        $ResticRepo,      "Machine")
[System.Environment]::SetEnvironmentVariable("RESTIC_BIN",         $ResticBin,       "Machine")
[System.Environment]::SetEnvironmentVariable("AGENT_POLL_INTERVAL",$PollInterval,    "Machine")
[System.Environment]::SetEnvironmentVariable("AGENT_TOKEN_FILE",   "$InstallDir\agent-token", "Machine")
[System.Environment]::SetEnvironmentVariable("RESTORE_TEST_ROOT",  "$InstallDir\restore-tests", "Machine")

# Also set for current process so the install command can pick them up
$env:CONTROL_PLANE_URL   = $ControlPlaneUrl
$env:ENROLLMENT_TOKEN    = $EnrollmentToken
$env:RESTIC_PASSWORD     = $ResticPassword
$env:RESTIC_REPO         = $ResticRepo
$env:RESTIC_BIN          = $ResticBin
$env:AGENT_POLL_INTERVAL = $PollInterval
$env:AGENT_TOKEN_FILE    = "$InstallDir\agent-token"
$env:RESTORE_TEST_ROOT   = "$InstallDir\restore-tests"

Write-Ok "Environment variables set (Machine scope)"

# ── Step 5: Install Windows Service ──────────────────────────────────────────

Write-Info "Installing Windows Service..."
& $AgentBin install
Write-Ok "Windows Service '$ServiceName' installed"

# ── Step 6: Start service ─────────────────────────────────────────────────────

Write-Info "Starting service..."
& $AgentBin start
Start-Sleep -Seconds 3
& $AgentBin status

# ── Done ──────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  ✓ Agent is running as a Windows Service!" -ForegroundColor Green
Write-Host ""
Write-Host "  Commands:" -ForegroundColor Yellow
Write-Host "  Status:   $AgentBin status"
Write-Host "  Stop:     $AgentBin stop"
Write-Host "  Restart:  $AgentBin restart"
Write-Host "  Remove:   $AgentBin stop; $AgentBin uninstall"
Write-Host ""
Write-Host "  Or via Services Manager: services.msc → OpenSourceBackup Agent"
Write-Host ""
Write-Host "  Token:    $InstallDir\agent-token" -ForegroundColor Cyan
Write-Host ""
