//go:build darwin
// +build darwin

// 未实现
// 实现思路
// 1) all_proxy环境变量
// 2) networksetup: https://gist.github.com/mnewt/3717166
// 3) 创建一个TUN, 流量转发, 然后再走代理
package proxy

import "errors"

// SetGlobalProxy 设置全局代理
func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	return errors.New("darwin无法用本软件设置全局代理")
}

// Off 关闭代理
func Off() error {
	return errors.New("darwin无法用本软件关闭全局代理")
}
