param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$PackageArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$script = Join-Path $PSScriptRoot "package-cross.sh"
if (!(Test-Path -LiteralPath $script)) {
    throw "package-cross.sh not found: $script"
}

$bashCandidates = @(
    $env:GIT_BASH,
    "C:\Program Files\Git\bin\bash.exe",
    "C:\Program Files\Git\usr\bin\bash.exe",
    "C:\Program Files (x86)\Git\bin\bash.exe",
    "C:\Program Files (x86)\Git\usr\bin\bash.exe"
) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }

$bash = $bashCandidates | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
if ([string]::IsNullOrWhiteSpace($bash)) {
    $cmd = Get-Command bash.exe -ErrorAction SilentlyContinue
    if ($null -ne $cmd) {
        $bash = $cmd.Source
    }
}
if ([string]::IsNullOrWhiteSpace($bash)) {
    throw "Git Bash was not found. Install Git for Windows or set GIT_BASH to bash.exe."
}

if ($PackageArgs.Count -eq 0) {
    $PackageArgs = @("all")
}

Push-Location $repoRoot
try {
    & $bash $script @PackageArgs
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
