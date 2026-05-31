# ==============================================================================
# OpenSourceBackup — Release Build Script (Windows)
# Baut alle Agent-Binaries + Windows MSI/EXE Installer
#
# Voraussetzungen:
#   - Go 1.22+
#   - NSIS (makensis): https://nsis.sourceforge.io/Download
#   - WiX Toolset: dotnet tool install --global wix
#
# Usage:
#   .\scripts\build-release.ps1
#   .\scripts\build-release.ps1 -Version v0.2.0
#   .\scripts\build-release.ps1 -SkipMSI      # nur NSIS EXE
#   .\scripts\build-release.ps1 -SkipNSIS     # nur MSI
# ==============================================================================

param(
    [string]$Version   = "v0.1.0",
    [switch]$SkipMSI,
    [switch]$SkipNSIS,
    [switch]$SkipServer
)

$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent

function Write-Step { param($n,$total,$msg) Write-Host "`n  [$n/$total] $msg" -ForegroundColor Cyan }
function Write-Ok   { param($msg) Write-Host "  [OK]  $msg" -ForegroundColor Green }
function Write-Info { param($msg) Write-Host "  [-->] $msg" -ForegroundColor DarkCyan }
function Write-Fail { param($msg) Write-Host "  [ERR] $msg" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  OpenSourceBackup — Release Build $Version" -ForegroundColor Cyan
Write-Host ""

Set-Location $Root

# ── Verify toolchain ─────────────────────────────────────────────────────────

Write-Step 1 6 "Toolchain prüfen"

$goVersion = (go version 2>&1) -replace ".*go(\d+\.\d+).*",'$1'
if ([version]$goVersion -lt [version]"1.22") { Write-Fail "Go 1.22+ erforderlich (gefunden: go $goVersion)" }
Write-Ok "Go $goVersion"

$hasNSIS = $null -ne (Get-Command makensis -ErrorAction SilentlyContinue)
$hasWix  = $null -ne (Get-Command wix -ErrorAction SilentlyContinue)

if ($hasNSIS) { Write-Ok "NSIS (makensis) gefunden" } else { Write-Host "  [--]  NSIS nicht gefunden — EXE-Installer wird übersprungen" -ForegroundColor Yellow }
if ($hasWix)  { Write-Ok "WiX gefunden" }              else { Write-Host "  [--]  WiX nicht gefunden — MSI-Installer wird übersprungen" -ForegroundColor Yellow }

# ── Directories ───────────────────────────────────────────────────────────────

Write-Step 2 6 "Ausgabe-Verzeichnisse anlegen"

$DistAgent   = Join-Path $Root "dist\agent\$Version"
$DistServer  = Join-Path $Root "dist\server\$Version"
$DistWindows = Join-Path $Root "dist\windows"
$DistTools   = Join-Path $Root "dist\tools"

@($DistAgent, $DistServer, $DistWindows, $DistTools) | ForEach-Object {
    New-Item -ItemType Directory -Force -Path $_ | Out-Null
}
Write-Ok "dist/ Struktur bereit"

# ── Step 3: Agent binaries (alle Plattformen) ─────────────────────────────────

Write-Step 3 6 "Agent-Binaries bauen (alle Plattformen)"

$AgentTargets = @(
    @{ OS="windows"; Arch="amd64"; Suffix=".exe" },
    @{ OS="linux";   Arch="amd64"; Suffix=""     },
    @{ OS="linux";   Arch="arm64"; Suffix=""     },
    @{ OS="freebsd"; Arch="amd64"; Suffix=""     }
)

foreach ($t in $AgentTargets) {
    $name   = "opensourcebackup-agent-$($t.OS)-$($t.Arch)$($t.Suffix)"
    $output = Join-Path $DistAgent $name
    Write-Info "Baue $name ..."
    $env:GOOS   = $t.OS
    $env:GOARCH = $t.Arch
    $env:CGO_ENABLED = "0"
    go build -ldflags "-s -w -X main.version=$Version" -o $output ./cmd/agent/...
    if ($LASTEXITCODE -ne 0) { Write-Fail "Build fehlgeschlagen: $name" }
    Write-Ok $name
}

$env:GOOS = ""; $env:GOARCH = ""; $env:CGO_ENABLED = ""

# ── Step 4: Server binaries ───────────────────────────────────────────────────

if (-not $SkipServer) {
    Write-Step 4 6 "Server-Binaries bauen"

    $ServerTargets = @(
        @{ OS="linux"; Arch="amd64"; Suffix="" },
        @{ OS="linux"; Arch="arm64"; Suffix="" }
    )

    foreach ($t in $ServerTargets) {
        $name   = "opensourcebackup-server-$($t.OS)-$($t.Arch)$($t.Suffix)"
        $output = Join-Path $DistServer $name
        Write-Info "Baue $name ..."
        $env:GOOS   = $t.OS
        $env:GOARCH = $t.Arch
        $env:CGO_ENABLED = "0"
        go build -ldflags "-s -w -X main.version=$Version" -o $output ./cmd/control-plane/...
        if ($LASTEXITCODE -ne 0) { Write-Fail "Build fehlgeschlagen: $name" }
        Write-Ok $name
    }

    $env:GOOS = ""; $env:GOARCH = ""; $env:CGO_ENABLED = ""
} else {
    Write-Host "  [--]  Server-Build übersprungen (-SkipServer)" -ForegroundColor Yellow
}

# ── Step 5: Restic herunterladen (für Windows-Installer) ─────────────────────

Write-Step 5 6 "Restic für Windows-Installer herunterladen"

$ResticExe = Join-Path $DistTools "restic.exe"
if (-not (Test-Path $ResticExe)) {
    $ResticVersion = "0.17.3"
    $ResticZip = Join-Path $env:TEMP "restic.zip"
    Write-Info "Lade restic $ResticVersion herunter..."
    Invoke-WebRequest `
      -Uri "https://github.com/restic/restic/releases/download/v${ResticVersion}/restic_${ResticVersion}_windows_amd64.zip" `
      -OutFile $ResticZip -UseBasicParsing
    Expand-Archive -Path $ResticZip -DestinationPath $DistTools -Force
    $extracted = Get-ChildItem "$DistTools\restic_*.exe" | Select-Object -First 1
    if ($extracted) { Move-Item $extracted.FullName $ResticExe -Force }
    Remove-Item $ResticZip -Force
    Write-Ok "restic.exe -> $ResticExe"
} else {
    Write-Ok "restic.exe bereits vorhanden"
}

# ── Step 6: Windows Installer ─────────────────────────────────────────────────

Write-Step 6 6 "Windows-Installer bauen"

$AgentExeForWindows = Join-Path $DistAgent "opensourcebackup-agent-windows-amd64.exe"

if (-not (Test-Path $AgentExeForWindows)) {
    Write-Fail "Agent-Binary nicht gefunden: $AgentExeForWindows"
}

# ── NSIS EXE ──
if ($hasNSIS -and -not $SkipNSIS) {
    Write-Info "Baue NSIS EXE-Installer..."
    $nsiFile = Join-Path $Root "build\windows\agent-installer.nsi"
    makensis /V2 $nsiFile
    if ($LASTEXITCODE -ne 0) { Write-Fail "NSIS Build fehlgeschlagen" }
    $ExeOut = Join-Path $DistWindows "OpenSourceBackup-Agent-Setup.exe"
    Write-Ok "EXE-Installer: $ExeOut"
} elseif (-not $hasNSIS) {
    Write-Host "  [--]  NSIS nicht gefunden. Installieren:" -ForegroundColor Yellow
    Write-Host "        https://nsis.sourceforge.io/Download" -ForegroundColor Yellow
}

# ── WiX MSI ──
if ($hasWix -and -not $SkipMSI) {
    Write-Info "Baue WiX MSI-Installer..."
    $wxsFile  = Join-Path $Root "build\windows\agent.wxs"
    $msiOut   = Join-Path $DistWindows "OpenSourceBackup-Agent-$Version.msi"
    Set-Location (Join-Path $Root "build\windows")
    wix build agent.wxs -o $msiOut
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [!]   MSI-Build fehlgeschlagen — WiX-Fehler prüfen" -ForegroundColor Yellow
    } else {
        Write-Ok "MSI-Installer: $msiOut"
    }
    Set-Location $Root
} elseif (-not $hasWix) {
    Write-Host "  [--]  WiX nicht gefunden. Installieren:" -ForegroundColor Yellow
    Write-Host "        dotnet tool install --global wix" -ForegroundColor Yellow
}

