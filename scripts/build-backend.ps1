param(
    [string]$ProjectRoot,
    [string]$OutputPath
)

$ErrorActionPreference = 'Stop'

if (-not $ProjectRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
}

$backendPath = Join-Path $ProjectRoot 'backend'
if (-not $OutputPath) {
    $OutputPath = Join-Path $backendPath 'jiuin-server.exe'
}

Push-Location $backendPath
try {
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }

    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed with exit code $LASTEXITCODE" }

    & go build -trimpath '-ldflags=-s -w' -o $OutputPath ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

    Write-Host "Built $OutputPath"
} finally {
    Pop-Location
}
