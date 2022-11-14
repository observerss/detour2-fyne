//go:build android
// +build android

// Android系统代理配置
package proxy

// SetGlobalProxy 设置全局代理
func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	return nil
}

// Off 关闭代理
func Off() error {
	return nil
}
