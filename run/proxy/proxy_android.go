//go:build android
// +build android

// Android系统代理配置
package proxy

import "errors"

// SetGlobalProxy 设置全局代理
func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	return errors.New("android无法用本软件设置全局代理")
}

// Off 关闭代理
func Off() error {
	return errors.New("android无法用本软件关闭全局代理")
}
