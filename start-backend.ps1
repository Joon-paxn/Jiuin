param(
    [string]$ProjectRoot
)

$ErrorActionPreference = 'Stop'

$healthUrl = 'http://127.0.0.1:8080/api/v1/health'

if (-not $ProjectRoot) {
    $ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
}

Set-Location (Join-Path $ProjectRoot 'backend')

function Test-BackendHealth {
    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $healthUrl -TimeoutSec 2 -ErrorAction Stop
        return $response.StatusCode -ge 200 -and $response.StatusCode -lt 300
    } catch {
        return $false
    }
}

function Get-BackendCachePath {
    $cacheBase = $env:LOCALAPPDATA
    if ([string]::IsNullOrWhiteSpace($cacheBase)) {
        $cacheBase = [System.IO.Path]::GetTempPath()
    }

    $cacheDirectory = Join-Path $cacheBase 'Jiuin\backend-dev'
    New-Item -ItemType Directory -Path $cacheDirectory -Force | Out-Null
    return Join-Path $cacheDirectory 'jiuin-backend.exe'
}

function Test-BackendBinaryIsStale([string]$BinaryPath) {
    $binary = Get-Item -LiteralPath $BinaryPath -ErrorAction SilentlyContinue
    if (-not $binary) {
        return $true
    }

    $inputs = @(
        Get-ChildItem -LiteralPath 'cmd', 'internal' -Recurse -File -Filter '*.go'
        Get-Item -LiteralPath 'go.mod', 'go.sum'
    )
    $latestInput = $inputs | Sort-Object LastWriteTimeUtc -Descending | Select-Object -First 1
    return $latestInput.LastWriteTimeUtc -gt $binary.LastWriteTimeUtc
}

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
if (Test-BackendHealth) {
    Write-Host "[Backend] A healthy server is already listening on $healthUrl; no second instance was started."
    return
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw '[Backend] Go was not found in PATH.'
}

$binaryPath = Get-BackendCachePath
if (Test-BackendBinaryIsStale $binaryPath) {
    Write-Host "[Backend] Building development binary (this is slower on the first run)..."
    & go build -o $binaryPath ./cmd/server
    if ($LASTEXITCODE -ne 0) {
        throw "[Backend] Build failed with exit code $LASTEXITCODE."
    }
} else {
    Write-Host "[Backend] Reusing cached development binary: $binaryPath"
}

Write-Host '[Backend] Starting server on 127.0.0.1:8080 ...'
& $binaryPath
if ($LASTEXITCODE -ne 0) {
    throw "[Backend] Server exited with code $LASTEXITCODE."
}
