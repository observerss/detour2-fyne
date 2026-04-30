param(
    [string]$ApkPath = "dist/android-amd64/Detour2.apk",
    [string]$AndroidToolsRoot = "D:/AndroidTools",
    [string]$JdkRoot = "C:/Program Files/Eclipse Adoptium/jdk-17.0.18.8-hotspot"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$apkFullPath = (Resolve-Path (Join-Path $repoRoot $ApkPath)).Path
$sdk = Join-Path $AndroidToolsRoot "sdk"
$buildTools = Get-ChildItem (Join-Path $sdk "build-tools") -Directory | Sort-Object Name -Descending | Select-Object -First 1
if ($null -eq $buildTools) { throw "Android build-tools not found" }
$androidJar = Join-Path $sdk "platforms/android-34/android.jar"
if (!(Test-Path $androidJar)) {
    $androidJar = Get-ChildItem (Join-Path $sdk "platforms") -Filter android.jar -Recurse | Sort-Object FullName -Descending | Select-Object -ExpandProperty FullName -First 1
}
if (!(Test-Path $androidJar)) { throw "android.jar not found" }

$javac = Join-Path $JdkRoot "bin/javac.exe"
$keytool = Join-Path $JdkRoot "bin/keytool.exe"
$java = Join-Path $JdkRoot "bin/java.exe"
foreach ($tool in @($javac, $keytool, $java)) {
    if (!(Test-Path $tool)) { throw "Missing JDK tool: $tool" }
}
$env:JAVA_HOME = $JdkRoot
$env:PATH = (Join-Path $JdkRoot "bin") + [System.IO.Path]::PathSeparator + $env:PATH

$aapt2 = Join-Path $buildTools.FullName "aapt2.exe"
$d8Jar = Join-Path $buildTools.FullName "lib/d8.jar"
$zipalign = Join-Path $buildTools.FullName "zipalign.exe"
$apksigner = Join-Path $buildTools.FullName "apksigner.bat"
foreach ($tool in @($aapt2, $d8Jar, $zipalign, $apksigner)) {
    if (!(Test-Path $tool)) { throw "Missing Android tool: $tool" }
}

$work = Join-Path $repoRoot ".android-vpn-build/local"
$classesDir = Join-Path $work "classes"
$dexDir = Join-Path $work "dex"
Remove-Item $work -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $classesDir, $dexDir | Out-Null
$localAndroidJar = Join-Path $work "android.jar"
Copy-Item $androidJar $localAndroidJar -Force

$src = Join-Path $repoRoot "android/src/main/java/github/com/observerss/detour2fyne/DetourVpnService.java"
& $javac -encoding UTF-8 -source 8 -target 8 -bootclasspath $androidJar -d $classesDir $src
if ($LASTEXITCODE -ne 0) { throw "javac failed" }

$classFiles = @(Get-ChildItem $classesDir -Filter *.class -Recurse | ForEach-Object {
    $_.FullName.Substring($work.Length + 1)
})
Push-Location $work
try {
    & (Join-Path $JdkRoot "bin/java.exe") -cp $d8Jar com.android.tools.r8.D8 --lib android.jar --min-api 23 --output dex @classFiles
} finally {
    Pop-Location
}
if ($LASTEXITCODE -ne 0) { throw "d8 failed" }
Move-Item (Join-Path $dexDir "classes.dex") (Join-Path $dexDir "classes2.dex") -Force

$manifestSrc = Join-Path $repoRoot "AndroidManifest.xml"
$patchedManifest = Join-Path $work "AndroidManifest.xml"
$manifestRaw = Get-Content $manifestSrc -Raw
$manifestRaw = $manifestRaw.Replace(
    "android:label=`"Detour2`">",
    "android:label=`"Detour2`"`n`t`tandroid:extractNativeLibs=`"false`">"
)
$patchedManifestContent = $manifestRaw.Replace(
    'android:permission="android.permission.BIND_VPN_SERVICE"',
    "android:permission=`"android.permission.BIND_VPN_SERVICE`"`n`t`t`tandroid:foregroundServiceType=`"systemExempted`""
)
[System.IO.File]::WriteAllText($patchedManifest, $patchedManifestContent, [System.Text.UTF8Encoding]::new($false))
$manifestApk = Join-Path $work "manifest.apk"
& $aapt2 link --manifest $patchedManifest -I $androidJar --min-sdk-version 15 --target-sdk-version 34 -o $manifestApk
if ($LASTEXITCODE -ne 0) { throw "aapt2 link failed" }

$manifestZip = [System.IO.Compression.ZipFile]::OpenRead($manifestApk)
try {
    $entry = $manifestZip.GetEntry("AndroidManifest.xml")
    if ($null -eq $entry) { throw "Compiled manifest not found" }
    [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, (Join-Path $work "AndroidManifest.bin"), $true)
} finally {
    $manifestZip.Dispose()
}

$unsigned = Join-Path $work "Detour2-unsigned.apk"
$aligned = Join-Path $work "Detour2-aligned.apk"
$signed = Join-Path $work "Detour2-signed.apk"
Copy-Item $apkFullPath $unsigned -Force

$zip = [System.IO.Compression.ZipFile]::Open($unsigned, [System.IO.Compression.ZipArchiveMode]::Update)
try {
    @($zip.Entries | Where-Object { $_.FullName -like 'META-INF/*' -or $_.FullName -eq 'AndroidManifest.xml' -or $_.FullName -eq 'classes2.dex' }) | ForEach-Object { $_.Delete() }
    [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, (Join-Path $work "AndroidManifest.bin"), "AndroidManifest.xml", [System.IO.Compression.CompressionLevel]::Optimal) | Out-Null
    [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, (Join-Path $dexDir "classes2.dex"), "classes2.dex", [System.IO.Compression.CompressionLevel]::Optimal) | Out-Null
} finally {
    $zip.Dispose()
}

& $zipalign -f -P 16 4 $unsigned $aligned
if ($LASTEXITCODE -ne 0) { throw "zipalign failed" }

$ksDir = Join-Path $AndroidToolsRoot "keystores"
$ks = Join-Path $ksDir "detour2-debug.keystore"
New-Item -ItemType Directory -Force -Path $ksDir | Out-Null
if (!(Test-Path $ks)) {
    & $keytool -genkeypair -keystore $ks -storepass android -keypass android -alias detour2 -keyalg RSA -keysize 2048 -validity 10000 -dname "CN=Detour2 Debug,O=Detour2,C=CN" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "keytool failed" }
}

& $apksigner sign --ks $ks --ks-pass pass:android --key-pass pass:android --ks-key-alias detour2 --out $signed $aligned
if ($LASTEXITCODE -ne 0) { throw "apksigner sign failed" }
Copy-Item $signed $apkFullPath -Force
& $zipalign -c -P 16 4 $apkFullPath
if ($LASTEXITCODE -ne 0) { throw "zipalign check failed" }
& $apksigner verify --verbose $apkFullPath
if ($LASTEXITCODE -ne 0) { throw "apksigner verify failed" }

Write-Host "Injected Android VPN Service into $apkFullPath"
Get-Item $apkFullPath | Select-Object FullName,Length,LastWriteTime
Get-FileHash $apkFullPath -Algorithm SHA256
