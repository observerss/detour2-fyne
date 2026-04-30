//go:build darwin && !ios
// +build darwin,!ios

package proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const networksetupPath = "/usr/sbin/networksetup"

var defaultBypassDomains = []string{"localhost", "127.0.0.1", "::1"}

const darwinProxySnapshotFile = "macos-proxy-snapshot.json"

type proxyEndpoint struct {
	Host string
	Port string
}

type darwinProxyConfig struct {
	HTTP  proxyEndpoint
	SOCKS proxyEndpoint
}

type darwinProxySnapshot struct {
	Version  int                                   `json:"version"`
	Services map[string]darwinServiceProxySnapshot `json:"services"`
}

type darwinServiceProxySnapshot struct {
	Web           darwinProtocolProxySnapshot `json:"web"`
	SecureWeb     darwinProtocolProxySnapshot `json:"secureWeb"`
	SOCKS         darwinProtocolProxySnapshot `json:"socks"`
	BypassDomains []string                    `json:"bypassDomains,omitempty"`
}

type darwinProtocolProxySnapshot struct {
	Enabled bool   `json:"enabled"`
	Server  string `json:"server,omitempty"`
	Port    string `json:"port,omitempty"`
}

// SetGlobalProxy 设置全局代理
func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	config, err := parseDarwinProxyConfig(proxyServer)
	if err != nil {
		return err
	}
	services, err := networkServices()
	if err != nil {
		return err
	}
	_ = saveDarwinProxySnapshot(services)
	bypassDomains := normalizeBypassDomains(bypasses)

	commands := make([][]string, 0, len(services)*7)
	for _, service := range services {
		commands = append(commands,
			[]string{"-setwebproxy", service, config.HTTP.Host, config.HTTP.Port, "off"},
			[]string{"-setsecurewebproxy", service, config.HTTP.Host, config.HTTP.Port, "off"},
			[]string{"-setsocksfirewallproxy", service, config.SOCKS.Host, config.SOCKS.Port, "off"},
			[]string{"-setwebproxystate", service, "on"},
			[]string{"-setsecurewebproxystate", service, "on"},
			[]string{"-setsocksfirewallproxystate", service, "on"},
		)
		if len(bypassDomains) > 0 {
			commands = append(commands, append([]string{"-setproxybypassdomains", service}, bypassDomains...))
		}
	}

	return runNetworkSetupCommands(commands)
}

// Off 关闭代理
func Off() error {
	if restored, err := restoreDarwinProxySnapshot(); restored || err != nil {
		return err
	}

	services, err := networkServices()
	if err != nil {
		return err
	}
	commands := make([][]string, 0, len(services)*3)
	for _, service := range services {
		commands = append(commands,
			[]string{"-setwebproxystate", service, "off"},
			[]string{"-setsecurewebproxystate", service, "off"},
			[]string{"-setsocksfirewallproxystate", service, "off"},
		)
	}
	return runNetworkSetupCommands(commands)
}

func Status() string {
	services, err := networkServices()
	if err != nil {
		return "macOS 系统代理状态读取失败: " + err.Error()
	}
	enabled := make([]string, 0, len(services))
	for _, service := range services {
		web := proxyEnabled("-getwebproxy", service)
		secureWeb := proxyEnabled("-getsecurewebproxy", service)
		socks := proxyEnabled("-getsocksfirewallproxy", service)
		if web || secureWeb || socks {
			enabled = append(enabled, service)
		}
	}
	if len(enabled) == 0 {
		return "macOS 系统代理未启用"
	}
	return "macOS 系统代理已启用: " + strings.Join(enabled, ", ")
}

