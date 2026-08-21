param(
    [string]$ProjectRoot
)

$ErrorActionPreference = 'Stop'

if (-not $ProjectRoot) {
    $ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
}

$backendRoot = Join-Path $ProjectRoot 'backend'
$healthUrl = 'http://127.0.0.1:8080/api/v1/health'

function Set-DefaultEnvironment([string]$Name, [string]$Value) {
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($Name, 'Process'))) {
        [Environment]::SetEnvironmentVariable($Name, $Value, 'Process')
    }
}

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
    return Join-Path $cacheDirectory 'jiuin-go.exe'
}

function Test-BackendBinaryIsStale([string]$BinaryPath) {
    $binary = Get-Item -LiteralPath $BinaryPath -ErrorAction SilentlyContinue
    if (-not $binary) {
        return $true
    }
    $inputs = @(
        Get-ChildItem -LiteralPath (Join-Path $backendRoot 'cmd'), (Join-Path $backendRoot 'internal') -Recurse -File
        Get-Item -LiteralPath (Join-Path $backendRoot 'go.mod'), (Join-Path $backendRoot 'go.sum')
    )
    $latestInput = $inputs | Sort-Object LastWriteTimeUtc -Descending | Select-Object -First 1
    return $latestInput.LastWriteTimeUtc -gt $binary.LastWriteTimeUtc
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw '[Backend] Go was not found in PATH.'
}

$developmentStorage = Join-Path $ProjectRoot '.local\music'
New-Item -ItemType Directory -Path $developmentStorage -Force | Out-Null
Set-DefaultEnvironment 'JIUIN_ENV' 'development'
Set-DefaultEnvironment 'JIUIN_GO_LISTEN_ADDR' '127.0.0.1:8080'
Set-DefaultEnvironment 'JIUIN_STORAGE_DIR' $developmentStorage
Set-DefaultEnvironment 'JIUIN_DATABASE_PATH' (Join-Path $developmentStorage 'music.db')
Set-DefaultEnvironment 'JIUIN_WS_ALLOWED_ORIGINS' 'http://127.0.0.1:5173,http://localhost:5173'

if (Test-BackendHealth) {
    Write-Host "[Backend] A healthy server is already listening on $healthUrl; no second instance was started."
    return
}

$binaryPath = Get-BackendCachePath
Push-Location $backendRoot
try {
    if (Test-BackendBinaryIsStale $binaryPath) {
        Write-Host '[Backend] Building development binary...'
        & go build -o $binaryPath ./cmd/jiuin-go
        if ($LASTEXITCODE -ne 0) {
            throw "[Backend] Build failed with exit code $LASTEXITCODE."
        }
    } else {
        Write-Host "[Backend] Reusing cached development binary: $binaryPath"
    }
    Write-Host '[Backend] Starting Go backup API and WebSocket on 127.0.0.1:8080 ...'
    & $binaryPath
    if ($LASTEXITCODE -ne 0) {
        throw "[Backend] Server exited with code $LASTEXITCODE."
    }
} finally {
    Pop-Location
}
