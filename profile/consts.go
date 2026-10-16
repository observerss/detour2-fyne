package profile

import (
	"fmt"
	"net/url"
	"strings"
)

//go:generate go run ../tools/gendefaults -config ../build.env -o defaults_gen.go

// The built-in lines are injected at build time from build.env (see
// build.env.example) through the generated defaults_gen.go. Neither file is
// committed; without them the app ships with no built-in lines.
var (
	DefaultProfileName string    // selected on first launch
	builtinProfiles    []Profile // seeded when profiles.json does not exist yet
)

// NewDefaultProfile is the blank template for "new" and "reset".
func NewDefaultProfile() *Profile {
	return &Profile{}
}

// DefaultProfiles returns fresh copies of the built-in lines.
func DefaultProfiles() map[string]*Profile {
	profs := make(map[string]*Profile, len(builtinProfiles))
	for _, builtin := range builtinProfiles {
		prof := builtin
		profs[prof.Name] = &prof
	}
	return profs
}

// NormalizeProfiles keeps map keys and names in sync and drops unusable
// entries. It never adds profiles: a profile the user deleted stays deleted.
func NormalizeProfiles(profs map[string]*Profile) map[string]*Profile {
	out := make(map[string]*Profile, len(profs))
	for name, prof := range profs {
		if prof == nil {
			continue
		}
		normalizedName := NormalizeProfileName(firstNonEmpty(prof.Name, name))
		if normalizedName == "" {
			continue
		}
		prof.Name = normalizedName
		out[normalizedName] = prof
	}
	return out
}

// NormalizeProfileName trims a profile name. "default" is an ordinary name:
// older versions used it, and mapping it elsewhere dropped those profiles.
func NormalizeProfileName(name string) string {
	return strings.TrimSpace(name)
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
