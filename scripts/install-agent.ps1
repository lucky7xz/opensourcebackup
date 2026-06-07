# ==============================================================================
# OpenSourceBackup - Agent Install Script (Windows)
# Installs the agent as a permanent Windows Service (auto-start on boot).
#
# For a network-share repo (\\server\share) the service runs under a user
# account, because a LocalSystem service has no network credential. The account
# is granted the "Log on as a service" right automatically.
#
# Usage (PowerShell as Administrator):
#   $env:CONTROL_PLANE_URL = "http://192.168.1.10:8080"
#   $env:ENROLLMENT_TOKEN  = "<token from wizard>"
#   $env:RESTIC_PASSWORD   = "<your backup password>"
#   $env:RESTIC_REPO       = "\\192.168.0.32\Public\OpenSourceBackup"
#   .\install-agent.ps1
#
# Optional for network-share repos (else current user + interactive prompt):
#   $env:OSB_SERVICE_USER     = "DOMAIN\user"
#   $env:OSB_SERVICE_PASSWORD = "<windows password>"
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

# Grants the "Log on as a service" right (SeServiceLogonRight) to an account via
# the LSA policy API. A Windows service running under a user account fails to
# start without it, and Windows does not grant it automatically via sc/SCM.
function Grant-LogonAsService {
    param([Parameter(Mandatory)][string]$Account)

    $sid = (New-Object System.Security.Principal.NTAccount($Account)).Translate([System.Security.Principal.SecurityIdentifier])

    if (-not ([System.Management.Automation.PSTypeName]'OSB.LsaRights').Type) {
        $csharp = @'
using System;
using System.Runtime.InteropServices;
namespace OSB {
  public static class LsaRights {
    [StructLayout(LayoutKind.Sequential)]
    struct LSA_UNICODE_STRING { public ushort Length; public ushort MaximumLength; public IntPtr Buffer; }
    [StructLayout(LayoutKind.Sequential)]
    struct LSA_OBJECT_ATTRIBUTES { public int Length; public IntPtr RootDirectory; public IntPtr ObjectName; public uint Attributes; public IntPtr SecurityDescriptor; public IntPtr SecurityQualityOfService; }
    [DllImport("advapi32.dll", SetLastError=true)]
    static extern uint LsaOpenPolicy(IntPtr SystemName, ref LSA_OBJECT_ATTRIBUTES oa, uint access, out IntPtr handle);
    [DllImport("advapi32.dll", SetLastError=true)]
    static extern uint LsaAddAccountRights(IntPtr handle, byte[] sid, LSA_UNICODE_STRING[] rights, uint count);
    [DllImport("advapi32.dll")] static extern uint LsaClose(IntPtr handle);
    [DllImport("advapi32.dll")] static extern int LsaNtStatusToWinError(uint status);
    static LSA_UNICODE_STRING Str(string s) {
      LSA_UNICODE_STRING u = new LSA_UNICODE_STRING();
      u.Buffer = Marshal.StringToHGlobalUni(s);
      u.Length = (ushort)(s.Length * 2);
      u.MaximumLength = (ushort)(s.Length * 2 + 2);
      return u;
    }
    public static void AddRight(byte[] sid, string right) {
      LSA_OBJECT_ATTRIBUTES oa = new LSA_OBJECT_ATTRIBUTES();
      IntPtr h;
      uint st = LsaOpenPolicy(IntPtr.Zero, ref oa, 0x000F0FFF, out h);
      if (st != 0) throw new Exception("LsaOpenPolicy failed, win error " + LsaNtStatusToWinError(st));
      try {
        LSA_UNICODE_STRING[] r = new LSA_UNICODE_STRING[] { Str(right) };
        uint res = LsaAddAccountRights(h, sid, r, 1);
        if (res != 0) throw new Exception("LsaAddAccountRights failed, win error " + LsaNtStatusToWinError(res));
      } finally { LsaClose(h); }
    }
  }
}
'@
        Add-Type -TypeDefinition $csharp -Language CSharp
    }

    $bytes = New-Object byte[] $sid.BinaryLength
    $sid.GetBinaryForm($bytes, 0)
    [OSB.LsaRights]::AddRight($bytes, "SeServiceLogonRight")
}

