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
    [string]$PollInterval     = $(if ($env:AGENT_POLL_INTERVAL) { $env:AGENT_POLL_INTERVAL } else { "30s" }),
    [string]$OsbVersion       = $(if ($env:OSB_VERSION) { $env:OSB_VERSION } else { "v0.1.0" }),
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
    Write-Info "Stopping and uninstalling agent autostart..."

    # Scheduled Task variant (network-share installs)
    if (Get-ScheduledTask -TaskName $ServiceName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $ServiceName -Confirm:$false
        Write-Ok "Scheduled Task removed"
    }

    # Windows Service variant (local/cloud installs)
    $agentBin = Join-Path $InstallDir "opensourcebackup-agent.exe"
    if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
        if (Test-Path $agentBin) {
            & $agentBin stop    2>$null
            & $agentBin uninstall
            Write-Ok "Service removed"
        } else {
            sc.exe stop $ServiceName    2>$null
            sc.exe delete $ServiceName  2>$null
            Write-Ok "Service removed via sc.exe"
        }
    }

    # Stale autostart Run-key from earlier versions
    $runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
    if (Get-ItemProperty -Path $runKey -Name $ServiceName -ErrorAction SilentlyContinue) {
        Remove-ItemProperty -Path $runKey -Name $ServiceName
        Write-Ok "Removed legacy HKCU\Run autostart"
    }

    Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue | Stop-Process -Force
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

# ── Step 5: Set up autostart ─────────────────────────────────────────────────
#
# Decision: a LocalSystem Windows Service has NO network credential, so it cannot
# reach an SMB/UNC repo (\\server\share). For those we install a Scheduled Task
# that runs as a real user account (which carries the SMB credential) and starts
# without an interactive login. Local/cloud repos use the simpler Windows Service.

$LogFile  = Join-Path $InstallDir "agent.log"
$IsUncRepo = $ResticRepo -match '^\\\\'

if ($IsUncRepo) {
    Write-Info "Repo is a network share ($ResticRepo) — installing as Scheduled Task (runs as user)..."

    $taskUser = if ($env:OSB_TASK_USER) { $env:OSB_TASK_USER } else { "$env:USERDOMAIN\$env:USERNAME" }
    Write-Host ""
    Write-Host "  The agent will run as: $taskUser" -ForegroundColor Yellow
    Write-Host "  This account needs read access to $ResticRepo." -ForegroundColor Yellow
    Write-Host "  The password is stored encrypted (Credential Store) so backups run without login." -ForegroundColor Yellow
    $sec  = Read-Host -AsSecureString "  Windows password for $taskUser"
    $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec)
    $plain = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)

    # Remove any prior service or task so install is idempotent
    if (Get-Service $ServiceName -ErrorAction SilentlyContinue) { & $AgentBin stop 2>$null; & $AgentBin uninstall 2>$null }
    if (Get-ScheduledTask -TaskName $ServiceName -ErrorAction SilentlyContinue) { Unregister-ScheduledTask -TaskName $ServiceName -Confirm:$false }

    $action = New-ScheduledTaskAction -Execute "cmd.exe" `
        -Argument "/c `"`"$AgentBin`" >> `"$LogFile`" 2>&1`"" -WorkingDirectory $InstallDir
    $triggers = @(
        (New-ScheduledTaskTrigger -AtStartup),
        (New-ScheduledTaskTrigger -AtLogOn -User $taskUser)
    )
    $settings = New-ScheduledTaskSettingsSet `
        -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 5) `
        -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew `
        -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries

    # NOTE: -User/-Password and -Principal are mutually exclusive parameter sets.
    # With -Password the LogonType is "Password" (runs without login). -RunLevel Highest.
    Register-ScheduledTask -TaskName $ServiceName -Action $action -Trigger $triggers `
        -Settings $settings -User $taskUser -Password $plain -RunLevel Highest `
        -Description "OpenSourceBackup Agent — polls the control plane and runs backups (runs without login)." | Out-Null
    $plain = $null

    Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-ScheduledTask -TaskName $ServiceName
    Start-Sleep -Seconds 3
    $state = (Get-ScheduledTask -TaskName $ServiceName).State
    Write-Ok "Scheduled Task '$ServiceName' installed (state: $state)"
    $autostartKind = "Scheduled Task (runs as $taskUser, with or without login)"
} else {
    Write-Info "Installing Windows Service (LocalSystem)..."
    & $AgentBin install
    Write-Ok "Windows Service '$ServiceName' installed"

    Write-Info "Starting service..."
    & $AgentBin start
    Start-Sleep -Seconds 3
    & $AgentBin status
    $autostartKind = "Windows Service (LocalSystem)"
}

# ── Done ──────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  ✓ Agent installed — autostart: $autostartKind" -ForegroundColor Green
Write-Host ""
if ($IsUncRepo) {
    Write-Host "  Manage via Task Scheduler: taskschd.msc → $ServiceName" -ForegroundColor Yellow
    Write-Host "  Logfile:  $LogFile" -ForegroundColor Cyan
} else {
    Write-Host "  Commands:" -ForegroundColor Yellow
    Write-Host "  Status:   $AgentBin status"
    Write-Host "  Stop:     $AgentBin stop"
    Write-Host "  Restart:  $AgentBin restart"
    Write-Host "  Remove:   $AgentBin stop; $AgentBin uninstall"
    Write-Host "  Or via Services Manager: services.msc → OpenSourceBackup Agent"
}
Write-Host ""
Write-Host "  Token:    $InstallDir\agent-token" -ForegroundColor Cyan
Write-Host ""
