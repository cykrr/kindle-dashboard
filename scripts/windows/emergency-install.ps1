<#
.SYNOPSIS
    Emergency deploy of the Kindle dashboard over USB mass storage.

.DESCRIPTION
    Use when SSH / USB-net is dead and the only way in is the Kindle mounted
    as a USB drive (e.g. I:\). Cross-compiles the ARM binary inside WSL, then
    copies the binary, launch/stop scripts, HA config and KUAL extensions
    straight onto the drive. Eject when done — the Kindle re-reads /mnt/us.

.PARAMETER Drive
    Drive letter the Kindle is mounted as. Default: I

.PARAMETER Distro
    WSL distro that holds the repo. Default: archlinux

.PARAMETER SkipBuild
    Skip the WSL cross-compile and copy the existing deploy/dashboard-native.

.PARAMETER ExtensionsOnly
    Copy only the KUAL extensions (ssh-manager, Kindle-Dashboard). No build,
    no binary, no dashboard files. Use to push an ssh-manager fix fast.

.PARAMETER NoEject
    Leave the drive mounted after copying. By default the script ejects it so
    the Kindle re-mounts /mnt/us and picks up the new files.

.EXAMPLE
    .\emergency-install.ps1
    .\emergency-install.ps1 -Drive J -SkipBuild
    .\emergency-install.ps1 -ExtensionsOnly
#>

[CmdletBinding()]
param(
    [string]$Drive = "I",
    [string]$Distro = "archlinux",
    [switch]$SkipBuild,
    [switch]$ExtensionsOnly,
    [switch]$NoEject
)

$ErrorActionPreference = "Stop"

function Say($msg, $color = "Cyan") { Write-Host $msg -ForegroundColor $color }
function Die($msg) { Write-Host "ERROR: $msg" -ForegroundColor Red; exit 1 }

# Repo root = two levels up from this script (works over the \\wsl.localhost UNC path too).
$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$DriveRoot = "${Drive}:\"
$DashDir = Join-Path $DriveRoot "kindle-dashboard"
$ExtDir  = Join-Path $DriveRoot "extensions"

Say "=== Emergency install -> $DriveRoot (repo: $RepoRoot) ===" "Cyan"

# --- Sanity: is the drive there and does it look like a Kindle? ---
if (-not (Test-Path $DriveRoot)) { Die "Drive $DriveRoot not found. Plug in the Kindle (USB mass storage)." }
$looksKindle = (Test-Path (Join-Path $DriveRoot "system")) -or (Test-Path (Join-Path $DriveRoot "documents"))
if (-not $looksKindle) {
    Write-Host "WARNING: $DriveRoot has no system/ or documents/ folder — may not be a Kindle." -ForegroundColor Yellow
    $ans = Read-Host "Continue anyway? (y/N)"
    if ($ans -ne "y") { Die "Aborted." }
}

# --- Build + dashboard files (skipped entirely with -ExtensionsOnly) ---
if ($ExtensionsOnly) {
    Say "Extensions-only mode: skipping build and dashboard files." "Yellow"
} else {
    $binary = Join-Path $RepoRoot "deploy\dashboard-native"
    if ($SkipBuild) {
        Say "Skipping build (using existing binary)." "Yellow"
        if (-not (Test-Path $binary)) { Die "No binary at $binary and -SkipBuild set. Build first." }
    } else {
        Say "Cross-compiling in WSL ($Distro)..." "Cyan"
        wsl.exe -d $Distro --cd /home/krr/pw -e bash -lc "./scripts/build/build-test.sh"
        if ($LASTEXITCODE -ne 0) { Die "WSL build failed (exit $LASTEXITCODE)." }
        if (-not (Test-Path $binary)) { Die "Build reported success but $binary missing." }
    }

    New-Item -ItemType Directory -Force -Path $DashDir | Out-Null
    Say "Copying dashboard files -> $DashDir" "Cyan"
    Copy-Item $binary -Destination $DashDir -Force
    Copy-Item (Join-Path $RepoRoot "scripts\kindle\launch.sh") -Destination $DashDir -Force
    Copy-Item (Join-Path $RepoRoot "scripts\kindle\stop.sh")   -Destination $DashDir -Force

    $hassCfg = Join-Path $RepoRoot "hass-config.js"
    if (Test-Path $hassCfg) {
        Copy-Item $hassCfg -Destination $DashDir -Force
        Say "  + hass-config.js" "DarkGray"
    } else {
        Say "  hass-config.js not found in repo root — skipping (device keeps existing one)." "Yellow"
    }
}

# --- Copy KUAL extensions ---
New-Item -ItemType Directory -Force -Path $ExtDir | Out-Null
Say "Copying KUAL extensions -> $ExtDir" "Cyan"
foreach ($ext in @("Kindle-Dashboard", "ssh-manager")) {
    $src = Join-Path $RepoRoot "kindle\kual\$ext"
    if (Test-Path $src) {
        Copy-Item $src -Destination $ExtDir -Recurse -Force
        Say "  + $ext" "DarkGray"
    } else {
        Say "  $ext not found at $src — skipping." "Yellow"
    }
}

Say ""
Say "=== Copy complete ===" "Green"

# --- Eject so the Kindle re-mounts /mnt/us and picks up the new files ---
if ($NoEject) {
    Say "Leaving $DriveRoot mounted (-NoEject). Eject manually to apply." "Yellow"
} else {
    Say "Ejecting $DriveRoot ..." "Cyan"
    try {
        $shell = New-Object -ComObject Shell.Application
        # Namespace 17 = ssfDRIVES (This PC); find the drive and invoke Eject.
        $drv = $shell.Namespace(17).ParseName("${Drive}:")
        if ($null -eq $drv) { throw "drive ${Drive}: not found in shell namespace" }
        $drv.InvokeVerb("Eject")
        Start-Sleep -Seconds 2
        if (Test-Path $DriveRoot) {
            Say "Eject issued but $DriveRoot still visible — unplug/eject manually." "Yellow"
        } else {
            Say "Ejected. Kindle will re-mount /mnt/us shortly." "Green"
        }
    } catch {
        Say "Auto-eject failed: $($_.Exception.Message)" "Yellow"
        Say "Eject $DriveRoot manually (taskbar 'Safely Remove Hardware')." "Yellow"
    } finally {
        if ($shell) { [void][Runtime.InteropServices.Marshal]::ReleaseComObject($shell) }
    }
}

Say ""
Say "Then: launcher runs from KUAL, or SSH in and run:" "White"
Say "     /mnt/us/kindle-dashboard/launch.sh" "White"
Say ""
Say "NOTE: FAT has no exec bit — if the binary won't run, on the device:" "Yellow"
Say "     chmod +x /mnt/us/kindle-dashboard/dashboard-native launch.sh stop.sh" "Yellow"
