package profile

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	DefaultProfileName   = "hk"
	DefaultRemoteAddress = DefaultHKRemoteAddress
	DefaultPassword      = "jy230101"

	DefaultJPRemoteAddress  = "tcp://47.116.180.26:7777"
	DefaultSIGRemoteAddress = "tcp://47.116.180.26:17779"
	DefaultHKRemoteAddress  = "tcp://47.116.180.26:17780"
)

var defaultProfileNames = []string{"hk", "jp", "sig"}

func NewDefaultProfile() *Profile {
	return NewDefaultProfileByName(DefaultProfileName)
}

func NewDefaultProfileByName(name string) *Profile {
	if name == "" {
		name = DefaultProfileName
	}
	return &Profile{
		Name:          name,
		RemoteAddress: defaultRemoteAddressForName(name),
		Password:      DefaultPassword,
	}
}

func defaultRemoteAddressForName(name string) string {
	switch name {
	case "jp":
		return DefaultJPRemoteAddress
	case "sig":
		return DefaultSIGRemoteAddress
	case "hk":
		return DefaultHKRemoteAddress
	default:
		return DefaultRemoteAddress
	}
}

func DefaultProfileNames() []string {
	names := make([]string, len(defaultProfileNames))
	copy(names, defaultProfileNames)
	return names
}

func DefaultProfiles() map[string]*Profile {
	profs := make(map[string]*Profile, len(defaultProfileNames))
	for _, name := range DefaultProfileNames() {
		profs[name] = NewDefaultProfileByName(name)
	}
	return profs
}

func NormalizeProfiles(profs map[string]*Profile) map[string]*Profile {
	if len(profs) == 0 {
		return DefaultProfiles()
	}
	if oldDefault, ok := profs["default"]; ok {
		if _, hasHK := profs[DefaultProfileName]; !hasHK && oldDefault != nil {
			migrated := *oldDefault
			migrated.Name = DefaultProfileName
			profs[DefaultProfileName] = &migrated
		}
		delete(profs, "default")
	}
	for name, prof := range profs {
		if prof == nil {
			delete(profs, name)
			continue
		}
		if prof.Name == "" || prof.Name == "default" {
			prof.Name = name
		}
		if prof.Name == "default" {
			prof.Name = DefaultProfileName
		}
		if prof.RemoteAddress == "" {
			prof.RemoteAddress = defaultRemoteAddressForName(prof.Name)
		}
		if prof.Password == "" {
			prof.Password = DefaultPassword
		}
	}
	for _, name := range DefaultProfileNames() {
		if _, ok := profs[name]; !ok {
			profs[name] = NewDefaultProfileByName(name)
		}
	}
	if len(profs) == 0 {
		return DefaultProfiles()
	}
	return profs
}

func NormalizeProfileName(name string) string {
	if name == "" || name == "default" {
		return DefaultProfileName
	}
	return name
}

func NormalizeRemoteAddress(value string) (string, error) {
	items := strings.Split(value, ",")
	remotes := make([]string, 0, len(items))
	for _, item := range items {
		remote, err := normalizeRemoteAddress(strings.TrimSpace(item))
		if err != nil {
			return "", err
		}
		remotes = append(remotes, remote)
	}
	if len(remotes) == 0 {
		return "", fmt.Errorf("上游地址不能为空")
	}
	return strings.Join(remotes, ","), nil
}

func normalizeRemoteAddress(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("上游地址不能为空")
	}
	if !strings.Contains(value, "://") {
		value = "tcp://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("上游地址格式错误: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("上游地址缺少 host: %s", value)
	}

	switch parsed.Scheme {
	case "tcp", "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("不支持的上游地址协议: %s", parsed.Scheme)
	}

	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/ws"
	}
	return parsed.String(), nil
}
