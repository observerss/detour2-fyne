param(
    [Parameter(Mandatory = $true)]
    [string]$ExePath,

    [string]$Subject = "CN=Detour2 Local Test",

    [switch]$Rezip,

    [string]$ZipPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT") {
    throw "This script must be run on Windows."
}

$resolvedExe = Resolve-Path -LiteralPath $ExePath
$exeFullPath = $resolvedExe.Path

Unblock-File -LiteralPath $exeFullPath -ErrorAction SilentlyContinue

$cert = Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert |
    Where-Object { $_.Subject -eq $Subject } |
    Sort-Object NotAfter -Descending |
    Select-Object -First 1

if ($null -eq $cert) {
    $cert = New-SelfSignedCertificate `
        -Type CodeSigningCert `
        -Subject $Subject `
        -KeyUsage DigitalSignature `
        -CertStoreLocation Cert:\CurrentUser\My `
        -NotAfter (Get-Date).AddYears(2)
}

$certFile = Join-Path $env:TEMP "detour2-local-test-code-signing.cer"
Export-Certificate -Cert $cert -FilePath $certFile | Out-Null
Import-Certificate -FilePath $certFile -CertStoreLocation Cert:\CurrentUser\Root | Out-Null
Import-Certificate -FilePath $certFile -CertStoreLocation Cert:\CurrentUser\TrustedPublisher | Out-Null

$signature = Set-AuthenticodeSignature -FilePath $exeFullPath -Certificate $cert -HashAlgorithm SHA256
if ($signature.Status -ne "Valid") {
    throw "Signing failed: $($signature.Status) $($signature.StatusMessage)"
}

Write-Host "Signed: $exeFullPath"
Get-AuthenticodeSignature -FilePath $exeFullPath | Format-List Status, StatusMessage, SignerCertificate

if ($Rezip) {
    if ([string]::IsNullOrWhiteSpace($ZipPath)) {
        $zipName = [System.IO.Path]::GetFileNameWithoutExtension($exeFullPath) + "-local-signed.zip"
        $ZipPath = Join-Path ([System.IO.Path]::GetDirectoryName($exeFullPath)) $zipName
    }
    Compress-Archive -LiteralPath $exeFullPath -DestinationPath $ZipPath -Force
    Unblock-File -LiteralPath $ZipPath -ErrorAction SilentlyContinue
    Write-Host "Packed: $ZipPath"
}
