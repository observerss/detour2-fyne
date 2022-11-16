//go:build !android && linux
// +build !android,linux

// 未实现
package startup

// Enable 开机自启动
func Enable() error {
	return nil
}

func Disable() error {
	return nil
}
