# Setup Kindle USB Networking (RNDIS)
# Run as administrator

$usbIp = "192.168.15.201"
$subnet = "255.255.255.0"
$gateway = "192.168.15.200"

Write-Host "=== Kindle USB Networking Setup ===" -ForegroundColor Cyan

# Find the RNDIS adapter
$adapter = Get-NetAdapter | Where-Object { $_.InterfaceDescription -like "*RNDIS*" -or $_.InterfaceDescription -like "*Remote NDIS*" -or $_.Name -like "*Ethernet*" } | Where-Object { $_.Status -eq "Up" } | Select-Object -First 1

if (-not $adapter) {
    Write-Host "No RNDIS adapter found. Make sure Kindle is plugged in and USB Net is started." -ForegroundColor Yellow
    Write-Host "Check 'Control Panel > Network Connections' for a new adapter." -ForegroundColor Yellow
    exit 1
}

Write-Host "Found adapter: $($adapter.Name) - $($adapter.InterfaceDescription)" -ForegroundColor Green

# Set static IP
New-NetIPAddress -InterfaceAlias $adapter.Name -IPAddress $usbIp -PrefixLength 24 -DefaultGateway $gateway -ErrorAction SilentlyContinue
if (-not $?) {
    # Address might already exist, update it
    Set-NetIPAddress -InterfaceAlias $adapter.Name -IPAddress $usbIp -PrefixLength 24 -DefaultGateway $gateway
}

Write-Host "IP set to $usbIp" -ForegroundColor Green
Write-Host "Kindle is at 192.168.15.200:2222" -ForegroundColor Green
Write-Host ""
Write-Host "Connect with: ssh root@192.168.15.200 -p 2222" -ForegroundColor Cyan
