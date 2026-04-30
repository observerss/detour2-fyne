#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

app_bundle_id="${IOS_APP_BUNDLE_ID:-github.com.observerss.detour2fyne}"
packet_bundle_id="${IOS_PACKET_TUNNEL_BUNDLE_ID:-${app_bundle_id}.PacketTunnel}"
app_group="${IOS_APP_GROUP:-group.${app_bundle_id}}"
app_target="${IOS_APP_TARGET:-}"
packet_target="${IOS_PACKET_TARGET:-PacketTunnel}"
deployment_target="${IOS_DEPLOYMENT_TARGET:-15.0}"
build_dir="${IOS_BUILD_DIR:-${repo_root}/ios/Build}"
gomobile_output="${IOS_GOMOBILE_OUTPUT:-${build_dir}/DetourTunnelBackend.xcframework}"
xcodeproj_path="${IOS_XCODEPROJ:-}"
skip_fyne_build="${IOS_SKIP_FYNE_BUILD:-0}"
skip_gomobile_bind="${IOS_SKIP_GOMOBILE_BIND:-0}"
patch_project="${IOS_PATCH_PROJECT:-1}"

usage() {
    cat >&2 <<EOF
usage: scripts/Prepare-iOSPacketTunnel.sh

Run on macOS after Apple signing/provisioning is available.

Environment:
  IOS_XCODEPROJ                  Existing .xcodeproj to patch. If empty, the script searches after ./build.sh ios.
  IOS_APP_TARGET                 Containing app target name. If empty, the first application target is used.
  IOS_PACKET_TARGET              Packet Tunnel target name. Default: PacketTunnel.
  IOS_APP_BUNDLE_ID              Default: ${app_bundle_id}
  IOS_PACKET_TUNNEL_BUNDLE_ID    Default: <IOS_APP_BUNDLE_ID>.PacketTunnel
  IOS_APP_GROUP                  Default: group.<IOS_APP_BUNDLE_ID>
  IOS_DEPLOYMENT_TARGET          Default: 15.0
  IOS_BUILD_DIR                  Default: ios/Build
  IOS_GOMOBILE_OUTPUT            Default: ios/Build/DetourTunnelBackend.xcframework
  IOS_SKIP_FYNE_BUILD=1          Do not run ./build.sh ios.
  IOS_SKIP_GOMOBILE_BIND=1       Do not run gomobile bind.
  IOS_PATCH_PROJECT=0            Stage files only; do not patch the Xcode project.

Optional automatic Xcode project patching needs Ruby and the xcodeproj gem:
  sudo gem install xcodeproj
EOF
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "missing required command: $1" >&2
        exit 1
    fi
}

