param(
    [string]$ProjectRoot
)

$ErrorActionPreference = 'Stop'

if (-not $ProjectRoot) {
    $ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
}

Set-Location (Join-Path $ProjectRoot 'backend')

$configPath = 'configs\development.env'
if (-not (Test-Path -LiteralPath $configPath)) {
    $configPath = 'configs\development.env.example'
}

Write-Host "[Backend] Using config: $configPath"

Get-Content -LiteralPath $configPath -Encoding UTF8 | ForEach-Object {
    if ($_ -match '^(?<key>[^=]+)=(?<value>.*)$') {
        [Environment]::SetEnvironmentVariable($matches.key, $matches.value, 'Process')
    }
}

Write-Host "[Backend] Environment variables loaded."
Write-Host "[Backend] Starting server on 127.0.0.1:8080 ..."

go run ./cmd/server
