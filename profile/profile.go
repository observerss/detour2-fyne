package profile

import (
	"encoding/json"
	"os"

	"github.com/observerss/detour2-fyne/utils"
)

var EmptyProfile = map[string]*Profile{}

type runState struct {
	ProfileName    string `json:"profileName,omitempty"`
	Socks5Port     string `json:"socks5Port,omitempty"`
	HTTPPort       string `json:"httpPort,omitempty"`
	LocalPort      string `json:"localPort,omitempty"`
	RunOnStartup   bool   `json:"runOnStartup,omitempty"`
	UseGlobalProxy bool   `json:"useGlobalProxy,omitempty"`
}

func LoadProfiles() (map[string]*Profile, error) {
	path, err := GetProfilePath()
	if err != nil {
		return EmptyProfile, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultProfiles(), nil
		}
		return EmptyProfile, err
	}
	profs := make(map[string]*Profile, 0)
	err = json.Unmarshal(content, &profs)
	if err != nil {
		return EmptyProfile, err
	}
	if len(profs) == 0 {
		return defaultProfiles(), nil
	}
	return NormalizeProfiles(profs), nil
}

func SaveProfiles(profs map[string]*Profile) error {
	profs = NormalizeProfiles(profs)
	tosave := make(map[string]*Profile, 0)
	for _, prof := range profs {
		if prof == nil || prof.Name == "" {
			continue
		}
		var newprof = *prof
		tosave[newprof.Name] = &newprof
	}
	data, err := json.MarshalIndent(tosave, "", "  ")
	if err != nil {
		return err
	}
	path, err := GetProfilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func GetProfilePath() (string, error) {
	return utils.AppConfigFile("profiles.json")
}

func defaultProfiles() map[string]*Profile {
	return DefaultProfiles()
}

func SetCurrentProfileName(name string) error {
	name = NormalizeProfileName(name)
	path, err := utils.AppConfigFile("run.json")
	if err != nil {
		return err
	}
	state := &runState{}
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else if len(content) > 0 {
		if err := json.Unmarshal(content, state); err != nil {
			return err
		}
	}
	state.ProfileName = name
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