func parseDarwinProxyConfig(value string) (darwinProxyConfig, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return darwinProxyConfig{}, fmt.Errorf("macOS 代理地址不能为空")
	}

	config := darwinProxyConfig{}
	foundNamedEndpoint := false
	for _, item := range strings.Split(value, ";") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		foundNamedEndpoint = true
		endpoint, err := parseEndpoint(parts[1])
		if err != nil {
			return darwinProxyConfig{}, err
		}
		switch strings.ToLower(strings.TrimSpace(parts[0])) {
		case "http", "https", "web":
			config.HTTP = endpoint
		case "socks", "socks5":
			config.SOCKS = endpoint
		default:
			return darwinProxyConfig{}, fmt.Errorf("macOS 代理类型不支持: %s", parts[0])
		}
	}

	if !foundNamedEndpoint {
		endpoint, err := parseEndpoint(value)
		if err != nil {
			return darwinProxyConfig{}, err
		}
		config.HTTP = endpoint
		config.SOCKS = endpoint
	}
	if config.HTTP.Host == "" {
		config.HTTP = config.SOCKS
	}
	if config.SOCKS.Host == "" {
		config.SOCKS = config.HTTP
	}
	if config.HTTP.Host == "" || config.SOCKS.Host == "" {
		return darwinProxyConfig{}, fmt.Errorf("macOS 代理地址缺少 HTTP 或 SOCKS 端点")
	}
	return config, nil
}

func parseEndpoint(value string) (proxyEndpoint, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return proxyEndpoint{}, fmt.Errorf("代理端点不能为空")
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return proxyEndpoint{}, fmt.Errorf("代理端点格式错误: %w", err)
		}
		value = parsed.Host
	}

	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return proxyEndpoint{}, fmt.Errorf("代理端点必须是 host:port: %s", value)
	}
	if strings.TrimSpace(host) == "" {
		host = "localhost"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return proxyEndpoint{}, fmt.Errorf("代理端口必须是 1-65535 之间的数字: %s", port)
	}
	return proxyEndpoint{Host: host, Port: port}, nil
}

func networkServices() ([]string, error) {
	output, err := exec.Command(networksetupPath, "-listallnetworkservices").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("读取 macOS 网络服务失败: %w: %s", err, strings.TrimSpace(string(output)))
	}
	services := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		service := strings.TrimSpace(line)
		if service == "" || strings.HasPrefix(service, "An asterisk") || strings.HasPrefix(service, "*") {
			continue
		}
		services = append(services, service)
	}
	if len(services) == 0 {
		return nil, fmt.Errorf("没有找到启用的 macOS 网络服务")
	}
	return services, nil
}

func normalizeBypassDomains(values []string) []string {
	seen := make(map[string]bool)
	domains := make([]string, 0, len(defaultBypassDomains)+len(values))
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		domains = append(domains, value)
	}
	for _, value := range defaultBypassDomains {
		add(value)
	}
	for _, value := range values {
		for _, item := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ',' || r == ';' || r == ' '
		}) {
			add(item)
		}
	}
	return domains
}

func proxyEnabled(flag string, service string) bool {
	output, err := exec.Command(networksetupPath, flag, service).CombinedOutput()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Enabled:") {
			return strings.Contains(strings.ToLower(line), "yes")
		}
	}
	return false
}