Write-Host ""
Write-Host "  OpenSourceBackup - Agent Installer (Windows)" -ForegroundColor Cyan
Write-Host ""

# --- Uninstall mode -----------------------------------------------------------

if ($Uninstall) {
    Write-Info "Stopping and uninstalling agent autostart..."

    $agentBin = Join-Path $InstallDir "opensourcebackup-agent.exe"
    if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
        if (Test-Path $agentBin) { & $agentBin stop 2>$null; & $agentBin uninstall }
        else { sc.exe stop $ServiceName 2>$null; sc.exe delete $ServiceName 2>$null }
        Write-Ok "Service removed"
    }

    # Legacy Scheduled Task (older installs)
    if (Get-ScheduledTask -TaskName $ServiceName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $ServiceName -Confirm:$false
        Write-Ok "Legacy Scheduled Task removed"
    }
    # Legacy HKCU Run-key (older installs)
    $runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
    if (Get-ItemProperty -Path $runKey -Name $ServiceName -ErrorAction SilentlyContinue) {
        Remove-ItemProperty -Path $runKey -Name $ServiceName
        Write-Ok "Removed legacy HKCU\Run autostart"
    }

    Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue | Stop-Process -Force
    exit 0
}

# --- Validate required parameters ---------------------------------------------

if (-not $ControlPlaneUrl) { Write-Fail "CONTROL_PLANE_URL is required" }
if (-not $EnrollmentToken) { Write-Fail "ENROLLMENT_TOKEN is required" }
if (-not $ResticPassword)  { Write-Fail "RESTIC_PASSWORD is required" }
if (-not $ResticRepo)      { Write-Fail "RESTIC_REPO is required" }

# --- Step 1: Create directories -----------------------------------------------

Write-Info "Creating directories..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path "$InstallDir\restore-tests" | Out-Null
Write-Ok "Install dir: $InstallDir"

# --- Step 2: Download agent binary --------------------------------------------

$AgentBin = Join-Path $InstallDir "opensourcebackup-agent.exe"
Write-Info "Downloading agent (windows-amd64, $OsbVersion)..."
Invoke-WebRequest -Uri "$ControlPlaneUrl/downloads/agent/$OsbVersion/windows-amd64" -OutFile $AgentBin -UseBasicParsing
Write-Ok "Agent: $AgentBin"

# --- Step 3: Download restic --------------------------------------------------

$ResticExe = Join-Path $InstallDir "restic.exe"
if (-not (Test-Path $ResticExe)) {
    Write-Info "Downloading restic..."
    $ResticVersion = "0.17.3"
    $ResticZip = Join-Path $env:TEMP "restic.zip"
    Invoke-WebRequest -Uri "https://github.com/restic/restic/releases/download/v${ResticVersion}/restic_${ResticVersion}_windows_amd64.zip" -OutFile $ResticZip -UseBasicParsing
    Expand-Archive -Path $ResticZip -DestinationPath $InstallDir -Force
    $extracted = Get-ChildItem "$InstallDir\restic_*.exe" | Select-Object -First 1
    if ($extracted) { Move-Item $extracted.FullName $ResticExe -Force }
    Remove-Item $ResticZip -Force
    Write-Ok "restic: $ResticExe"
} else {
    Write-Ok "restic already present: $ResticExe"
}
if (-not $ResticBin) { $ResticBin = $ResticExe }

# --- Step 4: Set machine-wide environment for the service ---------------------

Write-Info "Configuring service environment..."
[System.Environment]::SetEnvironmentVariable("CONTROL_PLANE_URL",  $ControlPlaneUrl, "Machine")
[System.Environment]::SetEnvironmentVariable("ENROLLMENT_TOKEN",   $EnrollmentToken, "Machine")
[System.Environment]::SetEnvironmentVariable("RESTIC_PASSWORD",    $ResticPassword,  "Machine")
[System.Environment]::SetEnvironmentVariable("RESTIC_REPO",        $ResticRepo,      "Machine")
[System.Environment]::SetEnvironmentVariable("RESTIC_BIN",         $ResticBin,       "Machine")
[System.Environment]::SetEnvironmentVariable("AGENT_POLL_INTERVAL",$PollInterval,    "Machine")
[System.Environment]::SetEnvironmentVariable("AGENT_TOKEN_FILE",   "$InstallDir\agent-token",   "Machine")
[System.Environment]::SetEnvironmentVariable("RESTORE_TEST_ROOT",  "$InstallDir\restore-tests", "Machine")
[System.Environment]::SetEnvironmentVariable("AGENT_LOG_FILE",     "$InstallDir\agent.log",     "Machine")

