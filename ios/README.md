# iOS Network Extension Overlay

This directory contains the native iOS overlay needed to turn the Fyne iOS shell into a real system-wide VPN client.

## Files

- `Native/DetourTunnelConfig.swift`: shared Codable config passed to the Packet Tunnel extension.
- `Native/DetourTunnelManager.swift`: main-app helper that saves and starts an `NETunnelProviderManager` profile.
- `PacketTunnel/PacketTunnelProvider.swift`: Network Extension entry point.
- `PacketTunnel/PacketFlowAdapter.swift`: blocking adapter over `NEPacketTunnelFlow` for gomobile/tun2socks.
- `PacketTunnel/Tun2SocksRunner.swift`: runner that starts the linked gomobile backend.
- `PacketTunnel/GomobileBackend.swift`: conditional Swift wiring for the generated gomobile framework.
- `../mobile/iospacket`: Go/gomobile backend that runs Detour2 local SOCKS5 and tun2socks in the extension process.
- `../run/proxy/proxy_ios.go`: Fyne/Go bridge that configures and starts `NETunnelProviderManager` on iOS cgo builds.
- `Detour2.entitlements`: containing app entitlement template.
- `PacketTunnel/PacketTunnel.entitlements`: Packet Tunnel extension entitlement template.
- `PacketTunnel/Info.plist`: extension plist template.

## Integration Steps

1. In Apple Developer, enable Network Extensions with Packet Tunnel for both the app ID and extension app ID.
2. Create matching provisioning profiles for `github.com.observerss.detour2fyne` and `github.com.observerss.detour2fyne.PacketTunnel`.
3. Build and stage the native overlay on macOS with `bash scripts/Prepare-iOSPacketTunnel.sh`.
4. If the generated Xcode project is found and Ruby `xcodeproj` is installed, the script creates or updates the `PacketTunnel` target, links `NetworkExtension.framework`, links `DetourTunnelBackend.xcframework`, applies entitlements, and embeds the extension in the containing app.
5. If automatic project patching is skipped, add the staged files from `Detour2Native` to the targets manually using the generated README in that directory.
6. Check `PacketTunnel/GomobileBackend.swift` against the generated header and adjust symbol names if gomobile emits different Swift names.
7. Make sure the containing app target links NetworkExtension. The Go bridge in `run/proxy/proxy_ios.go` creates the system VPN profile and passes the selected Fyne profile into `providerConfiguration`.

## Build the Go Backend

Run this on macOS after installing Go, Xcode, and gomobile:

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
gomobile bind -target=ios -o ios/Build/DetourTunnelBackend.xcframework ./mobile/iospacket
```

Or run the integrated helper, which performs the Fyne iOS build, generates the gomobile framework, stages the Swift overlay, and patches the Xcode project when possible:

```bash
sudo gem install xcodeproj
IOS_APP_BUNDLE_ID=github.com.observerss.detour2fyne \
IOS_PACKET_TUNNEL_BUNDLE_ID=github.com.observerss.detour2fyne.PacketTunnel \
IOS_APP_GROUP=group.github.com.observerss.detour2fyne \
bash scripts/Prepare-iOSPacketTunnel.sh
```

The generated framework exposes `NewBackend`, `Start`, and `Stop` around a PacketFlow interface. The backend starts Detour2 local SOCKS5 inside the Packet Tunnel extension, then attaches tun2socks to the iOS packet flow.

`GomobileBackend.swift` assumes gomobile emits `IospacketNewBackend`, `IospacketBackend`, and `IospacketPacketFlow`. Check `DetourTunnelBackend.xcframework/.../Headers/Iospacket.objc.h` after generation and adjust the file if gomobile emits slightly different Swift names.

`Tun2SocksRunner` uses a missing-backend factory until the generated framework is linked. `PacketTunnelProvider` installs the gomobile backend automatically when `DetourTunnelBackend` can be imported, which prevents shipping an iOS VPN switch that appears enabled while dropping all traffic.