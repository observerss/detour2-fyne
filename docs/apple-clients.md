# Apple Clients

This project can package the Fyne UI for macOS and iOS, but the two Apple platforms have very different rules for global proxy support.

## macOS

The macOS client uses the system `networksetup` tool to apply the local Detour2 proxy globally:

- HTTP and HTTPS traffic use the app's local HTTP proxy port.
- SOCKS traffic uses the app's local SOCKS5 proxy port.
- `localhost`, `127.0.0.1`, and `::1` are bypassed so the app does not proxy itself.
- The first enable operation stores a per-user snapshot of the existing Web, Secure Web, SOCKS, and bypass-domain settings, and disable restores that snapshot when available.
- If the app cannot change network settings as the current user, macOS shows an administrator authorization prompt.

Build on macOS:

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
./build.sh macos
```

After launch, save or select an upstream profile, start the local proxy, then enable global proxy on the Run tab.

### Signing and Notarization

Developer ID distribution requires signing and notarization on macOS with Xcode command line tools. The helper script is [scripts/Package-macOS.sh](../scripts/Package-macOS.sh).

Example using a notarytool keychain profile:

```bash
xcrun notarytool store-credentials detour2-notary \
	--apple-id "you@example.com" \
	--team-id "TEAMID" \
	--password "app-specific-password"

export MACOS_SIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
export MACOS_NOTARY_PROFILE="detour2-notary"
scripts/Package-macOS.sh
```

Example without a stored profile:

```bash
export MACOS_SIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
export APPLE_ID="you@example.com"
export APPLE_TEAM_ID="TEAMID"
export APPLE_APP_SPECIFIC_PASSWORD="app-specific-password"
scripts/Package-macOS.sh
```

For local unsigned packaging only:

```bash
MACOS_SKIP_SIGN=1 MACOS_SKIP_NOTARIZE=1 scripts/Package-macOS.sh
```

## iOS

iOS does not allow a normal app to set global proxy settings. A usable iOS global-proxy client must be distributed with an Apple Network Extension Packet Tunnel Provider.

Required Apple-side prerequisites:

- Apple Developer Program membership.
- App ID with `Network Extensions` capability and `Packet Tunnel` enabled.
- A separate Packet Tunnel Provider extension bundle ID, usually `<app bundle id>.PacketTunnel`.
- Matching provisioning profiles for the main app and the extension.
- An App Group if the main Fyne app will share profile data with the tunnel extension.

Included iOS engineering pieces:

- Add an Xcode Packet Tunnel Provider target to the packaged iOS app.
- The packet forwarding runtime runs in the extension process so iOS can suspend the main app while the tunnel stays active.
- `NEPacketTunnelFlow` is bridged to the Go/gomobile backend, which starts Detour2 local SOCKS5 and tun2socks inside the extension.
- The selected Detour2 upstream profile is passed into `NETunnelProviderProtocol.providerConfiguration` from the Fyne UI.
- Sign and install the app with the Network Extension entitlement.

The scaffold for this is in [ios/README.md](../ios/README.md). It includes the main-app tunnel manager, Packet Tunnel provider, entitlement templates, a Swift `NEPacketTunnelFlow` adapter, conditional gomobile backend wiring, the Go/gomobile backend under [mobile/iospacket](../mobile/iospacket), and an iOS cgo bridge in [run/proxy/proxy_ios.go](../run/proxy/proxy_ios.go) that creates the `NETunnelProviderManager` profile from the Fyne UI.

Build the current iOS UI shell on a macOS host:

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
./build.sh ios
```

Alibaba Wuying Cloud Desktop can be used as a remote development machine only if the chosen desktop image provides the required macOS/Xcode toolchain. If the Wuying desktop is Windows or Linux, it can help with general editing and some cross-build tasks, but iOS signing and Packet Tunnel packaging still need a macOS/Xcode environment.