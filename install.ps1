# install.ps1 — OpenConveneCLI installer for Windows (PowerShell)
#
# Usage:
#   irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex
#
# This script downloads the latest release binary from GitHub and installs it
# to a directory in PATH (prefers $env:LOCALAPPDATA\openconvene, falls back to cwd).

$ErrorActionPreference = "Stop"

$Repo = "masteryee-labs/open-convene-cli"
$BinaryName = "openconvene.exe"

function Write-Info  { Write-Host "[INFO]  $args" -ForegroundColor Blue }
function Write-Warn  { Write-Host "[WARN]  $args" -ForegroundColor Yellow }
function Write-Err   { Write-Host "[ERROR] $args" -ForegroundColor Red; exit 1 }
function Write-Ok    { Write-Host "[OK]    $args" -ForegroundColor Green }

# Detect architecture
$Arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -match "ARM64") {
    $Arch = "arm64"
}

Write-Info "Detected: windows/$Arch"

# Get latest release tag from GitHub API
Write-Info "Fetching latest release version..."
try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "openconvene-installer" }
    $LatestTag = $Release.tag_name
} catch {
    Write-Err "Failed to fetch latest release: $_"
}

if (-not $LatestTag) {
    Write-Err "Failed to determine latest release version."
}

Write-Info "Latest version: $LatestTag"

# Construct download URL
$AssetName = "openconvene-windows-$Arch.exe"
$DownloadUrl = "https://github.com/$Repo/releases/download/$LatestTag/$AssetName"

# Determine install directory
$InstallDir = "$env:LOCALAPPDATA\openconvene"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$DestPath = Join-Path $InstallDir $BinaryName

# Download
Write-Info "Downloading $AssetName..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $DestPath -UseBasicParsing
} catch {
    Write-Err "Failed to download $DownloadUrl`nPlease check that the release asset exists.`nError: $_"
}

Write-Ok "OpenConveneCLI $LatestTag installed to $DestPath"

# Check PATH
$PathDirs = $env:PATH -split ";"
if ($PathDirs -notcontains $InstallDir) {
    Write-Warn "$InstallDir is not in your PATH."
    Write-Warn "To add it permanently, run this as Administrator:"
    Write-Host ""
    Write-Host "  [Environment]::SetEnvironmentVariable('PATH', `"$InstallDir;`$([Environment]::GetEnvironmentVariable('PATH', 'User'))`", 'User')"
    Write-Host ""
    Write-Info "For this session, adding to PATH temporarily..."
    $env:PATH = "$InstallDir;$env:PATH"
}

# Verify
Write-Info "Verifying installation..."
try {
    $Version = & $DestPath --version 2>&1
    Write-Ok "Installed version: $Version"
} catch {
    Write-Warn "Could not verify version (this is OK if it's the first run)."
}

Write-Host ""
Write-Host "  Get started:"
Write-Host "    openconvene detect    # detect installed AI CLIs"
Write-Host "    openconvene init      # generate config"
Write-Host "    openconvene           # enter interactive REPL"
Write-Host ""
Write-Ok "Done!"
