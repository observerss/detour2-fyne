# Windows Testing

Smart App Control can block `Detour2.exe` before the app starts if Windows cannot establish publisher trust. This is expected for an unsigned cross-built executable, especially when it is launched directly from `\\wsl.localhost\...`.

## Recommended Local Test

Run this from Windows PowerShell, not from WSL bash:

```powershell
PowerShell -ExecutionPolicy Bypass -File \\wsl.localhost\Ubuntu-22.04\home\jingc\Projects\Personal\detour2-fyne\scripts\Prepare-WindowsLocal.ps1 -Run
```

The script expands [dist/windows-amd64/Detour2.zip](../dist/windows-amd64/Detour2.zip), copies the executable to `%LOCALAPPDATA%\Programs\Detour2`, removes the download/source marker, signs it with a CurrentUser self-signed test certificate, and starts it from a normal Windows-local path.

## Verify Trust State

```powershell
$exe = Join-Path $env:LOCALAPPDATA 'Programs\Detour2\Detour2.exe'
Get-AuthenticodeSignature $exe | Format-List Status, StatusMessage, SignerCertificate
Get-Item -LiteralPath $exe -Stream Zone.Identifier -ErrorAction SilentlyContinue
```

Expected local-test result: signature status is `Valid`, and no `Zone.Identifier` stream is printed.

If Smart App Control still blocks the self-signed binary, that is a Windows policy decision rather than an app crash. For public distribution, use a CA-issued Authenticode certificate, SignPath's open-source signing flow, Azure Trusted Signing, or Microsoft Store distribution.

## Verify Proxy Runtime

After the app opens, choose the upstream profile and start the local proxy. Then check ports and proxy behavior from Windows PowerShell:

```powershell
netstat -ano | findstr ":7776 :7777"
curl.exe --proxy socks5h://127.0.0.1:7776 https://api.ipify.org
curl.exe --proxy http://127.0.0.1:7777 https://api.ipify.org
```

Builds made with a `build.env` (see the README) start with the built-in lines and select the default one; builds without it start with an empty profile list, so configure your own upstream address and password first.

## Common Failure Modes

- `\\wsl.localhost\...\Detour2.exe` is blocked: run the prepare script and start `%LOCALAPPDATA%\Programs\Detour2\Detour2.exe` instead.
- `UnknownError` or `NotSigned` from `Get-AuthenticodeSignature`: rerun [scripts/Prepare-WindowsLocal.ps1](../scripts/Prepare-WindowsLocal.ps1) without `-NoSign`.
- No listener on `7776` or `7777`: check the app log panel for upstream connection or password errors.
