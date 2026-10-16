//go:build windows
// +build windows

// windows系统代理配置 (WinINet, 即 "Internet 选项" 里的局域网设置)
// thanks to https://github.com/Trisia/gosysproxy
package proxy

import (
	"errors"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	wininet             = windows.NewLazySystemDLL("wininet.dll")
	internetSetOption   = wininet.NewProc("InternetSetOptionW")
	internetQueryOption = wininet.NewProc("InternetQueryOptionW")
	globalFree          = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalFree")
)

const (
	_INTERNET_OPTION_PER_CONNECTION_OPTION  = 75
	_INTERNET_OPTION_PROXY_SETTINGS_CHANGED = 95
	_INTERNET_OPTION_REFRESH                = 37
)

const (
	_PROXY_TYPE_DIRECT         = 0x00000001 // direct to net
	_PROXY_TYPE_PROXY          = 0x00000002 // via named proxy
	_PROXY_TYPE_AUTO_PROXY_URL = 0x00000004 // autoproxy URL
	_PROXY_TYPE_AUTO_DETECT    = 0x00000008 // use autoproxy detection
)

const (
	_INTERNET_PER_CONN_FLAGS          = 1
	_INTERNET_PER_CONN_PROXY_SERVER   = 2
	_INTERNET_PER_CONN_PROXY_BYPASS   = 3
	_INTERNET_PER_CONN_AUTOCONFIG_URL = 4
	_INTERNET_PER_CONN_FLAGS_UI       = 10 // Windows 7+: flags as shown in the UI, incl. auto-detect
)

type internetPerConnOptionList struct {
	dwSize        uint32
	pszConnection *uint16
	dwOptionCount uint32
	dwOptionError uint32
	pOptions      unsafe.Pointer
}

// internetPerConnOption mirrors INTERNET_PER_CONN_OPTIONW; value is the
// DWORD / LPWSTR / FILETIME union (8 bytes, little endian).
type internetPerConnOption struct {
	dwOption uint32
	value    uint64
}

func dwordOption(option, value uint32) internetPerConnOption {
	return internetPerConnOption{dwOption: option, value: uint64(value)}
}

// stringOption stores str's address in the union. The address is invisible
// to the GC, so the returned pointer must be kept alive until the call is done.
func stringOption(option uint32, str string) (internetPerConnOption, *uint16, error) {
	ptr, err := windows.UTF16PtrFromString(str)
	if err != nil {
		return internetPerConnOption{}, nil, err
	}
	// store the address as an integer: writing it through a **uint16 view of
	// the uint64 field gets dropped by the compiler (observed with Go 1.26)
	o := internetPerConnOption{dwOption: option, value: uint64(uintptr(unsafe.Pointer(ptr)))}
	return o, ptr, nil
}

// takeString reads a WinINet-allocated string out of the union and frees it.
func takeString(o *internetPerConnOption) string {
	ptr := *(**uint16)(unsafe.Pointer(&o.value))
	if ptr == nil {
		return ""
	}
	s := windows.UTF16PtrToString(ptr)
	globalFree.Call(uintptr(unsafe.Pointer(ptr)))
	return s
}

// newList builds the option list for connection conn ("" = LAN settings).
func newList(conn string, options []internetPerConnOption) (internetPerConnOptionList, error) {
	list := internetPerConnOptionList{
		dwOptionCount: uint32(len(options)),
		pOptions:      unsafe.Pointer(&options[0]),
	}
	list.dwSize = uint32(unsafe.Sizeof(list))
	if conn != "" {
		ptr, err := windows.UTF16PtrFromString(conn)
		if err != nil {
			return list, err
		}
		list.pszConnection = ptr
	}
	return list, nil
}

func setOptions(conn string, options []internetPerConnOption, keep ...*uint16) error {
	list, err := newList(conn, options)
	if err != nil {
		return err
	}
	ret, _, callErr := internetSetOption.Call(
		0,
		_INTERNET_OPTION_PER_CONNECTION_OPTION,
		uintptr(unsafe.Pointer(&list)),
		unsafe.Sizeof(list))
	runtime.KeepAlive(keep)
	runtime.KeepAlive(options)
	if ret == 0 {
		return callErr
	}
	return Flush()
}

