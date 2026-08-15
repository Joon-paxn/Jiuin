param(
    [string]$ProjectRoot,
    [string]$OutputDirectory,
    [Parameter(Mandatory = $true)]
    [string]$FFmpegPath,
    [Parameter(Mandatory = $true)]
    [string]$FFprobePath,
    [ValidateSet('windows-amd64', 'linux-amd64')]
    [string]$Target = 'windows-amd64'
)

$ErrorActionPreference = 'Stop'

if (-not $ProjectRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
}
if (-not $OutputDirectory) {
    $OutputDirectory = Join-Path $ProjectRoot ("release\jiuin-backend-$Target")
}

$backendPath = Join-Path $ProjectRoot 'backend'
$ffmpeg = (Resolve-Path -LiteralPath $FFmpegPath -ErrorAction Stop).Path
$ffprobe = (Resolve-Path -LiteralPath $FFprobePath -ErrorAction Stop).Path
if (-not (Test-Path -LiteralPath $ffmpeg -PathType Leaf) -or -not (Test-Path -LiteralPath $ffprobe -PathType Leaf)) {
    throw 'FFmpegPath and FFprobePath must point to executable files.'
}

New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $OutputDirectory 'storage\music') -Force | Out-Null

Push-Location $backendPath
try {
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed with exit code $LASTEXITCODE" }

    $binaryName = if ($Target -eq 'windows-amd64') { 'jiuin-server.exe' } else { 'jiuin-server' }
    $env:GOOS = if ($Target -eq 'windows-amd64') { 'windows' } else { 'linux' }
    $env:GOARCH = 'amd64'
    & go build -trimpath '-ldflags=-s -w' -o (Join-Path $OutputDirectory $binaryName) ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
} finally {
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}

$ffmpegName = if ($Target -eq 'windows-amd64') { 'ffmpeg.exe' } else { 'ffmpeg' }
$ffprobeName = if ($Target -eq 'windows-amd64') { 'ffprobe.exe' } else { 'ffprobe' }
Copy-Item -LiteralPath $ffmpeg -Destination (Join-Path $OutputDirectory $ffmpegName) -Force
Copy-Item -LiteralPath $ffprobe -Destination (Join-Path $OutputDirectory $ffprobeName) -Force

if ($Target -eq 'windows-amd64') {
    $runtimeDirectory = Split-Path -Parent $ffmpeg
    Get-ChildItem -LiteralPath $runtimeDirectory -File -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -ne (Split-Path -Leaf $ffmpeg) -and $_.Name -ne (Split-Path -Leaf $ffprobe) } |
        Copy-Item -Destination $OutputDirectory -Force
}

Copy-Item -LiteralPath (Join-Path $backendPath 'README.md') -Destination (Join-Path $OutputDirectory 'README.backend.md') -Force
if ($Target -eq 'linux-amd64') {
    Copy-Item -LiteralPath (Join-Path $ProjectRoot 'scripts\start-backend-linux.sh') -Destination (Join-Path $OutputDirectory 'start-backend-linux.sh') -Force
}
$environmentTemplate = Get-Content -LiteralPath (Join-Path $backendPath 'configs\production.env.example') -Encoding utf8
$environmentTemplate = $environmentTemplate -replace '^JIUIN_MUSIC_DIRECTORY=.*$', 'JIUIN_MUSIC_DIRECTORY=storage/music'
$environmentTemplate = $environmentTemplate -replace '^JIUIN_FFMPEG_PATH=.*$', "JIUIN_FFMPEG_PATH=./$ffmpegName"
$environmentTemplate = $environmentTemplate -replace '^JIUIN_FFPROBE_PATH=.*$', "JIUIN_FFPROBE_PATH=./$ffprobeName"
$environmentTemplate | Set-Content -LiteralPath (Join-Path $OutputDirectory 'backend.env.example') -Encoding utf8

Write-Host "Backend package created: $OutputDirectory"