func saveDarwinProxySnapshot(services []string) error {
	path, err := darwinProxySnapshotPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	snapshot := darwinProxySnapshot{
		Version:  1,
		Services: make(map[string]darwinServiceProxySnapshot, len(services)),
	}
	for _, service := range services {
		web, err := readProxySnapshot("-getwebproxy", service)
		if err != nil {
			return err
		}
		secureWeb, err := readProxySnapshot("-getsecurewebproxy", service)
		if err != nil {
			return err
		}
		socks, err := readProxySnapshot("-getsocksfirewallproxy", service)
		if err != nil {
			return err
		}
		bypasses, err := readProxyBypassDomains(service)
		if err != nil {
			return err
		}
		snapshot.Services[service] = darwinServiceProxySnapshot{
			Web:           web,
			SecureWeb:     secureWeb,
			SOCKS:         socks,
			BypassDomains: bypasses,
		}
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func restoreDarwinProxySnapshot() (bool, error) {
	path, err := darwinProxySnapshotPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, err
	}

	snapshot := darwinProxySnapshot{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return true, err
	}
	commands := make([][]string, 0, len(snapshot.Services)*7)
	for service, serviceSnapshot := range snapshot.Services {
		commands = append(commands, restoreProxyCommands(service, "-setwebproxy", "-setwebproxystate", serviceSnapshot.Web)...)
		commands = append(commands, restoreProxyCommands(service, "-setsecurewebproxy", "-setsecurewebproxystate", serviceSnapshot.SecureWeb)...)
		commands = append(commands, restoreProxyCommands(service, "-setsocksfirewallproxy", "-setsocksfirewallproxystate", serviceSnapshot.SOCKS)...)
		commands = append(commands, restoreBypassCommands(service, serviceSnapshot.BypassDomains)...)
	}
	if err := runNetworkSetupCommands(commands); err != nil {
		return true, err
	}
	_ = os.Remove(path)
	return true, nil
}

func restoreProxyCommands(service string, setFlag string, stateFlag string, snapshot darwinProtocolProxySnapshot) [][]string {
	commands := make([][]string, 0, 2)
	if snapshot.Server != "" && snapshot.Port != "" {
		commands = append(commands, []string{setFlag, service, snapshot.Server, snapshot.Port, "off"})
	}
	state := "off"
	if snapshot.Enabled && snapshot.Server != "" && snapshot.Port != "" {
		state = "on"
	}
	commands = append(commands, []string{stateFlag, service, state})
	return commands
}

func restoreBypassCommands(service string, bypasses []string) [][]string {
	if len(bypasses) == 0 {
		return [][]string{{"-setproxybypassdomains", service, "Empty"}}
	}
	return [][]string{append([]string{"-setproxybypassdomains", service}, bypasses...)}
}

func readProxySnapshot(flag string, service string) (darwinProtocolProxySnapshot, error) {
	output, err := exec.Command(networksetupPath, flag, service).CombinedOutput()
	if err != nil {
		return darwinProtocolProxySnapshot{}, fmt.Errorf("networksetup %s %s: %w: %s", flag, service, err, strings.TrimSpace(string(output)))
	}
	snapshot := darwinProtocolProxySnapshot{}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "Enabled":
			snapshot.Enabled = strings.EqualFold(value, "yes")
		case "Server":
			snapshot.Server = value
		case "Port":
			snapshot.Port = value
		}
	}
	return snapshot, nil
}

func readProxyBypassDomains(service string) ([]string, error) {
	output, err := exec.Command(networksetupPath, "-getproxybypassdomains", service).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("networksetup -getproxybypassdomains %s: %w: %s", service, err, strings.TrimSpace(string(output)))
	}
	domains := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "There aren't") {
			continue
		}
		domains = append(domains, line)
	}
	return domains, nil
}

func darwinProxySnapshotPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			if err != nil {
				return "", err
			}
			return "", homeErr
		}
		dir = filepath.Join(home, ".config")
	}
	dir = filepath.Join(dir, "detour2-fyne")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, darwinProxySnapshotFile), nil
}

func runNetworkSetupCommands(commands [][]string) error {
	if len(commands) == 0 {
		return nil
	}
	if err := runNetworkSetupDirect(commands); err == nil {
		return nil
	} else if adminErr := runNetworkSetupWithAdmin(commands); adminErr != nil {
		return fmt.Errorf("macOS 系统代理设置失败: %v; 管理员授权执行也失败: %w", err, adminErr)
	}
	return nil
}

func runNetworkSetupDirect(commands [][]string) error {
	for _, args := range commands {
		output, err := exec.Command(networksetupPath, args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("networksetup %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func runNetworkSetupWithAdmin(commands [][]string) error {
	script := buildNetworkSetupShellScript(commands)
	appleScript := fmt.Sprintf("do shell script %s with administrator privileges", strconv.Quote(script))
	output, err := exec.Command("/usr/bin/osascript", "-e", appleScript).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func buildNetworkSetupShellScript(commands [][]string) string {
	lines := []string{"set -e"}
	for _, args := range commands {
		quoted := make([]string, 0, len(args)+1)
		quoted = append(quoted, shellQuote(networksetupPath))
		for _, arg := range args {
			quoted = append(quoted, shellQuote(arg))
		}
		lines = append(lines, strings.Join(quoted, " "))
	}
	return strings.Join(lines, "\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
