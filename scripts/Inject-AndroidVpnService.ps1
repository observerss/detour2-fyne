# Android:
#   scripts/Inject-AndroidVpnService.ps1       Docker-based injector used by package-cross.sh.
#   scripts/Inject-AndroidVpnServiceLocal.ps1  Windows-local fallback when Docker Desktop volume mounts fail.
#
param(
    [string]$ApkPath = "dist/android-arm64/Detour2.apk",
    [string]$DockerImage = "fyneio/fyne-cross-images:android",
    [string]$AndroidToolsRoot = "D:/AndroidTools"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$apkFullPath = Resolve-Path (Join-Path $repoRoot $ApkPath)
$androidToolsFullPath = $AndroidToolsRoot
New-Item -ItemType Directory -Force -Path (Join-Path $androidToolsFullPath "keystores") | Out-Null

$repoDockerPath = ($repoRoot.Path -replace '\\', '/')
$toolsDockerPath = ($androidToolsFullPath -replace '\\', '/')
$apkDockerPath = "/workspace/$($ApkPath -replace '\\', '/')"
$hostBuildDir = Join-Path $repoRoot ".android-vpn-build"
$hostScriptPath = Join-Path $hostBuildDir "inject.sh"
New-Item -ItemType Directory -Force -Path $hostBuildDir | Out-Null

$script = @'
set -eu

SDK="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-/usr/local/android_sdk}}"
API="${DETOUR2_ANDROID_API:-34}"
# newest build-tools by version (dir name), not by full path
BT="$(find "$SDK/build-tools" "/android-tools/sdk/build-tools" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | awk -F/ '{print $NF "\t" $0}' | sort -V | tail -1 | cut -f2)"
ANDROID_JAR="$(find "/android-tools/sdk/platforms/android-${API}" "$SDK/platforms/android-${API}" -name android.jar 2>/dev/null | head -1)"
if [ -z "$ANDROID_JAR" ]; then
  ANDROID_JAR="$(find "/android-tools/sdk/platforms" "$SDK/platforms" -name android.jar 2>/dev/null | sort -V | tail -1)"
fi
WORK="/workspace/.android-vpn-build/work"
SRC="/workspace/android/src/main/java/github/com/observerss/detour2fyne/DetourVpnService.java"
MANIFEST_SRC="/workspace/AndroidManifest.xml"
APK="__APK_PATH__"
KS="/android-tools/keystores/detour2-debug.keystore"
SIGNED="$WORK/Detour2-signed.apk"
ALIGNED="$WORK/Detour2-aligned.apk"
UNSIGNED="$WORK/Detour2-unsigned.apk"
PATCHED_MANIFEST="$WORK/AndroidManifest.xml"
MANIFEST_APK="$WORK/manifest.apk"

if [ ! -f "$APK" ]; then
  echo "APK not found: $APK" >&2
  exit 2
fi
if [ -z "$BT" ] || [ ! -x "$BT/aapt2" ] || [ ! -x "$BT/d8" ] || [ ! -x "$BT/apksigner" ] || [ ! -x "$BT/zipalign" ]; then
  echo "Android build-tools aapt2/d8/apksigner/zipalign not found under $SDK" >&2
  exit 3
fi
if [ -z "$ANDROID_JAR" ]; then
  echo "android.jar not found under $SDK/platforms" >&2
  exit 4
fi
export LD_LIBRARY_PATH="$SDK/ndk-bundle/toolchains/llvm/prebuilt/linux-x86_64/lib64:${LD_LIBRARY_PATH:-}"
if ! command -v javac >/dev/null 2>&1; then
  echo "javac not found in Docker image" >&2
  exit 5
fi
if ! command -v zip >/dev/null 2>&1; then
  echo "zip not found in Docker image" >&2
  exit 6
fi

rm -rf "$WORK"
mkdir -p "$WORK/classes" "$WORK/dex" "$(dirname "$KS")"

javac -encoding UTF-8 -source 8 -target 8 -bootclasspath "$ANDROID_JAR" -d "$WORK/classes" "$SRC"
"$BT/d8" --lib "$ANDROID_JAR" --min-api 23 --output "$WORK/dex" $(find "$WORK/classes" -name '*.class' | sort)
mv "$WORK/dex/classes.dex" "$WORK/dex/classes2.dex"

sed \
  -e '0,/android:label="Detour2">/s/android:label="Detour2">/android:label="Detour2"\n\t\tandroid:extractNativeLibs="false">/' \
  -e 's/android:permission="android.permission.BIND_VPN_SERVICE"/android:permission="android.permission.BIND_VPN_SERVICE"\n\t\t\tandroid:foregroundServiceType="systemExempted"/' \
  "$MANIFEST_SRC" > "$PATCHED_MANIFEST"
"$BT/aapt2" link \
  --manifest "$PATCHED_MANIFEST" \
  -I "$ANDROID_JAR" \
  --min-sdk-version 15 \
  --target-sdk-version 34 \
  -o "$MANIFEST_APK"
unzip -p "$MANIFEST_APK" AndroidManifest.xml > "$WORK/AndroidManifest.xml"

cp "$APK" "$UNSIGNED"
zip -q -d "$UNSIGNED" 'META-INF/*' AndroidManifest.xml classes2.dex 2>/dev/null || true
(cd "$WORK" && zip -q -j "$UNSIGNED" AndroidManifest.xml)
(cd "$WORK/dex" && zip -q -j "$UNSIGNED" classes2.dex)

"$BT/zipalign" -f -P 16 4 "$UNSIGNED" "$ALIGNED"

if [ ! -f "$KS" ]; then
  keytool -genkeypair \
    -keystore "$KS" \
    -storepass android \
    -keypass android \
    -alias detour2 \
    -keyalg RSA \
    -keysize 2048 \
    -validity 10000 \
    -dname "CN=Detour2 Debug,O=Detour2,C=CN" >/dev/null
fi

"$BT/apksigner" sign \
  --ks "$KS" \
  --ks-pass pass:android \
  --key-pass pass:android \
  --ks-key-alias detour2 \
  --out "$SIGNED" \
  "$ALIGNED"

cp "$SIGNED" "$APK"
"$BT/zipalign" -c -P 16 4 "$APK"
"$BT/apksigner" verify --verbose "$APK"
unzip -l "$APK" | grep -q 'classes2.dex'
printf 'Injected DetourVpnService into %s\n' "$APK"
'@

$script = $script.Replace("__APK_PATH__", $apkDockerPath)
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($hostScriptPath, $script.Replace("`r`n", "`n"), $utf8NoBom)

docker run --rm `
    -v "${repoDockerPath}:/workspace" `
    -v "${toolsDockerPath}:/android-tools" `
    -w /workspace `
    $DockerImage `
  sh /workspace/.android-vpn-build/inject.sh

if ($LASTEXITCODE -ne 0) {
  throw "Android VPN Service injection failed with exit code $LASTEXITCODE"
}

Remove-Item -LiteralPath $hostBuildDir -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "Injected Android VPN Service into $apkFullPath"
