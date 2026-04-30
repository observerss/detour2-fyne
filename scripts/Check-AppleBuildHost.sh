#!/usr/bin/env bash
set -euo pipefail

check_cmd() {
    local name="$1"
    if command -v "$name" >/dev/null 2>&1; then
        printf '[ok] %s: %s\n' "$name" "$(command -v "$name")"
    else
        printf '[missing] %s\n' "$name"
    fi
}

echo "Host: $(uname -a)"
if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "[warn] This is not macOS. iOS signing, NetworkExtension packaging, notarytool, and stapler require macOS with Xcode."
fi

check_cmd xcodebuild
check_cmd xcrun
check_cmd codesign
check_cmd security
check_cmd go
check_cmd fyne
check_cmd gomobile
check_cmd git

if command -v xcodebuild >/dev/null 2>&1; then
    xcodebuild -version || true
fi

if command -v xcrun >/dev/null 2>&1; then
    xcrun --find notarytool || true
    xcrun --find stapler || true
fi

if command -v go >/dev/null 2>&1; then
    go version || true
fi

if command -v security >/dev/null 2>&1; then
    echo "Code signing identities:"
    security find-identity -v -p codesigning || true
fi

cat <<'EOF'

Expected for release builds:
  - Xcode is installed and selected with xcode-select.
  - Go, fyne, and gomobile are installed.
  - Apple Developer ID Application certificate is available for macOS signing.
  - iOS Development/Distribution certificates and provisioning profiles include Packet Tunnel capability.
  - notarytool credentials are stored or APPLE_ID/APPLE_TEAM_ID/APPLE_APP_SPECIFIC_PASSWORD are set.
EOF