# ── SHA256 Checksums ──────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  Erstelle Checksums..." -ForegroundColor Cyan

$AllDists = Get-ChildItem -Recurse -File (Join-Path $Root "dist") |
    Where-Object { $_.Extension -notin @(".md5",".sha256",".txt") }

$ChecksumFile = Join-Path $Root "dist\checksums.sha256.txt"
$AllDists | ForEach-Object {
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower()
    $rel  = $_.FullName.Substring($Root.Length + 1) -replace '\\','/'
    "$hash  $rel"
} | Set-Content $ChecksumFile -Encoding utf8
Write-Ok "Checksums: dist\checksums.sha256.txt"

# ── Zusammenfassung ───────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  ═══════════════════════════════════════════" -ForegroundColor Green
Write-Host "  ✓  Release $Version fertig!" -ForegroundColor Green
Write-Host "  ═══════════════════════════════════════════" -ForegroundColor Green
Write-Host ""
Write-Host "  Agent-Binaries:   dist\agent\$Version\" -ForegroundColor Cyan
Write-Host "  Server-Binaries:  dist\server\$Version\" -ForegroundColor Cyan
if ($hasNSIS) {
    Write-Host "  Windows EXE:      dist\windows\OpenSourceBackup-Agent-Setup.exe" -ForegroundColor Cyan
}
if ($hasWix) {
    Write-Host "  Windows MSI:      dist\windows\OpenSourceBackup-Agent-$Version.msi" -ForegroundColor Cyan
}
Write-Host "  Checksums:        dist\checksums.sha256.txt" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Silent MSI-Install Beispiel:" -ForegroundColor Yellow
Write-Host "  msiexec /qn /i OpenSourceBackup-Agent-$Version.msi" -ForegroundColor DarkCyan
Write-Host "    CONTROL_PLANE_URL=http://192.168.1.10:8080" -ForegroundColor DarkCyan
Write-Host "    ENROLLMENT_TOKEN=abc123" -ForegroundColor DarkCyan
Write-Host "    RESTIC_PASSWORD=meinPasswort" -ForegroundColor DarkCyan
Write-Host "    RESTIC_REPO=Z:\Backups" -ForegroundColor DarkCyan
Write-Host ""
