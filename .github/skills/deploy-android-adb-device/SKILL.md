---
name: deploy-android-adb-device
description: 'Build, install, and verify the Detour2 Android APK on a connected ADB physical device. Use when the user asks to 部署到adb真机, 安装到手机, deploy to adb device, install APK on Android phone, rebuild Android APK, or verify Detour2 VPN APK on real device.'
argument-hint: '[optional device serial, defaults to first non-emulator device]'
---

# Deploy Android APK to ADB Device

Use this skill to build the current Detour2 Android arm64 APK, inject the Android VPN Service, install it to a connected physical ADB device, grant notification permission, launch the app, and verify the APK/device state.

## When to Use

- User says: `部署到adb真机`, `安装到手机`, `部署到手机`, `adb安装`, `deploy to adb`, `install APK on Android phone`.
- User wants the latest Android build installed after source changes.
- User wants to verify Android VPN packaging on a real device.

## Assumptions

- Workspace root: `d:\Projects\Personal\detour2-fyne` unless discovered otherwise.
- ADB path: `D:\AndroidTools\sdk\platform-tools\adb.exe`.
- Android build-tools path: `D:\AndroidTools\sdk\build-tools\35.0.0`.
- Java path: `C:\Program Files\Eclipse Adoptium\jdk-17.0.18.8-hotspot`.
- Build script: `scripts/package-cross.sh android`.
- Final APK: `dist\android-arm64\Detour2.apk`.
- Prefer a physical device over emulator. Known physical serial often used here: `691d47d7`.

## Procedure

### 1. Check clean context and connected devices

Run in PowerShell from the workspace root:

```powershell
Set-Location 'd:\Projects\Personal\detour2-fyne'
$adb = 'D:\AndroidTools\sdk\platform-tools\adb.exe'
if (!(Test-Path $adb)) { throw "ADB not found: $adb" }
& $adb devices
```

Select the first line matching `^\S+\s+device$` that does not start with `emulator-`. Do not use `$devices[0]` if `$devices` may be a string; it can select only the first character. Use array wrapping:

```powershell
$deviceLines = @(& $adb devices | Where-Object { $_ -match '^\S+\s+device$' -and $_ -notmatch '^emulator-' })
if ($deviceLines.Count -lt 1) { throw 'No physical ADB device found' }
$serial = ((@($deviceLines)[0] -split '\s+')[0])
Write-Host "selected device: $serial"
```

### 2. Build latest Android arm64 APK

Prefer Windows Git Bash because WSL `/mnt/d` can be unavailable in this environment.

```powershell
Set-Location 'd:\Projects\Personal\detour2-fyne'
& 'C:\Program Files\Git\bin\bash.exe' scripts/package-cross.sh android
```

Expected behavior:

- Fyne-cross builds `android-arm64`.
- Docker injector may fail with Docker Desktop mount issues.
- The script should automatically fall back to `scripts/Inject-AndroidVpnServiceLocal.ps1`.
- Final APK should be copied to `dist\android-arm64\Detour2.apk`.

If build fails with a Go compile error, fix the source, run `gofmt`, then rebuild. Do not install an old APK after a failed build.

### 3. Install to the physical device and grant notifications

```powershell
Set-Location 'd:\Projects\Personal\detour2-fyne'
$adb = 'D:\AndroidTools\sdk\platform-tools\adb.exe'
$apk = 'dist\android-arm64\Detour2.apk'
$deviceLines = @(& $adb devices | Where-Object { $_ -match '^\S+\s+device$' -and $_ -notmatch '^emulator-' })
if ($deviceLines.Count -lt 1) { throw 'No physical ADB device found' }
$serial = ((@($deviceLines)[0] -split '\s+')[0])
& $adb -s $serial install -r $apk
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& $adb -s $serial shell pm grant github.com.observerss.detour2fyne android.permission.POST_NOTIFICATIONS 2>$null
& $adb -s $serial shell dumpsys package github.com.observerss.detour2fyne | Select-String 'POST_NOTIFICATIONS' -Context 0,1
```

Expected install output includes `Success` and notification permission `granted=true` when supported.

### 4. Launch and verify device/package state

```powershell
$adb = 'D:\AndroidTools\sdk\platform-tools\adb.exe'
$serial = '<selected-serial>'
& $adb -s $serial shell monkey -p github.com.observerss.detour2fyne 1
& $adb -s $serial shell dumpsys package github.com.observerss.detour2fyne | Select-String 'versionCode|versionName|DetourVpnService' -Context 0,1
```

Expected:

- `Events injected: 1`
- `versionCode=80` or newer
- `targetSdk=34`
- `DetourVpnService` appears with `android.permission.BIND_VPN_SERVICE`

### 5. Verify local APK contents

```powershell
Set-Location 'd:\Projects\Personal\detour2-fyne'
& 'D:\AndroidTools\sdk\build-tools\35.0.0\aapt.exe' dump badging 'dist\android-arm64\Detour2.apk' | Select-String 'package:|sdkVersion|targetSdkVersion|native-code'
Add-Type -AssemblyName System.IO.Compression.FileSystem
$zip = [System.IO.Compression.ZipFile]::OpenRead((Resolve-Path 'dist\android-arm64\Detour2.apk'))
try { Write-Host ('classes2.dex present: ' + [bool]$zip.GetEntry('classes2.dex')) } finally { $zip.Dispose() }
```

Expected:

- `targetSdkVersion:'34'`
- `native-code: 'arm64-v8a'`
- `classes2.dex present: True`

## Common Pitfalls

- **WSL chdir failure**: If `bash scripts/package-cross.sh android` fails with `/mnt/d` errors, use `C:\Program Files\Git\bin\bash.exe` from PowerShell.
- **Docker mount failure**: This is expected sometimes; `package-cross.sh` should fall back to the Windows local injector.
- **Missing import compile error**: Fix source and rerun `gofmt`; never deploy a stale APK after failed build.
- **Multiple devices**: Prefer physical serial, exclude `emulator-*` unless the user explicitly asks for emulator.
- **PowerShell string indexing**: Always array-wrap device lines with `@(...)` before selecting `[0]`.
- **Notifications not visible**: Grant `POST_NOTIFICATIONS` and verify notification channel/package settings if needed.

## Completion Checklist

- Physical ADB device selected.
- Latest `dist\android-arm64\Detour2.apk` built successfully.
- APK installed with `Success`.
- `POST_NOTIFICATIONS` granted when supported.
- App launched with `monkey`.
- `DetourVpnService`, `targetSdk=34`, `arm64-v8a`, and `classes2.dex` verified.
- If any build-source fix was needed, format, test relevant files, and commit the fix.
