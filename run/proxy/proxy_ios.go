//go:build ios && cgo
// +build ios,cgo

package proxy

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -fblocks
#cgo LDFLAGS: -framework Foundation -framework NetworkExtension

#import <Foundation/Foundation.h>
#import <NetworkExtension/NetworkExtension.h>
#include <stdlib.h>

static char* detour_copy_string(NSString* value) {
	if (value == nil) {
		return NULL;
	}
	return strdup([value UTF8String]);
}

static void detour_set_error(char** error_out, NSString* message) {
	if (error_out == NULL) {
		return;
	}
	*error_out = detour_copy_string(message);
}

static NETunnelProviderManager* detour_find_manager(NSArray<NETunnelProviderManager*>* managers, NSString* localized_description) {
	for (NETunnelProviderManager* manager in managers) {
		if ([manager.localizedDescription isEqualToString:localized_description]) {
			return manager;
		}
	}
	return [[NETunnelProviderManager alloc] init];
}

static NSDictionary* detour_provider_config(const char* remote_address, const char* password, int mtu, int socks5_port, int pool_size) {
	NSString* remote = [NSString stringWithUTF8String:remote_address == NULL ? "" : remote_address];
	NSString* pass = [NSString stringWithUTF8String:password == NULL ? "" : password];
	return @{
		@"remoteAddress": remote,
		@"password": pass,
		@"dnsServers": @[@"1.1.1.1", @"8.8.8.8"],
		@"mtu": @(mtu),
		@"localSocksPort": @(socks5_port),
		@"poolSize": @(pool_size),
	};
}

static int detour_ios_start_tunnel(const char* provider_bundle_id, const char* remote_address, const char* password, int mtu, int socks5_port, int pool_size, char** error_out) {
	@autoreleasepool {
		NSString* provider = [NSString stringWithUTF8String:provider_bundle_id == NULL ? "" : provider_bundle_id];
		NSString* localized = @"Detour2";
		__block int result = 0;
		__block NSString* failure = nil;
		dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);

		[NETunnelProviderManager loadAllFromPreferencesWithCompletionHandler:^(NSArray<NETunnelProviderManager*>* managers, NSError* error) {
			if (error != nil) {
				result = -1;
				failure = error.localizedDescription;
				dispatch_semaphore_signal(semaphore);
				return;
			}

			NETunnelProviderManager* manager = detour_find_manager(managers, localized);
			NETunnelProviderProtocol* tunnel_protocol = [[NETunnelProviderProtocol alloc] init];
			tunnel_protocol.providerBundleIdentifier = provider;
			tunnel_protocol.serverAddress = localized;
			tunnel_protocol.providerConfiguration = detour_provider_config(remote_address, password, mtu, socks5_port, pool_size);

			manager.localizedDescription = localized;
			manager.protocolConfiguration = tunnel_protocol;
			manager.enabled = YES;

			[manager saveToPreferencesWithCompletionHandler:^(NSError* save_error) {
				if (save_error != nil) {
					result = -2;
					failure = save_error.localizedDescription;
					dispatch_semaphore_signal(semaphore);
					return;
				}

				[manager loadFromPreferencesWithCompletionHandler:^(NSError* load_error) {
					if (load_error != nil) {
						result = -3;
						failure = load_error.localizedDescription;
						dispatch_semaphore_signal(semaphore);
						return;
					}

					NSError* start_error = nil;
					[manager.connection startVPNTunnelAndReturnError:&start_error];
					if (start_error != nil) {
						result = -4;
						failure = start_error.localizedDescription;
					}
					dispatch_semaphore_signal(semaphore);
				}];
			}];
		}];

		dispatch_semaphore_wait(semaphore, DISPATCH_TIME_FOREVER);
		if (result != 0) {
			detour_set_error(error_out, failure == nil ? @"Unknown iOS tunnel start error" : failure);
		}
		return result;
	}
}

