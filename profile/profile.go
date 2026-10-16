package profile

import (
	"encoding/json"
	"os"

	"github.com/observerss/detour2-fyne/utils"
)

// LoadProfiles reads profiles.json. On first launch (no file yet) it returns
// the built-in lines. On error it returns an empty map and the error; callers
// must not save on top of a file they could not read.
func LoadProfiles() (map[string]*Profile, error) {
	path, err := GetProfilePath()
	if err != nil {
		return map[string]*Profile{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultProfiles(), nil
		}
		return map[string]*Profile{}, err
	}
	profs := make(map[string]*Profile, 0)
	if err := json.Unmarshal(content, &profs); err != nil {
		return map[string]*Profile{}, err
	}
	return NormalizeProfiles(profs), nil
}

func SaveProfiles(profs map[string]*Profile) error {
	tosave := make(map[string]*Profile, len(profs))
	for name, prof := range NormalizeProfiles(profs) {
		newprof := *prof
		tosave[name] = &newprof
	}
	data, err := json.MarshalIndent(tosave, "", "  ")
	if err != nil {
		return err
	}
	path, err := GetProfilePath()
	if err != nil {
		return err
	}
	return utils.WriteFileAtomic(path, data, 0o600)
}

func GetProfilePath() (string, error) {
	return utils.AppConfigFile("profiles.json")
}

// SetCurrentProfileName updates profileName in run.json and keeps every other
// field, including ones this package does not know about.
func SetCurrentProfileName(name string) error {
	path, err := utils.AppConfigFile("run.json")
	if err != nil {
		return err
	}
	state := map[string]any{}
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else if len(content) > 0 {
		if err := json.Unmarshal(content, &state); err != nil {
			return err
		}
	}
	if name = NormalizeProfileName(name); name == "" {
		delete(state, "profileName")
	} else {
		state["profileName"] = name
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return utils.WriteFileAtomic(path, data, 0o600)
}
