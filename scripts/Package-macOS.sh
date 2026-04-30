#!/usr/bin/env bash
set -euo pipefail

app_name="${MACOS_APP_NAME:-Detour2}"
app_id="${MACOS_APP_ID:-github.com.observerss.detour2fyne}"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
dist_dir="${MACOS_DIST_DIR:-${repo_root}/dist/darwin-universal}"
app_path="${MACOS_APP_PATH:-}"
zip_path="${MACOS_ZIP_PATH:-${dist_dir}/${app_name}-macOS.zip}"
sign_identity="${MACOS_SIGN_IDENTITY:-}"
notary_profile="${MACOS_NOTARY_PROFILE:-}"
apple_id="${APPLE_ID:-}"
apple_team_id="${APPLE_TEAM_ID:-}"
apple_password="${APPLE_APP_SPECIFIC_PASSWORD:-}"
skip_build="${MACOS_SKIP_BUILD:-0}"
skip_sign="${MACOS_SKIP_SIGN:-0}"
skip_notarize="${MACOS_SKIP_NOTARIZE:-0}"
skip_staple="${MACOS_SKIP_STAPLE:-0}"

usage() {
    cat >&2 <<EOF
usage: scripts/Package-macOS.sh

Environment:
  MACOS_APP_PATH                 Existing .app path. If empty, script builds or searches for ${app_name}.app.
  MACOS_DIST_DIR                 Output directory. Default: dist/darwin-universal
  MACOS_SIGN_IDENTITY            Developer ID Application identity, for example "Developer ID Application: Name (TEAMID)".
  MACOS_NOTARY_PROFILE           Optional notarytool keychain profile name.
  APPLE_ID                       Apple ID email for notarytool when no profile is used.
  APPLE_TEAM_ID                  Apple Developer Team ID when no profile is used.
  APPLE_APP_SPECIFIC_PASSWORD    App-specific password when no profile is used.
  MACOS_SKIP_BUILD=1             Do not run ./build.sh macos.
  MACOS_SKIP_SIGN=1              Do not run codesign.
  MACOS_SKIP_NOTARIZE=1          Do not submit to notarytool.
  MACOS_SKIP_STAPLE=1            Do not staple notarization ticket.
EOF
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "missing required command: $1" >&2
        exit 1
    fi
}

find_app() {
    if [[ -n "$app_path" ]]; then
        echo "$app_path"
        return 0
    fi

    local candidate
    for candidate in \
        "${repo_root}/${app_name}.app" \
        "${repo_root}/dist/darwin-arm64/${app_name}.app" \
        "${repo_root}/dist/darwin-amd64/${app_name}.app" \
        "${dist_dir}/${app_name}.app"; do
        if [[ -d "$candidate" ]]; then
            echo "$candidate"
            return 0
        fi
    done

    find "${repo_root}/dist" "${repo_root}" -maxdepth 4 -type d -name "${app_name}.app" 2>/dev/null | head -n 1
}

submit_notary() {
    local package_path="$1"
    if [[ -n "$notary_profile" ]]; then
        xcrun notarytool submit "$package_path" --keychain-profile "$notary_profile" --wait
        return 0
    fi
    if [[ -z "$apple_id" || -z "$apple_team_id" || -z "$apple_password" ]]; then
        echo "notarization needs MACOS_NOTARY_PROFILE or APPLE_ID + APPLE_TEAM_ID + APPLE_APP_SPECIFIC_PASSWORD" >&2
        exit 1
    fi
    xcrun notarytool submit "$package_path" \
        --apple-id "$apple_id" \
        --team-id "$apple_team_id" \
        --password "$apple_password" \
        --wait
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "macOS packaging must run on a macOS host with Xcode command line tools." >&2
    exit 1
fi

need_cmd xcrun
need_cmd codesign
need_cmd ditto

if [[ "$skip_build" != "1" ]]; then
    (cd "$repo_root" && bash build.sh macos)
fi

resolved_app="$(find_app)"
if [[ -z "$resolved_app" || ! -d "$resolved_app" ]]; then
    echo "could not find ${app_name}.app; set MACOS_APP_PATH or run ./build.sh macos first" >&2
    exit 1
fi

mkdir -p "$dist_dir"

if [[ "$skip_sign" != "1" ]]; then
    if [[ -z "$sign_identity" ]]; then
        echo "MACOS_SIGN_IDENTITY is required unless MACOS_SKIP_SIGN=1" >&2
        exit 1
    fi
    codesign \
        --force \
        --deep \
        --options runtime \
        --timestamp \
        --identifier "$app_id" \
        --sign "$sign_identity" \
        "$resolved_app"
    codesign --verify --deep --strict --verbose=2 "$resolved_app"
    spctl --assess --type execute --verbose=2 "$resolved_app" || true
fi

rm -f "$zip_path"
ditto -c -k --keepParent "$resolved_app" "$zip_path"
echo "created $zip_path"

if [[ "$skip_notarize" != "1" ]]; then
    submit_notary "$zip_path"
    if [[ "$skip_staple" != "1" ]]; then
        xcrun stapler staple "$resolved_app"
        xcrun stapler validate "$resolved_app"
    fi
fi

echo "macOS package ready: $zip_path"