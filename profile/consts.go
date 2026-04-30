package profile

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	DefaultProfileName = ""
	DefaultPassword    = ""
)

var defaultProfileNames = []string{}

func NewDefaultProfile() *Profile {
	return NewDefaultProfileByName(DefaultProfileName)
}

func NewDefaultProfileByName(name string) *Profile {
	return &Profile{
		Name:          NormalizeProfileName(name),
		RemoteAddress: "",
		Password:      DefaultPassword,
	}
}

func defaultRemoteAddressForName(name string) string {
	return ""
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
	for name, prof := range profs {
		if prof == nil {
			delete(profs, name)
			continue
		}
		normalizedName := NormalizeProfileName(firstNonEmpty(prof.Name, name))
		if normalizedName == "" {
			delete(profs, name)
			continue
		}
		prof.Name = normalizedName
		if name != normalizedName {
			delete(profs, name)
			profs[normalizedName] = prof
		}
	}
	return profs
}

func NormalizeProfileName(name string) string {
	name = strings.TrimSpace(name)
	if strings.EqualFold(name, "default") {
		return DefaultProfileName
	}
	return name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