$env:CONTROL_PLANE_URL   = $ControlPlaneUrl
$env:ENROLLMENT_TOKEN    = $EnrollmentToken
$env:RESTIC_PASSWORD     = $ResticPassword
$env:RESTIC_REPO         = $ResticRepo
$env:RESTIC_BIN          = $ResticBin
$env:AGENT_POLL_INTERVAL = $PollInterval
$env:AGENT_TOKEN_FILE    = "$InstallDir\agent-token"
$env:RESTORE_TEST_ROOT   = "$InstallDir\restore-tests"
$env:AGENT_LOG_FILE      = "$InstallDir\agent.log"
Write-Ok "Environment variables set (Machine scope)"

# --- Step 5: Install as a Windows Service -------------------------------------
#
# A LocalSystem service has NO network credential and cannot reach an SMB/UNC
# repo (\\server\share). For those the service runs under a user account (which
# carries the SMB credential) and is granted the "Log on as a service" right.
# Local/cloud repos use LocalSystem.

$IsUncRepo = $ResticRepo -match '^\\\\'

# Idempotent: remove any prior service / legacy task / running process
if (Get-Service $ServiceName -ErrorAction SilentlyContinue) { & $AgentBin stop 2>$null; & $AgentBin uninstall 2>$null; Start-Sleep -Seconds 1 }
if (Get-ScheduledTask -TaskName $ServiceName -ErrorAction SilentlyContinue) { Unregister-ScheduledTask -TaskName $ServiceName -Confirm:$false }
Get-Process -Name "opensourcebackup-agent" -ErrorAction SilentlyContinue | Stop-Process -Force

if ($IsUncRepo) {
    $svcUser = if ($env:OSB_SERVICE_USER) { $env:OSB_SERVICE_USER } else { "$env:USERDOMAIN\$env:USERNAME" }
    Write-Info "Network-share repo ($ResticRepo) - service runs as user: $svcUser"
    Write-Host "  This account needs read/write access to $ResticRepo." -ForegroundColor Yellow

    if ($env:OSB_SERVICE_PASSWORD) {
        $plain = $env:OSB_SERVICE_PASSWORD
    } else {
        $sec  = Read-Host -AsSecureString "  Windows password for $svcUser"
        $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec)
        $plain = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
    }

    Grant-LogonAsService -Account $svcUser
    Write-Ok "Granted 'Log on as a service' to $svcUser"

    # The agent reads OSB_SERVICE_USER / OSB_SERVICE_PASSWORD and registers the
    # service to run under that account (password via env, never on argv).
    $env:OSB_SERVICE_USER     = $svcUser
    $env:OSB_SERVICE_PASSWORD = $plain
    & $AgentBin install
    $env:OSB_SERVICE_PASSWORD = $null; $plain = $null
    $autostartKind = "Windows Service (runs as $svcUser)"
} else {
    Write-Info "Local/cloud repo - installing Windows Service (LocalSystem)..."
    & $AgentBin install
    $autostartKind = "Windows Service (LocalSystem)"
}

Write-Ok "Windows Service '$ServiceName' installed"
Write-Info "Starting service..."
& $AgentBin start
Start-Sleep -Seconds 3
& $AgentBin status

# --- Done ---------------------------------------------------------------------

Write-Host ""
Write-Host "  Agent installed - autostart: $autostartKind" -ForegroundColor Green
Write-Host ""
Write-Host "  Manage:   services.msc  ->  OpenSourceBackup Agent" -ForegroundColor Yellow
Write-Host "  Commands: $AgentBin status | stop | restart | uninstall"
Write-Host "  Logfile:  $InstallDir\agent.log" -ForegroundColor Cyan
Write-Host "  Token:    $InstallDir\agent-token" -ForegroundColor Cyan
Write-Host ""
