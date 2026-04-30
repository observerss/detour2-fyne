param(
    [string]$ZipPath,

    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "Programs\Detour2"),

    [switch]$NoSign,

    [switch]$Run
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT") {
    throw "This script must be run on Windows."
}

$repoRoot = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($ZipPath)) {
    $ZipPath = Join-Path $repoRoot "dist\windows-amd64\Detour2.zip"
} elseif (-not [System.IO.Path]::IsPathRooted($ZipPath)) {
    $ZipPath = Join-Path $repoRoot $ZipPath
}

$zipFullPath = (Resolve-Path -LiteralPath $ZipPath).Path
Unblock-File -LiteralPath $zipFullPath -ErrorAction SilentlyContinue

$tempDir = Join-Path $env:TEMP ("detour2-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    Expand-Archive -LiteralPath $zipFullPath -DestinationPath $tempDir -Force
    $sourceExe = Get-ChildItem -LiteralPath $tempDir -Filter "Detour2.exe" -Recurse |
        Select-Object -First 1
    if ($null -eq $sourceExe) {
        throw "Detour2.exe was not found in $zipFullPath"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $targetExe = Join-Path $InstallDir "Detour2.exe"
    Copy-Item -LiteralPath $sourceExe.FullName -Destination $targetExe -Force
    Unblock-File -LiteralPath $targetExe -ErrorAction SilentlyContinue

    if (-not $NoSign) {
        $signScript = Join-Path $PSScriptRoot "Sign-WindowsLocal.ps1"
        & $signScript -ExePath $targetExe
    }

    Write-Host "Ready: $targetExe"
    Get-AuthenticodeSignature -FilePath $targetExe | Format-List Status, StatusMessage, SignerCertificate

    if ($Run) {
        Start-Process -FilePath $targetExe -WorkingDirectory $InstallDir
    }
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