static int detour_ios_stop_tunnel(char** error_out) {
	@autoreleasepool {
		NSString* localized = @"Detour2";
		__block int result = 0;
		__block NSString* failure = nil;
		dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);

		[NETunnelProviderManager loadAllFromPreferencesWithCompletionHandler:^(NSArray<NETunnelProviderManager*>* managers, NSError* error) {
			if (error != nil) {
				result = -1;
				failure = error.localizedDescription;
				dispatch_semaphore_signal(semaphore);
				return;
			}

			NETunnelProviderManager* manager = nil;
			for (NETunnelProviderManager* item in managers) {
				if ([item.localizedDescription isEqualToString:localized]) {
					manager = item;
					break;
				}
			}
			if (manager != nil) {
				[manager.connection stopVPNTunnel];
			}
			dispatch_semaphore_signal(semaphore);
		}];

		dispatch_semaphore_wait(semaphore, DISPATCH_TIME_FOREVER);
		if (result != 0) {
			detour_set_error(error_out, failure == nil ? @"Unknown iOS tunnel stop error" : failure);
		}
		return result;
	}
}
*/
import "C"

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

const defaultProviderBundleID = "github.com.observerss.detour2fyne.PacketTunnel"

var (
	iosStatusMu sync.Mutex
	iosStatus   = "iOS Packet Tunnel 未启用"
)

func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	config, err := parseIOSProxyConfig(proxyServer)
	if err != nil {
		setIOSStatus("iOS Packet Tunnel 配置错误: " + err.Error())
		return err
	}

	provider := C.CString(config.ProviderBundleID)
	remote := C.CString(config.RemoteAddress)
	password := C.CString(config.Password)
	defer C.free(unsafe.Pointer(provider))
	defer C.free(unsafe.Pointer(remote))
	defer C.free(unsafe.Pointer(password))

	var errorOut *C.char
	code := int(C.detour_ios_start_tunnel(provider, remote, password, C.int(config.MTU), C.int(config.Socks5Port), C.int(config.PoolSize), &errorOut))
	if errorOut != nil {
		defer C.free(unsafe.Pointer(errorOut))
	}
	if code != 0 {
		message := C.GoString(errorOut)
		if message == "" {
			message = fmt.Sprintf("iOS Packet Tunnel 启动失败(%d)", code)
		}
		setIOSStatus(message)
		return fmt.Errorf("%s", message)
	}

	setIOSStatus("iOS Packet Tunnel 已请求启动")
	return nil
}

func Off() error {
	var errorOut *C.char
	code := int(C.detour_ios_stop_tunnel(&errorOut))
	if errorOut != nil {
		defer C.free(unsafe.Pointer(errorOut))
	}
	if code != 0 {
		message := C.GoString(errorOut)
		if message == "" {
			message = fmt.Sprintf("iOS Packet Tunnel 停止失败(%d)", code)
		}
		setIOSStatus(message)
		return fmt.Errorf("%s", message)
	}
	setIOSStatus("iOS Packet Tunnel 已请求停止")
	return nil
}

func Status() string {
	iosStatusMu.Lock()
	defer iosStatusMu.Unlock()
	return iosStatus
}

func setIOSStatus(value string) {
	iosStatusMu.Lock()
	iosStatus = value
	iosStatusMu.Unlock()
}

type iosProxyConfig struct {
	ProviderBundleID string
	RemoteAddress    string
	Password         string
	MTU              int
	Socks5Port       int
	PoolSize         int
}

func parseIOSProxyConfig(raw string) (*iosProxyConfig, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, err
	}
	config := &iosProxyConfig{
		ProviderBundleID: strings.TrimSpace(values.Get("providerBundleID")),
		RemoteAddress:    strings.TrimSpace(values.Get("remoteAddress")),
		Password:         values.Get("password"),
		MTU:              parsePositiveInt(values.Get("mtu"), 1500),
		Socks5Port:       parsePositiveInt(values.Get("socks5Port"), 7776),
		PoolSize:         parsePositiveInt(values.Get("poolSize"), 64),
	}
	if config.ProviderBundleID == "" {
		config.ProviderBundleID = defaultProviderBundleID
	}
	if config.RemoteAddress == "" {
		return nil, fmt.Errorf("remoteAddress 不能为空")
	}
	if config.Password == "" {
		return nil, fmt.Errorf("password 不能为空")
	}
	return config, nil
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
