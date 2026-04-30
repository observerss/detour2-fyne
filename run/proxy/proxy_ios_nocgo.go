//go:build ios && !cgo
// +build ios,!cgo

package proxy

import "errors"

func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	return errors.New("iOS Packet Tunnel 需要 cgo + NetworkExtension 构建；请使用 macOS/Xcode/gomobile 打包")
}

func Off() error {
	return errors.New("iOS Packet Tunnel 需要 cgo + NetworkExtension 构建；请使用 macOS/Xcode/gomobile 打包")
}

func Status() string {
	return "iOS Packet Tunnel 需要 cgo 构建"
}
