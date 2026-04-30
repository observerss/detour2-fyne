# Apple Build Environment

This project now has a macOS client path and an iOS Packet Tunnel path, but Apple packaging has hard platform requirements.

## Reality Check

- macOS app signing and notarization require macOS with Xcode command line tools.
- iOS app signing, provisioning, and Network Extension Packet Tunnel packaging require macOS with Xcode.
- Alibaba Wuying Cloud Desktop is useful if it gives you a stable remote development machine for editing, Go dependency preparation, and Windows/Linux/Android work.
- If the Wuying desktop image is Windows or Linux, it cannot replace the final macOS/Xcode host for iOS signing or notarization.

Use Wuying as the control workstation, then use one of these for Apple builds:

- a physical Mac you control;
- a dedicated macOS cloud build host;
- a CI provider with macOS runners and secure signing secret storage.

## Wuying Workstation Setup

On the Wuying desktop, install the general toolchain:

```powershell
winget install Git.Git
winget install GoLang.Go
```

Then in the repo:

```powershell
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
go test ./...
```

Use this machine for code editing, normal Go checks, docs, Windows packaging, and preparing release assets. Do not store Apple app-specific passwords or cloud access keys in the repo.

## macOS Host Setup

On the macOS build host:

```bash
xcode-select --install
go install fyne.io/fyne/v2/cmd/fyne@latest
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

Run the host check:

```bash
scripts/Check-AppleBuildHost.sh
```

Command Line Tools alone can run Go checks and may build an unsigned macOS `.app`, but it is not enough for iOS. Both Fyne iOS packaging and `gomobile bind -target=ios` require a full Xcode install selected by `xcode-select`:

```bash
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
sudo xcodebuild -license accept
xcodebuild -runFirstLaunch
xcrun --sdk iphoneos --show-sdk-path
```

If Xcode is missing or not selected, typical failures are `-os=ios requires XCode`, `gomobile: -target="ios" requires Xcode`, and `xcrun: error: SDK "iphoneos" cannot be located`.

For a fresh remote Mac, the bootstrap helper installs Go in the user directory, installs `fyne` and `gomobile`, initializes gomobile, runs Go checks, and then invokes the iOS Packet Tunnel preparation flow:

```bash
DETOUR2_MAC_SOCKS_PROXY=172.20.208.30:7778 \
bash scripts/Bootstrap-AppleBuildHost.sh
```

If RubyGems is slow or blocked, skip the optional Xcode project patching gem and still validate the Go/Fyne/gomobile path:

```bash
DETOUR2_SKIP_XCODEPROJ_GEM=1 \
DETOUR2_MAC_SOCKS_PROXY=172.20.208.30:7778 \
bash scripts/Bootstrap-AppleBuildHost.sh
```

`DETOUR2_XCODEPROJ_GEM_TIMEOUT` controls how long the helper waits for `gem install xcodeproj`; `0` disables the timeout.

When syncing from Windows, execute shell scripts with `bash script.sh` or preserve executable bits. The helper scripts call `build.sh` through `bash` so remote archives copied from Windows still work.

## Apple Developer Setup

Create these identifiers in Apple Developer:

- `github.com.observerss.detour2fyne` for the containing app.
- `github.com.observerss.detour2fyne.PacketTunnel` for the Packet Tunnel extension.
- `group.github.com.observerss.detour2fyne` if you use App Group storage.

Enable capabilities:

- Network Extensions: Packet Tunnel Provider.
- App Groups, if shared storage is used.

Create matching provisioning profiles for local testing and distribution.

## macOS Client Build

```bash
./build.sh macos
export MACOS_SIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
export MACOS_NOTARY_PROFILE="detour2-notary"
scripts/Package-macOS.sh
```

Verification:

```bash
codesign --verify --deep --strict --verbose=2 Detour2.app
spctl --assess --type execute --verbose=2 Detour2.app
xcrun stapler validate Detour2.app
```

## iOS Client Build

```bash
./build.sh ios
gomobile bind -target=ios -o ios/Build/DetourTunnelBackend.xcframework ./mobile/iospacket
```

The helper script can run the build, generate the gomobile backend, stage native files, and patch an Xcode project when Ruby `xcodeproj` is available:

```bash
sudo gem install xcodeproj
IOS_APP_BUNDLE_ID=github.com.observerss.detour2fyne \
IOS_PACKET_TUNNEL_BUNDLE_ID=github.com.observerss.detour2fyne.PacketTunnel \
IOS_APP_GROUP=group.github.com.observerss.detour2fyne \
bash scripts/Prepare-iOSPacketTunnel.sh
```

Then in Xcode:

1. Confirm the containing app target and `PacketTunnel` extension target both use the expected provisioning profiles.
2. Confirm `DetourTunnelBackend.xcframework` is linked into the Packet Tunnel target.
3. Keep `GomobileBackend.swift` in the Packet Tunnel target and check the generated header for gomobile symbol names; adjust that file only if gomobile emits different Swift names.
4. Install to a real iOS device from Xcode and approve the VPN profile.

Simulator is not a reliable validation target for Packet Tunnel behavior; use a physical device.

## Secret Handling

- Do not commit Apple passwords, signing certificates, provisioning profiles, or cloud access keys.
- Use a keychain profile for notarytool when possible.
- Keep `.p12` certificates and provisioning profiles in a secure secret store if using CI.
- The cloud access key shared in chat should be rotated before any real deployment work continues.