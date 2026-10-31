# mqvi coturn one-time setup runner (Windows PowerShell)
#
# Installs + configures coturn on a mqvi backend host. Run ONCE per host — coturn
# then lives independently of redeploys (do not run on every deploy).
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File deploy\coturn-setup.ps1
#   powershell -ExecutionPolicy Bypass -File deploy\coturn-setup.ps1 -Server root@1.2.3.4

param(
    [string]$Server
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "  mqvi coturn Setup (one-time)" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# --- Ask for server address if not provided ---
if (-not $Server) {
    $ip = Read-Host "Server address (e.g. root@1.2.3.4)"
    if (-not $ip) {
        Write-Host "  ERROR: Server address is required." -ForegroundColor Red
        exit 1
    }
    $Server = $ip
}
Write-Host "  Target: $Server" -ForegroundColor White
Write-Host ""

# --- SSH Agent: ask passphrase once ---
Write-Host "[1/3] Setting up SSH agent..." -ForegroundColor Yellow
$agentService = Get-Service ssh-agent -ErrorAction SilentlyContinue
if ($agentService) {
    if ($agentService.StartType -eq 'Disabled' -or $agentService.Status -ne 'Running') {
        Write-Host "  SSH agent needs admin to start (one-time)..." -ForegroundColor DarkYellow
        $proc = Start-Process powershell -Verb RunAs -Wait -PassThru -ArgumentList `
            '-NoProfile -Command "Set-Service ssh-agent -StartupType Manual; Start-Service ssh-agent"'
        if ($proc.ExitCode -ne 0) {
            Write-Host "  ERROR: Could not start ssh-agent. Run as admin once or enable the service manually." -ForegroundColor Red
            exit 1
        }
    }
}
$ErrorActionPreference = "Continue"
ssh-add "$env:USERPROFILE\.ssh\id_ed25519" 2>$null
$ErrorActionPreference = "Stop"
Write-Host "  OK - SSH key loaded" -ForegroundColor Green

# --- Upload setup script to /tmp (path-independent; the script auto-detects .env) ---
Write-Host ""
Write-Host "[2/3] Uploading coturn-setup.sh..." -ForegroundColor Yellow
$setupScript = Join-Path $ScriptDir "coturn-setup.sh"
scp $setupScript "${Server}:/tmp/mqvi-coturn-setup.sh"
if ($LASTEXITCODE -ne 0) {
    Write-Host "  ERROR: SCP failed!" -ForegroundColor Red
    exit 1
}
Write-Host "  OK - Uploaded" -ForegroundColor Green

# --- Run setup once (strip CRLF, then run; coturn-setup.sh auto-detects the .env) ---
Write-Host ""
Write-Host "[3/3] Running coturn-setup.sh on the server..." -ForegroundColor Yellow
ssh $Server "sed -i 's/\r`$//' /tmp/mqvi-coturn-setup.sh && sudo bash /tmp/mqvi-coturn-setup.sh; rc=`$?; rm -f /tmp/mqvi-coturn-setup.sh; exit `$rc"
if ($LASTEXITCODE -ne 0) {
    Write-Host "  ERROR: coturn setup failed on the server!" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "  coturn setup complete on $Server" -ForegroundColor Green
Write-Host "  Restart the backend to load TURN_SECRET/TURN_URLS (e.g. redeploy)." -ForegroundColor Yellow
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""
