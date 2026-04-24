# ============================================================================
# Hera Installer for Windows
# ============================================================================
# Installation script for Windows (PowerShell).
# Downloads the latest Hera binary release.
#
# Usage:
#   irm https://raw.githubusercontent.com/sadewadee/hera/main/scripts/install.ps1 | iex
#
# Or download and run with options:
#   .\install.ps1 -SkipSetup
#
# ============================================================================

param(
    [switch]$SkipSetup,
    [string]$Branch = "main",
    [string]$HeraHome = "$env:LOCALAPPDATA\hera",
    [string]$InstallDir = "$env:LOCALAPPDATA\hera\bin"
)

$ErrorActionPreference = "Stop"

# ============================================================================
# Configuration
# ============================================================================

$RepoUrlHttps = "https://github.com/sadewadee/hera"
$BinaryName = "hera.exe"

# ============================================================================
# Helper functions
# ============================================================================

function Write-Banner {
    Write-Host ""
    Write-Host "+-----------------------------------------------------------+" -ForegroundColor Magenta
    Write-Host "|                 Hera Agent Installer                      |" -ForegroundColor Magenta
    Write-Host "+-----------------------------------------------------------+" -ForegroundColor Magenta
    Write-Host "|  A self-improving multi-platform AI agent.                |" -ForegroundColor Magenta
    Write-Host "+-----------------------------------------------------------+" -ForegroundColor Magenta
    Write-Host ""
}

function Write-Info {
    param([string]$Message)
    Write-Host "-> $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[!] $Message" -ForegroundColor Yellow
}

function Write-Fail {
    param([string]$Message)
    Write-Host "[X] $Message" -ForegroundColor Red
}

# ============================================================================
# Detect architecture
# ============================================================================

function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default {
            Write-Fail "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# ============================================================================
# Download and install
# ============================================================================

function Install-Hera {
    $arch = Get-Arch
    $os = "windows"

    Write-Info "Detected: $os/$arch"

    # Create directories
    if (-not (Test-Path $HeraHome)) {
        New-Item -ItemType Directory -Path $HeraHome -Force | Out-Null
    }
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    # Fetch latest release tag
    Write-Info "Fetching latest release..."
    $releases = Invoke-RestMethod -Uri "$RepoUrlHttps/releases/latest" -MaximumRedirection 0 -ErrorAction SilentlyContinue 2>$null
    $tag = ($releases.tag_name) -replace '^v', ''

    if (-not $tag) {
        Write-Warn "Could not determine latest release. Using 'latest'."
        $tag = "latest"
    }

    $zipName = "hera_${tag}_${os}_${arch}.zip"
    $downloadUrl = "$RepoUrlHttps/releases/latest/download/$zipName"

    Write-Info "Downloading $zipName..."
    $zipPath = Join-Path $env:TEMP $zipName
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
    } catch {
        Write-Fail "Download failed: $_"
        Write-Fail "URL: $downloadUrl"
        exit 1
    }

    Write-Info "Extracting to $InstallDir..."
    Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force
    Remove-Item $zipPath -Force

    # Verify binary
    $binaryPath = Join-Path $InstallDir $BinaryName
    if (-not (Test-Path $binaryPath)) {
        Write-Fail "Binary not found after extraction: $binaryPath"
        exit 1
    }

    Write-Success "Hera installed to $binaryPath"

    # Add to PATH
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-Info "Adding $InstallDir to user PATH..."
        [Environment]::SetEnvironmentVariable(
            "Path",
            "$currentPath;$InstallDir",
            "User"
        )
        Write-Success "Added to PATH. Restart your terminal for changes to take effect."
    } else {
        Write-Info "$InstallDir already in PATH."
    }

    # Set HERA_HOME
    [Environment]::SetEnvironmentVariable("HERA_HOME", $HeraHome, "User")
    Write-Success "HERA_HOME set to $HeraHome"
}

# ============================================================================
# Setup wizard
# ============================================================================

function Run-Setup {
    Write-Info "Running initial setup..."
    $binaryPath = Join-Path $InstallDir $BinaryName
    & $binaryPath setup
}

# ============================================================================
# Main
# ============================================================================

Write-Banner
Install-Hera

if (-not $SkipSetup) {
    Run-Setup
}

Write-Host ""
Write-Success "Installation complete!"
Write-Host ""
Write-Host "  To get started:"
Write-Host "    hera            # Start interactive session"
Write-Host "    hera setup      # Re-run setup wizard"
Write-Host "    hera --help     # Show all commands"
Write-Host ""
