//go:build linux
// +build linux

// 未实现
// 实现思路
// 1) all_proxy环境变量
// 2) gsettings/kwriteconfig: 要分Gnome和KDE来处理, 建议只处理Gnome
// 3) 创建一个TUN, 流量转发, 然后再走代理
package proxy
