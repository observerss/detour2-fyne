#!/usr/bin/env bash
set -euo pipefail

export PATH="$HOME/bin:/Users/user/sdk/go/bin:/Users/user/go/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

cd "${1:-$HOME/Projects/detour2-fyne}"

echo '--- Xcode ---'
xcode-select -p
xcodebuild -version
xcrun --sdk iphoneos --show-sdk-path

echo '--- Go/Fyne/Gomobile ---'
go version
fyne version || true
gomobile version || true

echo '--- Signing ---'
security find-identity -v -p codesigning
profile_dir="$HOME/Library/MobileDevice/Provisioning Profiles"
if [[ -d "$profile_dir" ]]; then
  find "$profile_dir" -maxdepth 1 -name '*.mobileprovision' -print | sed -n '1,80p'
else
  echo 'No provisioning profile directory.'
fi

echo '--- Devices ---'
xcrun devicectl list devices || true

echo '--- iOS backend compile smoke ---'
SDKROOT="$(xcrun --sdk iphoneos --show-sdk-path)" \
GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
go test ./mobile/iospacket -run '^$' -count=0

echo '--- gomobile framework ---'
mkdir -p ios/Build
GOFLAGS="${GOFLAGS:--mod=mod}" gomobile bind \
  -target=ios \
  -iosversion=13.0 \
  -bundleid=github.com.observerss.detour2fyne.tunnel.backend \
  -o ios/Build/DetourTunnelBackend.xcframework \
  ./mobile/iospacket
find ios/Build/DetourTunnelBackend.xcframework -maxdepth 4 -type f | sort | sed -n '1,80p'

echo '--- overlay staging ---'
IOS_SKIP_FYNE_BUILD=1 IOS_SKIP_GOMOBILE_BIND=1 IOS_PATCH_PROJECT=0 bash scripts/Prepare-iOSPacketTunnel.sh
find ios/Build/Detour2Native -maxdepth 4 -type f | sort | sed -n '1,120p'

echo '--- Fyne iOS package ---'
bash build.sh ios

echo '[ok] iOS build chain completed. If a signed .app was produced, install/run it from Xcode or xcrun devicectl.'
