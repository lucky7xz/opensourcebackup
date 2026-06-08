# ==============================================================================
# OpenSourceBackup - Agent Update (Windows)
#
# Swaps the installed agent binary for a new build WITHOUT touching config,
# environment variables, the enrollment token, or the service registration.
#
# What it does: stop service -> back up current exe -> swap -> start -> verify.
#
# Usage (PowerShell as Administrator):
#   .\update-agent.ps1 -NewBinary "C:\path\to\new\opensourcebackup-agent.exe"
#
# If -NewBinary is omitted it looks next to this script in
# ..\dist\agent-update\opensourcebackup-agent.exe.
#
# Rollback: a timestamped .bak is kept in the install dir. To revert, stop the
# service, copy the .bak back over opensourcebackup-agent.exe, start again.
# ==============================================================================

#Requires -RunAsAdministrator

param(
    [string]$NewBinary  = "$PSScriptRoot\..\dist\agent-update\opensourcebackup-agent.exe",
    [string]$InstallDir = "C:\ProgramData\opensourcebackup"
)

$ErrorActionPreference = "Stop"
function Write-Ok   { param($m) Write-Host "  [OK]  $m" -ForegroundColor Green }
function Write-Info { param($m) Write-Host "  [-->] $m" -ForegroundColor Cyan }
function Write-Fail { param($m) Write-Host "  [ERR] $m" -ForegroundColor Red; exit 1 }

$ServiceName = "OpenSourceBackupAgent"
$Target      = Join-Path $InstallDir "opensourcebackup-agent.exe"

Write-Host ""
Write-Host "  OpenSourceBackup - Agent Update" -ForegroundColor Cyan
Write-Host ""

# --- Validate -----------------------------------------------------------------
if (-not (Test-Path $NewBinary)) { Write-Fail "New binary not found: $NewBinary" }
if (-not (Test-Path $Target))    { Write-Fail "No installed agent at $Target - run install-agent.ps1 first" }

$newSize = (Get-Item $NewBinary).Length
$oldSize = (Get-Item $Target).Length
Write-Info "Current: $Target ($oldSize bytes)"
Write-Info "New:     $NewBinary ($newSize bytes)"
if ($newSize -eq $oldSize) {
    Write-Host "  Same size as installed binary - already up to date? Continuing anyway." -ForegroundColor Yellow
}

# --- Stop service -------------------------------------------------------------
Write-Info "Stopping service..."
if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
    & $Target stop 2>$null
    # Fall back to SCM if the agent's own stop did not settle it
    $svc = Get-Service $ServiceName -ErrorAction SilentlyContinue
    if ($svc.Status -ne 'Stopped') { Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue }
    Start-Sleep -Seconds 2
    Write-Ok "Service stopped"
} else {
    Write-Fail "Service '$ServiceName' not found - is the agent installed as a service?"
}

# Make sure no stray process keeps a handle on the exe
Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# --- Backup + swap ------------------------------------------------------------
$stamp  = Get-Date -Format "yyyyMMdd-HHmmss"
$backup = Join-Path $InstallDir "opensourcebackup-agent.exe.bak-$stamp"
Write-Info "Backing up current binary -> $backup"
Copy-Item $Target $backup -Force
Write-Ok "Backup created"

Write-Info "Swapping in new binary..."
Copy-Item $NewBinary $Target -Force
Write-Ok "Binary replaced"

# --- Start + verify -----------------------------------------------------------
Write-Info "Starting service..."
& $Target start
Start-Sleep -Seconds 3

$svc = Get-Service $ServiceName -ErrorAction SilentlyContinue
if ($svc.Status -eq 'Running') {
    Write-Ok "Service running"
} else {
    Write-Host "  [WARN] Service status: $($svc.Status). Check the log below." -ForegroundColor Yellow
}

& $Target status

# --- Show recent log ----------------------------------------------------------
$logFile = Join-Path $InstallDir "agent.log"
if (Test-Path $logFile) {
    Write-Host ""
    Write-Info "Last 12 log lines:"
    Get-Content $logFile -Tail 12 | ForEach-Object { Write-Host "    $_" }
}

Write-Host ""
Write-Host "  Agent updated. Rollback binary: $backup" -ForegroundColor Green
Write-Host "  If anything is wrong: stop service, copy the .bak back, start." -ForegroundColor Yellow
Write-Host ""
