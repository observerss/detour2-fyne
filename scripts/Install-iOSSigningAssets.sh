#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/Install-iOSSigningAssets.sh --p12 /path/cert.p12 --profile /path/profile.mobileprovision [--p12-password PASSWORD] [--keychain detour2-ios-build.keychain-db] [--keychain-password PASSWORD]

Installs an iOS signing certificate and provisioning profile on the macOS build host.
The provisioning profile should include the app/extension bundle IDs, Network Extension packet tunnel entitlement, and app group.
USAGE
}

p12_path=""
profile_path=""
p12_password=""
keychain_name="detour2-ios-build.keychain-db"
keychain_password="detour2-ios-build"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --p12) p12_path="${2:-}"; shift 2 ;;
    --profile) profile_path="${2:-}"; shift 2 ;;
    --p12-password) p12_password="${2:-}"; shift 2 ;;
    --keychain) keychain_name="${2:-}"; shift 2 ;;
    --keychain-password) keychain_password="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "$p12_path" || -z "$profile_path" ]]; then
  usage >&2
  exit 2
fi
if [[ ! -f "$p12_path" ]]; then
  echo "p12 not found: $p12_path" >&2
  exit 1
fi
if [[ ! -f "$profile_path" ]]; then
  echo "mobileprovision not found: $profile_path" >&2
  exit 1
fi

keychain_path="$HOME/Library/Keychains/$keychain_name"
profile_dir="$HOME/Library/MobileDevice/Provisioning Profiles"
mkdir -p "$profile_dir"

if [[ ! -f "$keychain_path" ]]; then
  security create-keychain -p "$keychain_password" "$keychain_name"
fi
security unlock-keychain -p "$keychain_password" "$keychain_path"
security set-keychain-settings -lut 21600 "$keychain_path"
security list-keychains -d user -s "$keychain_path" $(security list-keychains -d user | sed 's/[" ]//g')
security import "$p12_path" -k "$keychain_path" -P "$p12_password" -T /usr/bin/codesign -T /usr/bin/security -T /usr/bin/xcodebuild
security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "$keychain_password" "$keychain_path" >/dev/null

uuid=$(/usr/bin/security cms -D -i "$profile_path" | /usr/bin/plutil -extract UUID raw -o - -)
name=$(/usr/bin/security cms -D -i "$profile_path" | /usr/bin/plutil -extract Name raw -o - - 2>/dev/null || true)
cp "$profile_path" "$profile_dir/$uuid.mobileprovision"

echo "[ok] Imported certificate into $keychain_path"
echo "[ok] Installed provisioning profile: $name ($uuid)"
echo '--- signing identities ---'
security find-identity -v -p codesigning "$keychain_path"
echo '--- profile entitlements containing Network Extension/App Groups ---'
/usr/bin/security cms -D -i "$profile_dir/$uuid.mobileprovision" | /usr/bin/plutil -p - | grep -E 'application-identifier|networkextension|application-groups|packet-tunnel|com.apple.developer.team-identifier' || true
