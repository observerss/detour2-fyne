#!/usr/bin/env bash
set -euo pipefail

target="${1:-desktop}"
app_id="github.com.observerss.detour2fyne"
release_targets=(linux windows macos android ios)

export PATH="${HOME}/go/bin:${PATH}"
if [[ -n "${FYNE_BIN:-}" ]]; then
	fyne_bin="$FYNE_BIN"
else
	fyne_bin="$(command -v fyne || true)"
fi
if [[ -z "$fyne_bin" ]]; then
	echo "fyne CLI not found; install it with: go install fyne.io/fyne/v2/cmd/fyne@latest" >&2
	exit 1
fi

go generate ./...

case "$target" in
	all)
		for release_target in "${release_targets[@]}"; do
			bash "$0" "$release_target"
		done
		;;
	desktop)
		"$fyne_bin" package
		;;
	linux)
		"$fyne_bin" package -os linux -appID "$app_id" -icon Icon.png
		;;
	windows)
		"$fyne_bin" package -os windows -appID "$app_id" -icon Icon.png
		;;
	macos|darwin)
		"$fyne_bin" package -os darwin -appID "$app_id" -icon Icon.png
		;;
	android)
		"$fyne_bin" package -os android/arm64 -appID "$app_id" -icon Icon.png
		;;
	ios|iphone)
		"$fyne_bin" package -os ios -appID "$app_id" -icon Icon.png
		;;
	*)
		echo "usage: $0 [desktop|linux|windows|macos|android|ios|all]" >&2
		exit 2
		;;
esac