// snapshot reads the current proxy settings of connection conn.
func snapshot(conn string) (*Settings, error) {
	s, err := queryOptions(conn, _INTERNET_PER_CONN_FLAGS_UI)
	if err != nil {
		// FLAGS_UI is not supported before Windows 7
		s, err = queryOptions(conn, _INTERNET_PER_CONN_FLAGS)
	}
	return s, err
}

func queryOptions(conn string, flagsOption uint32) (*Settings, error) {
	options := []internetPerConnOption{
		{dwOption: flagsOption},
		{dwOption: _INTERNET_PER_CONN_PROXY_SERVER},
		{dwOption: _INTERNET_PER_CONN_PROXY_BYPASS},
		{dwOption: _INTERNET_PER_CONN_AUTOCONFIG_URL},
	}
	list, err := newList(conn, options)
	if err != nil {
		return nil, err
	}
	size := list.dwSize
	ret, _, callErr := internetQueryOption.Call(
		0,
		_INTERNET_OPTION_PER_CONNECTION_OPTION,
		uintptr(unsafe.Pointer(&list)),
		uintptr(unsafe.Pointer(&size)))
	runtime.KeepAlive(options)
	if ret == 0 {
		return nil, callErr
	}
	return &Settings{
		Flags:         uint32(options[0].value),
		ProxyServer:   takeString(&options[1]),
		ProxyBypass:   takeString(&options[2]),
		AutoConfigURL: takeString(&options[3]),
	}, nil
}

// apply writes back settings previously returned by snapshot.
func apply(conn string, s *Settings) error {
	server, serverPtr, err := stringOption(_INTERNET_PER_CONN_PROXY_SERVER, s.ProxyServer)
	if err != nil {
		return err
	}
	bypass, bypassPtr, err := stringOption(_INTERNET_PER_CONN_PROXY_BYPASS, s.ProxyBypass)
	if err != nil {
		return err
	}
	pac, pacPtr, err := stringOption(_INTERNET_PER_CONN_AUTOCONFIG_URL, s.AutoConfigURL)
	if err != nil {
		return err
	}
	options := []internetPerConnOption{
		dwordOption(_INTERNET_PER_CONN_FLAGS, s.Flags),
		server,
		bypass,
		pac,
	}
	return setOptions(conn, options, serverPtr, bypassPtr, pacPtr)
}

// setProxy points connection conn at proxyServer ("host:port").
func setProxy(conn, proxyServer string, bypass string) error {
	if proxyServer == "" {
		return errors.New("代理服务器(proxyServer)配置为空")
	}
	server, serverPtr, err := stringOption(_INTERNET_PER_CONN_PROXY_SERVER, proxyServer)
	if err != nil {
		return err
	}
	bypassOpt, bypassPtr, err := stringOption(_INTERNET_PER_CONN_PROXY_BYPASS, bypass)
	if err != nil {
		return err
	}
	options := []internetPerConnOption{
		dwordOption(_INTERNET_PER_CONN_FLAGS, _PROXY_TYPE_PROXY|_PROXY_TYPE_DIRECT),
		server,
		bypassOpt,
	}
	return setOptions(conn, options, serverPtr, bypassPtr)
}

func pointsAt(s *Settings, proxyServer string) bool {
	return s != nil && s.Flags&_PROXY_TYPE_PROXY != 0 && sameEndpoint(s.ProxyServer, proxyServer)
}

// sameEndpoint treats localhost and 127.0.0.1 alike: older builds set
// "localhost:port", and their leftovers must still be recognised as ours.
func sameEndpoint(a, b string) bool {
	norm := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		return strings.Replace(s, "localhost:", "127.0.0.1:", 1)
	}
	return norm(a) == norm(b)
}

// defaultSettings is what Windows ships with: direct + "automatically detect".
func defaultSettings() *Settings {
	return &Settings{Flags: _PROXY_TYPE_DIRECT | _PROXY_TYPE_AUTO_DETECT}
}

// Flush 更新系统配置使生效
func Flush() error {
	if ret, _, err := internetSetOption.Call(0, _INTERNET_OPTION_PROXY_SETTINGS_CHANGED, 0, 0); ret == 0 {
		return err
	}
	if ret, _, err := internetSetOption.Call(0, _INTERNET_OPTION_REFRESH, 0, 0); ret == 0 {
		return err
	}
	return nil
}
