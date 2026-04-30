#!/usr/bin/env bash
set -euo pipefail

target="${1:-all}"
registry="${FYNE_CROSS_REGISTRY:-docker.1ms.run}"
app_id="github.com.observerss.detour2fyne"
app_name="Detour2"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
tmp_root="${DETOUR2_FYNE_BUILD_TMP:-/tmp/detour2_build}"
tmp_app="${tmp_root}/Personal/detour2-fyne"
repo_go_version="$(awk '/^go / {print $2; exit}' "${repo_root}/go.mod")"
go_version="${FYNE_CROSS_GO_VERSION:-${repo_go_version:-1.26}}"

export PATH="${HOME}/go/bin:${PATH}"

if ! command -v go >/dev/null 2>&1; then
    for go_dir in "/c/Program Files/Go/bin" "/c/Go/bin"; do
        if [[ -x "${go_dir}/go.exe" ]]; then
            export PATH="${go_dir}:${PATH}"
            break
        fi
    done
fi

usage() {
    cat >&2 <<EOF
usage: $0 [windows|linux|android|android-arm64|android-amd64|android-emulator|darwin|ios|all]

Environment:
  DETOUR2_FYNE_BUILD_TMP  Temporary build root. Default: /tmp/detour2_build
  FYNE_CROSS_REGISTRY     Docker registry mirror. Default: docker.1ms.run
  FYNE_CROSS_GO_VERSION   Go directive used only in the temp workspace. Default: current repo go.mod version
  MACOSX_SDK_PATH         Required for darwin builds on Linux, for example /path/to/MacOSX.sdk
EOF
}

if ! command -v fyne-cross >/dev/null 2>&1; then
    echo "fyne-cross not found; install it with: go install github.com/fyne-io/fyne-cross@latest" >&2
    exit 1
fi

copy_tree() {
    local src="$1"
    local dst="$2"

    rm -rf "$dst"
    mkdir -p "$dst"
    if command -v rsync >/dev/null 2>&1; then
        rsync -a --delete \
            --exclude .git \
            --exclude dist \
            --exclude fyne-cross \
            --exclude vendor \
            "$src/" "$dst/"
    else
        (cd "$src" && tar \
            --exclude ./.git \
            --exclude ./dist \
            --exclude ./fyne-cross \
            --exclude ./vendor \
            -cf - .) | (cd "$dst" && tar -xf -)
    fi
}

prepare_workspace() {
    mkdir -p "${tmp_root}/Personal"
    copy_tree "$repo_root" "$tmp_app"

    (cd "$tmp_app" && go mod edit -go="$go_version")
    (cd "$tmp_app" && go mod vendor)
}

inject_android_vpn_service() {
    local target_dir="$1"
    if [[ "$target_dir" != android-* ]]; then
        return 0
    fi

    local inject_script="${repo_root}/scripts/Inject-AndroidVpnService.ps1"
    local local_inject_script="${repo_root}/scripts/Inject-AndroidVpnServiceLocal.ps1"
    local apk_path="dist/${target_dir}/Detour2.apk"
    local injected=0

    run_powershell_script() {
        local script_path="$1"
        local runner=""
        local script_arg="$script_path"
        if command -v powershell.exe >/dev/null 2>&1; then
            runner="powershell.exe"
            if command -v cygpath >/dev/null 2>&1; then
                script_arg="$(cygpath -w "$script_path")"
            fi
        elif command -v pwsh >/dev/null 2>&1; then
            runner="pwsh"
        elif command -v powershell >/dev/null 2>&1; then
            runner="powershell"
        else
            return 127
        fi
        "$runner" -ExecutionPolicy Bypass -File "$script_arg" -ApkPath "$apk_path"
    }

    echo "[i] Injecting Android VPN Service DEX..."
    if [[ -f "$inject_script" ]]; then
        if run_powershell_script "$inject_script"; then
            injected=1
        else
            echo "[!] Docker-based Android VPN injection failed; trying local injector fallback..." >&2
        fi
    else
        echo "[!] Docker-based Android VPN injection script not found; trying local injector fallback..." >&2
    fi

    if [[ "$injected" -eq 0 ]]; then
        if [[ -f "$local_inject_script" ]]; then
            if run_powershell_script "$local_inject_script"; then
                injected=1
            else
                echo "[!] Local Android VPN injection fallback failed" >&2
            fi
        else
            echo "[!] Local Android VPN injection script not found" >&2
        fi
    fi

    if [[ "$injected" -ne 1 ]]; then
        echo "Android VPN Service injection failed; APK is incomplete without DetourVpnService/classes2.dex" >&2
        exit 1
    fi
}

copy_dist() {
    local target_dir="$1"
    local src_dist="${tmp_app}/fyne-cross/dist/${target_dir}"
    local dst_dist="${repo_root}/dist/${target_dir}"

    if [[ ! -d "$src_dist" ]]; then
        echo "missing fyne-cross dist directory: $src_dist" >&2
        exit 1
    fi

    mkdir -p "$dst_dist"
    find "$dst_dist" -mindepth 1 -maxdepth 1 -type f -delete
    cp -f "$src_dist"/* "$dst_dist"/

    inject_android_vpn_service "$target_dir"

    echo "[✓] copied ${target_dir} artifacts"
    find "$dst_dist" -maxdepth 1 -type f -printf '%p %s bytes\n' | sort
    (cd "$repo_root" && sha256sum "dist/${target_dir}"/*)
}

run_fyne_cross() {
    local platform="$1"
    local arch="$2"
    local target_dir="${platform}-${arch}"
    local go_proxy="${FYNE_CROSS_GOPROXY:-${GOPROXY:-https://goproxy.cn,direct}}"
    local go_sumdb="${FYNE_CROSS_GOSUMDB:-${GOSUMDB:-sum.golang.google.cn}}"
    local go_toolchain="${FYNE_CROSS_GOTOOLCHAIN:-auto}"
    shift 2

    (cd "$tmp_app" && fyne-cross "$platform" \
        -arch="$arch" \
        -app-id "$app_id" \
        -name "$app_name" \
        -env GOFLAGS=-mod=vendor \
        -env "GOTOOLCHAIN=${go_toolchain}" \
        -env "GOPROXY=${go_proxy}" \
        -env "GOSUMDB=${go_sumdb}" \
        -docker-registry "$registry" \
        "$@")
    copy_dist "$target_dir"
}

build_target() {
    case "$1" in
        windows)
            run_fyne_cross windows amd64
            ;;
        linux)
            run_fyne_cross linux amd64
            ;;
        android|android-arm64)
            run_fyne_cross android arm64
            ;;
        android-amd64|android-x86_64|android-emulator)
            run_fyne_cross android amd64
            ;;
        darwin|macos)
            if [[ -z "${MACOSX_SDK_PATH:-}" ]]; then
                echo "darwin packaging needs MACOSX_SDK_PATH; skipping" >&2
                return 0
            fi
            run_fyne_cross darwin arm64 -macosx-sdk-path "$MACOSX_SDK_PATH"
            ;;
        ios|iphone)
            if [[ "$(uname -s)" != "Darwin" ]]; then
                echo "ios packaging is supported only on darwin hosts; skipping" >&2
                return 0
            fi
            run_fyne_cross ios arm64
            ;;
        *)
            usage
            exit 2
            ;;
    esac
}

prepare_workspace

case "$target" in
    all)
        for release_target in windows linux android darwin ios; do
            build_target "$release_target"
        done
        ;;
    *)
        build_target "$target"
        ;;
esac
