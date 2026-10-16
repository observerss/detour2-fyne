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
  DETOUR2_DEFAULT_PROFILES / DETOUR2_DEFAULT_PROFILE / DETOUR2_DEFAULT_PASSWORD
                          Built-in lines baked into the app; override build.env (see build.env.example)
  ANDROID_TOOLS_ROOT      Holds keystores/ for APK signing. Default: /mnt/d/AndroidTools
  DETOUR2_ANDROID_SDK     Linux SDK with build-tools 35 for native APK injection. Default: ~/.cache/detour2-android/sdk
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
            --exclude build.env \
            "$src/" "$dst/"
    else
        (cd "$src" && tar \
            --exclude ./.git \
            --exclude ./dist \
            --exclude ./fyne-cross \
            --exclude ./vendor \
            --exclude ./build.env \
            -cf - .) | (cd "$dst" && tar -xf -)
    fi
}

prepare_workspace() {
    # bake the built-in lines from build.env into profile/defaults_gen.go
    (cd "$repo_root" && go generate ./profile/)
    mkdir -p "${tmp_root}/Personal"
    copy_tree "$repo_root" "$tmp_app"

    (cd "$tmp_app" && go mod edit -go="$go_version")
    (cd "$tmp_app" && go mod vendor)
}

# Linux build-tools >= 35 (zipalign -P 16 for 16 KB pages); the fyne-cross
# image only ships 33. Pinned to Google's published checksum.
android_build_tools_url="https://dl.google.com/android/repository/build-tools_r35_linux.zip"
android_build_tools_sha1="2cfaa0bbb2336e9ec18ed3ecea84fa2e2af607bc"

ensure_android_linux_sdk() {
    local sdk="$1"
    local bt="${sdk}/build-tools/35.0.0"
    if [[ -x "${bt}/zipalign" ]]; then
        return 0
    fi
    local zip="${sdk}/build-tools_r35_linux.zip"
    mkdir -p "${sdk}/build-tools"
    echo "[i] Downloading Android build-tools 35 (Linux)..."
    curl -fL --retry 2 --connect-timeout 20 -o "$zip" "$android_build_tools_url" || return 1
    if [[ "$(sha1sum "$zip" | cut -d' ' -f1)" != "$android_build_tools_sha1" ]]; then
        echo "build-tools checksum mismatch: $zip" >&2
        rm -f "$zip"
        return 1
    fi
    rm -rf "${sdk}/build-tools/android-15" "$bt"
    unzip -q "$zip" -d "${sdk}/build-tools" && mv "${sdk}/build-tools/android-15" "$bt"
    rm -f "$zip"
}

# inject_android_vpn_service_native runs the sh script embedded in
# Inject-AndroidVpnService.ps1 (between "$script = @'" and "'@") in the
# fyne-cross android image, with Linux build-tools from DETOUR2_ANDROID_SDK.
# The signing keystore stays in ANDROID_TOOLS_ROOT/keystores, shared with the
# Windows injectors.
inject_android_vpn_service_native() {
    local inject_script="$1"
    local apk_path="$2"
    local tools_root="${ANDROID_TOOLS_ROOT:-/mnt/d/AndroidTools}"
    local linux_sdk="${DETOUR2_ANDROID_SDK:-${HOME}/.cache/detour2-android/sdk}"
    local image="${DETOUR2_ANDROID_IMAGE:-${registry}/fyneio/fyne-cross-images:android}"
    local build_dir="${repo_root}/.android-vpn-build"

    ensure_android_linux_sdk "$linux_sdk" || return 1
    mkdir -p "$build_dir" "${tools_root}/keystores" "${linux_sdk}/platforms"
    # android.jar is platform independent; reuse the one in ANDROID_TOOLS_ROOT
    local platforms=()
    if [[ -d "${tools_root}/sdk/platforms" ]]; then
        platforms=(-v "${tools_root}/sdk/platforms:/android-tools/sdk/platforms:ro")
    fi
    sed -n "/^\$script = @'\$/,/^'@\$/p" "$inject_script" | sed '1d;$d' | tr -d '\r' \
        | sed "s#__APK_PATH__#/workspace/${apk_path}#" > "${build_dir}/inject.sh"
    if [[ ! -s "${build_dir}/inject.sh" ]]; then
        echo "could not extract injector script from $inject_script" >&2
        return 1
    fi
    docker run --rm \
        -u "$(id -u):$(id -g)" -e HOME=/tmp \
        -v "${repo_root}:/workspace" \
        -v "${linux_sdk}:/android-tools/sdk:ro" \
        "${platforms[@]}" \
        -v "${tools_root}/keystores:/android-tools/keystores" \
        -w /workspace \
        "$image" \
        sh /workspace/.android-vpn-build/inject.sh
    local status=$?
    rm -rf "$build_dir"
    return $status
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
    if [[ "$(uname -s)" == "Linux" ]] && command -v docker >/dev/null 2>&1 && [[ -f "$inject_script" ]]; then
        # Linux/WSL: the PowerShell wrappers cannot handle \\wsl.localhost paths,
        # so run the same embedded sh script directly with the Linux docker CLI
        if inject_android_vpn_service_native "$inject_script" "$apk_path"; then
            return 0
        fi
        echo "[!] Native Android VPN injection failed; trying PowerShell injectors..." >&2
    fi
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

package_windows_readme_and_tools() {
    local target_dir="$1"
    if [[ "$target_dir" != windows-* ]]; then
        return 0
    fi

    local dst_dist="${repo_root}/dist/${target_dir}"
    local exe_path="${dst_dist}/${app_name}.exe"
    local zip_path="${dst_dist}/${app_name}.zip"
    local readme_source="${repo_root}/docs/README-WindowsPackage.md"
    local sign_script="${repo_root}/scripts/Sign-WindowsLocal.ps1"
    local sign_cmd="${repo_root}/scripts/Sign-WindowsLocal.cmd"
    local stage_dir="${tmp_root}/windows-package-${target_dir}"

    if [[ ! -f "$exe_path" ]]; then
        return 0
    fi

    rm -rf "$stage_dir"
    mkdir -p "${stage_dir}/tools"
    cp -f "$exe_path" "${stage_dir}/${app_name}.exe"
    if [[ -f "$readme_source" ]]; then
        cp -f "$readme_source" "${stage_dir}/README-Windows.md"
    fi
    if [[ -f "$sign_script" ]]; then
        cp -f "$sign_script" "${stage_dir}/tools/Sign-WindowsLocal.ps1"
    fi
    if [[ -f "$sign_cmd" ]]; then
        cp -f "$sign_cmd" "${stage_dir}/tools/Sign-WindowsLocal.cmd"
    fi

    rm -f "$zip_path"
    if command -v powershell.exe >/dev/null 2>&1; then
        local stage_win
        local zip_win
        if command -v cygpath >/dev/null 2>&1; then
            stage_win="$(cygpath -w "$stage_dir")"
            zip_win="$(cygpath -w "$zip_path")"
        else
            stage_win="$stage_dir"
            zip_win="$zip_path"
        fi
        powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "Compress-Archive -Path (Join-Path '${stage_win}' '*') -DestinationPath '${zip_win}' -Force; Unblock-File -LiteralPath '${zip_win}' -ErrorAction SilentlyContinue"
    elif command -v zip >/dev/null 2>&1; then
        (cd "$stage_dir" && zip -qr "$zip_path" .)
    else
        echo "zip tool not found; leaving ${app_name}.exe without portable zip" >&2
        return 0
    fi

    echo "[✓] packaged Windows portable zip with README and local test signing tool"
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
    package_windows_readme_and_tools "$target_dir"

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
