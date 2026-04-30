#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

socks_proxy="${DETOUR2_MAC_SOCKS_PROXY:-172.20.208.30:7778}"
go_version="${DETOUR2_GO_VERSION:-}"
skip_fyne_build="${DETOUR2_SKIP_FYNE_BUILD:-0}"
skip_gomobile_bind="${DETOUR2_SKIP_GOMOBILE_BIND:-0}"
skip_xcodeproj_gem="${DETOUR2_SKIP_XCODEPROJ_GEM:-0}"
xcodeproj_gem_timeout="${DETOUR2_XCODEPROJ_GEM_TIMEOUT:-180}"

export PATH="$HOME/sdk/go/bin:$HOME/go/bin:/usr/local/go/bin:$PATH"

log() {
    printf '\n==> %s\n' "$1"
}

warn() {
    printf '[warn] %s\n' "$1" >&2
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        printf '[missing] %s\n' "$1" >&2
        return 1
    fi
}

run_with_timeout() {
    local seconds="$1"
    shift

    if [[ "$seconds" == "0" ]]; then
        "$@"
        return $?
    fi

    "$@" &
    local pid=$!
    local elapsed=0
    while kill -0 "$pid" >/dev/null 2>&1; do
        if (( elapsed >= seconds )); then
            warn "command timed out after ${seconds}s: $*"
            kill "$pid" >/dev/null 2>&1 || true
            wait "$pid" 2>/dev/null || true
            return 124
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done

    wait "$pid"
}

fetch() {
    local url="$1"
    local output="$2"
    if curl -fL --retry 2 --connect-timeout 20 -o "$output" "$url"; then
        return 0
    fi
    warn "direct download failed, retrying through SOCKS proxy ${socks_proxy}: ${url}"
    curl --socks5 "$socks_proxy" -fL --retry 2 --connect-timeout 20 -o "$output" "$url"
}

fetch_text() {
    local url="$1"
    if curl -fsSL --connect-timeout 20 "$url"; then
        return 0
    fi
    warn "direct request failed, retrying through SOCKS proxy ${socks_proxy}: ${url}"
    curl --socks5 "$socks_proxy" -fsSL --connect-timeout 20 "$url"
}

install_go_if_missing() {
    if command -v go >/dev/null 2>&1; then
        return 0
    fi

    log "Installing Go in user directory"
    mkdir -p "$HOME/sdk"

    if [[ -z "$go_version" ]]; then
        go_version="$(fetch_text 'https://go.dev/VERSION?m=text' | head -n 1)"
    fi
    if [[ -z "$go_version" ]]; then
        echo "could not determine Go version" >&2
        exit 1
    fi

    local arch goarch url archive
    arch="$(uname -m)"
    case "$arch" in
        arm64|aarch64) goarch="arm64" ;;
        x86_64) goarch="amd64" ;;
        *) echo "unsupported macOS arch: $arch" >&2; exit 2 ;;
    esac

    archive="/tmp/detour2-${go_version}.darwin-${goarch}.tar.gz"
    url="https://go.dev/dl/${go_version}.darwin-${goarch}.tar.gz"
    echo "Downloading ${url}"
    fetch "$url" "$archive"
    rm -rf "$HOME/sdk/go"
    tar -C "$HOME/sdk" -xzf "$archive"
}

install_go_tools() {
    log "Configuring Go"
    go version
    go env -w GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn GOTOOLCHAIN=auto
    go env GOPROXY GOSUMDB GOTOOLCHAIN

    log "Installing fyne and gomobile"
    go install fyne.io/fyne/v2/cmd/fyne@latest
    go install golang.org/x/mobile/cmd/gomobile@latest
    gomobile init
}

install_xcodeproj_if_possible() {
    if [[ "$skip_xcodeproj_gem" == "1" ]]; then
        warn "Skipping Ruby xcodeproj installation"
        return 0
    fi
    if ! command -v ruby >/dev/null 2>&1 || ! command -v gem >/dev/null 2>&1; then
        warn "Ruby/gem not found; Xcode project patching may be skipped"
        return 0
    fi
    if ruby -e 'require "xcodeproj"' >/dev/null 2>&1; then
        return 0
    fi

    log "Installing Ruby xcodeproj gem for project patching"
    if ! run_with_timeout "$xcodeproj_gem_timeout" gem install --user-install --no-document xcodeproj; then
        warn "xcodeproj gem install failed; continuing with staged native files only"
    fi
}

run_checks_and_builds() {
    cd "$repo_root"
    log "Apple host check"
    bash scripts/Check-AppleBuildHost.sh

    log "Go tests"
    go test ./...

    log "iOS backend compile check"
    GOOS=ios GOARCH=arm64 go build ./mobile/iospacket

    log "iOS Packet Tunnel preparation"
    IOS_SKIP_FYNE_BUILD="$skip_fyne_build" \
    IOS_SKIP_GOMOBILE_BIND="$skip_gomobile_bind" \
    bash scripts/Prepare-iOSPacketTunnel.sh
}

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "This script must run on macOS." >&2
    exit 1
fi

need_cmd xcodebuild
need_cmd xcrun
need_cmd git
need_cmd curl

install_go_if_missing
install_go_tools
install_xcodeproj_if_possible
run_checks_and_builds

log "Done"