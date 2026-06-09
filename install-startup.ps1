<#
.SYNOPSIS
    Builds, installs to Program Files, registers for auto-start, and launches the Windows Macro Daemon.
.DESCRIPTION
    Builds the daemon (optional), copies it to %ProgramFiles%\KindleDashboard,
    creates directories in %ProgramData%\KindleDashboard for logs/config,
    adds a Run registry key for auto-start on login, and launches it.

    Elevates via gsudo if not already admin.
.PARAMETER Build
    Run 'go mod tidy && go build' before installing.
.PARAMETER Uninstall
    Remove the startup entry, stop the daemon, and remove Program Files dir.
.PARAMETER NoLaunch
    Install but don't start the daemon.
#>

param(
    [switch]$Build,
    [switch]$Uninstall,
    [switch]$NoLaunch
)

# ── Configuration ──
$regPath = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
$regName = "KindleMacroDaemon"
$installDir = "$env:ProgramFiles\KindleDashboard"
$dataDir    = "$env:ProgramData\KindleDashboard"
$binaryPath = "$installDir\macro-daemon.exe"
$logPath    = "$dataDir\macro-daemon.log"
$configPath = "$dataDir\.env"

$scriptDir = $PSScriptRoot
if (-not $scriptDir) { $scriptDir = Get-Location }

# ── Elevate via gsudo if not admin ──
$gsudoPath = if ($env:MACRO_GSUDO) { $env:MACRO_GSUDO } else { "C:\Users\krr\scoop\apps\gsudo\current\gsudo.exe" }
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    if (-not (Test-Path $gsudoPath)) {
        Write-Error "Not running as admin and gsudo not found at $gsudoPath. Set MACRO_GSUDO env var or run from an admin prompt."
        exit 1
    }

    $myArgs = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "`"$PSCommandPath`"")
    if ($Build)     { $myArgs += "-Build" }
    if ($Uninstall) { $myArgs += "-Uninstall" }
    if ($NoLaunch)  { $myArgs += "-NoLaunch" }

    Write-Host "→ Elevating via gsudo..." -ForegroundColor Yellow
    & $gsudoPath powershell.exe $myArgs
    exit $LASTEXITCODE
}

# ── Build ──
if ($Build) {
    Write-Host "→ Running go mod tidy..." -ForegroundColor Cyan
    & "go" "mod" "tidy"
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go mod tidy failed"
        exit 1
    }

    Write-Host "→ Building macro-daemon.exe ..." -ForegroundColor Cyan
    & "go" "build" "-o" "macro-daemon.exe" "-ldflags" "-H windowsgui" "."
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go build failed"
        exit 1
    }
    Write-Host "✓ Build complete" -ForegroundColor Green
}

# ── Uninstall ──
if ($Uninstall) {
    # Stop running daemon
    $procs = Get-Process -Name "macro-daemon" -ErrorAction SilentlyContinue
    if ($procs) {
        $procs | Stop-Process -Force
        Write-Host "Stopped running daemon process(es)"
    }

    # Remove Run key
    if (Get-ItemProperty -Path $regPath -Name $regName -ErrorAction SilentlyContinue) {
        Remove-ItemProperty -Path $regPath -Name $regName
        Write-Host "Removed startup entry: $regName"
    } else {
        Write-Host "No startup entry found for $regName"
    }

    # Remove installed files
    if (Test-Path $installDir) {
        Remove-Item -Path $installDir -Recurse -Force
        Write-Host "Removed: $installDir"
    }
    Write-Host "Preserved data directory: $dataDir (delete manually if desired)"
    return
}

# ── Ensure source binary ──
$sourceExe = Join-Path $scriptDir "macro-daemon.exe"
if (-not (Test-Path $sourceExe)) {
    Write-Error "Binary not found at $sourceExe — re-run with -Build to compile it"
    exit 1
}

# ── Install directories ──
Write-Host "→ Installing to $installDir ..." -ForegroundColor Cyan
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
New-Item -ItemType Directory -Path $dataDir -Force | Out-Null

# ── Copy binary ──
Copy-Item -Path $sourceExe -Destination $binaryPath -Force
Write-Host "✓ Installed binary: $binaryPath"

# ── Seed .env from existing ──
$sourceEnv = Join-Path $scriptDir ".env"
if (Test-Path $sourceEnv) {
    Copy-Item -Path $sourceEnv -Destination $configPath -Force
    Write-Host "✓ Copied config: $configPath"
} elseif (-not (Test-Path $configPath)) {
    @"
MACRO_API_KEY=your-super-secret-key
MACRO_PORT=:8080
MACRO_GSUDO=C:\Users\krr\scoop\apps\gsudo\current\gsudo.exe
"@ | Out-File -FilePath $configPath -Encoding UTF8
    Write-Host "✓ Created default config: $configPath (edit your API key!)"
}

# ── Registry Run key ──
Set-ItemProperty -Path $regPath -Name $regName -Value $binaryPath
Write-Host "✓ Added startup entry: $regName → $binaryPath" -ForegroundColor Green

# ── Launch ──
if (-not $NoLaunch) {
    $running = Get-Process -Name "macro-daemon" -ErrorAction SilentlyContinue
    if (-not $running) {
        try {
            $proc = Start-Process -FilePath $binaryPath -WindowStyle Hidden -PassThru
            Write-Host "✓ Started daemon (PID $($proc.Id))" -ForegroundColor Green
        } catch {
            Write-Error "Failed to start daemon: $_"
            exit 1
        }
    } else {
        Write-Host "✓ Daemon already running (PID $($running.Id)) — restart for new binary" -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠ Skipped launch (-NoLaunch)" -ForegroundColor Yellow
}

Write-Host "`nInstall complete. The daemon will auto-start on next login." -ForegroundColor Green
Write-Host "Logs: $logPath" -ForegroundColor Gray
Write-Host "Config: $configPath" -ForegroundColor Gray
