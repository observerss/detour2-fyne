# Detour2 Windows Portable Package

This package is a portable Windows build. Unzip it to a local folder, then run `Detour2.exe`.

## Recommended use

- Extract the zip before running the app.
- Do not run the app directly from `\\wsl.localhost\...`, a network share, or inside the zip archive.
- For normal public distribution, use a real Authenticode code-signing certificate from a trusted CA and timestamp the signature.

## Local test signing tool

The `tools\Sign-WindowsLocal.ps1` script is included only for local developer/testing machines.

It creates or reuses a self-signed code-signing certificate under the current Windows user, imports that certificate into the current user's `Root` and `TrustedPublisher` stores, and signs the selected executable.

Example:

```powershell
PowerShell -ExecutionPolicy Bypass -File .\tools\Sign-WindowsLocal.ps1 -ExePath .\Detour2.exe
```

Optional re-zip after signing:

```powershell
PowerShell -ExecutionPolicy Bypass -File .\tools\Sign-WindowsLocal.ps1 -ExePath .\Detour2.exe -Rezip -ZipPath .\Detour2-local-signed.zip
```

## Important warning

Do not ask ordinary users to run the local test signing tool. A self-signed certificate does not establish public trust and should not be treated as a release signature.

If Windows SmartScreen or Smart App Control blocks this build, use a properly signed release, Microsoft Store distribution, or another trusted release channel.
