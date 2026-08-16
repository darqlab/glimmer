# Downloads and installs the latest (or a specific) glimmer release for Windows.
#
# Usage:
#   iwr -useb https://raw.githubusercontent.com/darqlab/glimmer/main/install.ps1 | iex
#   $env:GLIMMER_VERSION = "v0.2.0"; .\install.ps1 -Dest "C:\Tools\glimmer"
#
# Params / env vars:
#   -Version / $env:GLIMMER_VERSION   release tag to install (default: latest)
#   -Dest                             install directory (default: %LOCALAPPDATA%\Programs\glimmer)

param(
    [string]$Version = $(if ($env:GLIMMER_VERSION) { $env:GLIMMER_VERSION } else { "latest" }),
    [string]$Dest = "$env:LOCALAPPDATA\Programs\glimmer"
)

$ErrorActionPreference = "Stop"

$repo  = "darqlab/glimmer"
$asset = "glimmer-windows-amd64.zip"

if ($Version -eq "latest") {
    $apiUrl = "https://api.github.com/repos/$repo/releases/latest"
} else {
    $apiUrl = "https://api.github.com/repos/$repo/releases/tags/$Version"
}

Write-Host "==> resolving $Version release for $asset"
$release = Invoke-RestMethod -Uri $apiUrl -Headers @{ "User-Agent" = "glimmer-installer" }
$downloadUrl = ($release.assets | Where-Object { $_.name -eq $asset } | Select-Object -First 1).browser_download_url

if (-not $downloadUrl) {
    Write-Error "glimmer: could not find asset '$asset' in release '$Version' of $repo"
    exit 1
}

$tmp = New-Item -ItemType Directory -Path (Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid()))
$zipPath = Join-Path $tmp.FullName $asset

try {
    Write-Host "==> downloading $downloadUrl"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath

    Write-Host "==> extracting"
    Expand-Archive -Path $zipPath -DestinationPath $tmp.FullName -Force

    New-Item -ItemType Directory -Path $Dest -Force | Out-Null
    Copy-Item -Path (Join-Path $tmp.FullName "glimmer.exe") -Destination (Join-Path $Dest "glimmer.exe") -Force

    Write-Host "==> installed $Dest\glimmer.exe"

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$Dest*") {
        Write-Host "==> adding $Dest to your user PATH (restart your terminal to take effect)"
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$Dest", "User")
    }
} finally {
    Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
}
