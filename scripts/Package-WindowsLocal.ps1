param(
    [string]$SourceExe,

    [string]$OutputDir,

    [string]$AppName = "Detour2",

    [switch]$NoZip
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT") {
    throw "This script must be run on Windows."
}

$repoRoot = Split-Path -Parent $PSScriptRoot

if ([string]::IsNullOrWhiteSpace($SourceExe)) {
    $SourceExe = Join-Path $repoRoot "detour2-fyne.exe"
} elseif (-not [System.IO.Path]::IsPathRooted($SourceExe)) {
    $SourceExe = Join-Path $repoRoot $SourceExe
}

if ([string]::IsNullOrWhiteSpace($OutputDir)) {
    $OutputDir = Join-Path $repoRoot "dist\windows-amd64"
} elseif (-not [System.IO.Path]::IsPathRooted($OutputDir)) {
    $OutputDir = Join-Path $repoRoot $OutputDir
}

$sourceFullPath = (Resolve-Path -LiteralPath $SourceExe).Path
$targetExe = Join-Path $OutputDir ("{0}.exe" -f $AppName)
if ([string]::Equals($sourceFullPath, [System.IO.Path]::GetFullPath($targetExe), [System.StringComparison]::OrdinalIgnoreCase)) {
    Write-Host "Using existing target exe: $targetExe"
} else {
    Copy-Item -LiteralPath $sourceFullPath -Destination $targetExe -Force
    Unblock-File -LiteralPath $targetExe -ErrorAction SilentlyContinue
    Write-Host "Copied: $sourceFullPath -> $targetExe"
}

Get-Item -LiteralPath $targetExe | Select-Object FullName, Length, LastWriteTime

if (-not $NoZip) {
    $zipPath = Join-Path $OutputDir ("{0}.zip" -f $AppName)
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }

    $stageDir = Join-Path $env:TEMP ("detour2-package-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $stageDir | Out-Null

    try {
        $stageExe = Join-Path $stageDir ([System.IO.Path]::GetFileName($targetExe))
        Copy-Item -LiteralPath $targetExe -Destination $stageExe -Force

        $readmeSource = Join-Path $repoRoot "docs\README-WindowsPackage.md"
        if (Test-Path -LiteralPath $readmeSource) {
            Copy-Item -LiteralPath $readmeSource -Destination (Join-Path $stageDir "README-Windows.md") -Force
        }

        $toolsDir = Join-Path $stageDir "tools"
        New-Item -ItemType Directory -Path $toolsDir -Force | Out-Null
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot "Sign-WindowsLocal.ps1") -Destination (Join-Path $toolsDir "Sign-WindowsLocal.ps1") -Force
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot "Sign-WindowsLocal.cmd") -Destination (Join-Path $toolsDir "Sign-WindowsLocal.cmd") -Force

        Compress-Archive -Path (Join-Path $stageDir "*") -DestinationPath $zipPath -Force
        Unblock-File -LiteralPath $zipPath -ErrorAction SilentlyContinue
        Write-Host "Packed: $zipPath"
    } finally {
        Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}