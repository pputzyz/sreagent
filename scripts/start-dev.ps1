# SREAgent Development Server Startup Script
# This script ensures consistent environment variables across restarts

$env:SREAGENT_SECRET_KEY = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
$env:SREAGENT_DEV_SKIP_SSRF_CHECK = "true"

Write-Host "Starting SREAgent server..." -ForegroundColor Green
Write-Host "  SECRET_KEY: set" -ForegroundColor Gray
Write-Host "  SSRF_CHECK: skipped" -ForegroundColor Gray

cd c:\project\sreagent
.\bin\sreagent.exe --config configs\config.yaml