find_xcodeproj() {
    if [[ -n "$xcodeproj_path" ]]; then
        case "$xcodeproj_path" in
            /*) echo "$xcodeproj_path" ;;
            *) echo "$repo_root/$xcodeproj_path" ;;
        esac
        return 0
    fi

    local found_project
    found_project="$(find "$repo_root" \
        \( -path "${repo_root}/.git" -o -path "${repo_root}/vendor" -o -path "${repo_root}/fyne-cross" \) -prune \
        -o -name "*.xcodeproj" -type d -print 2>/dev/null | head -n 1 || true)"
    echo "$found_project"
}

copy_file() {
    local source_file="$1"
    local target_file="$2"
    mkdir -p "$(dirname "$target_file")"
    cp "$source_file" "$target_file"
}

copy_directory() {
    local source_dir="$1"
    local target_dir="$2"
    rm -rf "$target_dir"
    mkdir -p "$(dirname "$target_dir")"
    if command -v ditto >/dev/null 2>&1; then
        ditto "$source_dir" "$target_dir"
    else
        cp -R "$source_dir" "$target_dir"
    fi
}

escape_sed_replacement() {
    printf '%s' "$1" | sed 's/[&#]/\\&/g'
}

copy_with_app_group() {
    local source_file="$1"
    local target_file="$2"
    local escaped_app_group
    escaped_app_group="$(escape_sed_replacement "$app_group")"
    mkdir -p "$(dirname "$target_file")"
    sed "s#group.github.com.observerss.detour2fyne#${escaped_app_group}#g" "$source_file" > "$target_file"
}

stage_overlay() {
    local project_path="$1"
    local stage_dir
    if [[ -n "$project_path" ]]; then
        stage_dir="${IOS_NATIVE_STAGE:-$(dirname "$project_path")/Detour2Native}"
    else
        stage_dir="${IOS_NATIVE_STAGE:-${build_dir}/Detour2Native}"
    fi

    rm -rf "$stage_dir"
    mkdir -p "$stage_dir/PacketTunnel" "$stage_dir/Native" "$stage_dir/Frameworks"

    copy_file "$repo_root/ios/PacketTunnel/PacketTunnelProvider.swift" "$stage_dir/PacketTunnel/PacketTunnelProvider.swift"
    copy_file "$repo_root/ios/PacketTunnel/Tun2SocksRunner.swift" "$stage_dir/PacketTunnel/Tun2SocksRunner.swift"
    copy_file "$repo_root/ios/PacketTunnel/GomobileBackend.swift" "$stage_dir/PacketTunnel/GomobileBackend.swift"
    copy_file "$repo_root/ios/PacketTunnel/PacketFlowAdapter.swift" "$stage_dir/PacketTunnel/PacketFlowAdapter.swift"
    copy_file "$repo_root/ios/PacketTunnel/Info.plist" "$stage_dir/PacketTunnel/Info.plist"
    copy_with_app_group "$repo_root/ios/PacketTunnel/PacketTunnel.entitlements" "$stage_dir/PacketTunnel/PacketTunnel.entitlements"

    copy_file "$repo_root/ios/Native/DetourTunnelConfig.swift" "$stage_dir/Native/DetourTunnelConfig.swift"
    copy_file "$repo_root/ios/Native/DetourTunnelManager.swift" "$stage_dir/Native/DetourTunnelManager.swift"
    copy_with_app_group "$repo_root/ios/Detour2.entitlements" "$stage_dir/Detour2.entitlements"

    if [[ -d "$gomobile_output" ]]; then
        copy_directory "$gomobile_output" "$stage_dir/Frameworks/DetourTunnelBackend.xcframework"
    else
        echo "[warn] gomobile framework not found: $gomobile_output" >&2
        echo "[warn] The Xcode target can still be patched, but Packet Tunnel linking will fail until the framework exists." >&2
    fi

    cat > "$stage_dir/Integration.env" <<EOF
IOS_APP_BUNDLE_ID=${app_bundle_id}
IOS_PACKET_TUNNEL_BUNDLE_ID=${packet_bundle_id}
IOS_APP_GROUP=${app_group}
IOS_PACKET_TARGET=${packet_target}
IOS_DEPLOYMENT_TARGET=${deployment_target}
EOF

    cat > "$stage_dir/README.generated.md" <<EOF
# Detour2 iOS Native Overlay

This directory was generated by bash scripts/Prepare-iOSPacketTunnel.sh.

- Packet Tunnel bundle id: ${packet_bundle_id}
- App Group: ${app_group}
- Gomobile framework: Frameworks/DetourTunnelBackend.xcframework

If the Xcode project was not patched automatically, add the Swift files under PacketTunnel to a Packet Tunnel Provider target, add the Native config files where needed, link NetworkExtension.framework and DetourTunnelBackend.xcframework, set the entitlement files, and embed the PacketTunnel appex in the containing app.
EOF

    echo "$stage_dir"
}

check_gomobile_symbols() {
    if [[ ! -d "$gomobile_output" ]]; then
        return 0
    fi
    local header_file
    header_file="$(find "$gomobile_output" -name 'Iospacket.objc.h' -print 2>/dev/null | head -n 1)"
    if [[ -z "$header_file" ]]; then
        echo "[warn] Could not find Iospacket.objc.h under $gomobile_output; check generated gomobile names in Xcode." >&2
        return 0
    fi
    for symbol in IospacketNewBackend IospacketBackend IospacketPacketFlow; do
        if ! grep -q "$symbol" "$header_file"; then
            echo "[warn] $symbol was not found in $header_file; adjust Detour2Native/PacketTunnel/GomobileBackend.swift after generation." >&2
        fi
    done
}

patch_xcode_project() {
    local project_path="$1"
    local stage_dir="$2"

    if [[ "$patch_project" != "1" ]]; then
        echo "[i] IOS_PATCH_PROJECT=0, staged files only: $stage_dir"
        return 0
    fi
    if [[ -z "$project_path" || ! -d "$project_path" ]]; then
        echo "[warn] No .xcodeproj found. Set IOS_XCODEPROJ=/path/to/App.xcodeproj and rerun to patch automatically." >&2
        echo "[i] Staged files: $stage_dir"
        return 0
    fi
    if ! command -v ruby >/dev/null 2>&1; then
        echo "[warn] ruby not found; staged files only: $stage_dir" >&2
        return 0
    fi
    if ! ruby -e 'begin; require "xcodeproj"; rescue LoadError; exit 42; end' >/dev/null 2>&1; then
        echo "[warn] Ruby xcodeproj gem is not installed." >&2
        echo "[i] Install it on macOS with: sudo gem install xcodeproj" >&2
        echo "[i] Then run:" >&2
        echo "ruby scripts/Patch-iOSPacketTunnelProject.rb --project \"$project_path\" --stage-dir \"$stage_dir\" --app-bundle-id \"$app_bundle_id\" --packet-bundle-id \"$packet_bundle_id\" --app-group \"$app_group\"" >&2
        return 0
    fi

    ruby "$script_dir/Patch-iOSPacketTunnelProject.rb" \
        --project "$project_path" \
        --stage-dir "$stage_dir" \
        --app-target "$app_target" \
        --packet-target "$packet_target" \
        --app-bundle-id "$app_bundle_id" \
        --packet-bundle-id "$packet_bundle_id" \
        --app-group "$app_group" \
        --deployment-target "$deployment_target"
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "iOS Packet Tunnel integration must run on macOS with Xcode." >&2
    exit 1
fi

need_cmd go
need_cmd xcodebuild
need_cmd xcrun

if [[ "$skip_fyne_build" != "1" ]]; then
    (cd "$repo_root" && bash build.sh ios)
fi

if [[ "$skip_gomobile_bind" != "1" ]]; then
    need_cmd gomobile
    mkdir -p "$build_dir"
    (cd "$repo_root" && gomobile bind -target=ios -o "$gomobile_output" ./mobile/iospacket)
fi

check_gomobile_symbols

resolved_project="$(find_xcodeproj)"
stage_dir="$(stage_overlay "$resolved_project")"
patch_xcode_project "$resolved_project" "$stage_dir"

echo "[ok] iOS Packet Tunnel overlay is ready."
if [[ -n "$resolved_project" ]]; then
    echo "[ok] Xcode project: $resolved_project"
    echo "[ok] Native overlay: $stage_dir"
    echo "Open with: open \"$resolved_project\""
else
    echo "[ok] Native overlay: $stage_dir"
fi