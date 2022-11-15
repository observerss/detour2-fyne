package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/observerss/detour2-fyne/utils"
)

var EmptyProfile = map[string]*Profile{}

func LoadProfiles() (map[string]*Profile, error) {
	path, err := GetProfilePath()
	if err != nil {
		return EmptyProfile, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return EmptyProfile, err
	}
	profs := make(map[string]*Profile, 0)
	err = json.Unmarshal(content, &profs)
	if err != nil {
		return EmptyProfile, err
	}
	for _, prof := range profs {
		prof.AccessKeyId = Decrypt(prof.AccessKeyId)
		prof.AccessKeySecret = Decrypt(prof.AccessKeySecret)
		prof.AccountId = Decrypt(prof.AccountId)
	}
	return profs, nil
}

func SaveProfiles(profs map[string]*Profile) error {
	tosave := make(map[string]*Profile, 0)
	for _, prof := range profs {
		var newprof = *prof
		newprof.AccessKeyId = Encrypt(prof.AccessKeyId)
		newprof.AccessKeySecret = Encrypt(prof.AccessKeySecret)
		newprof.AccountId = Encrypt(prof.AccountId)
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
	var home, path string
	var err error
	switch runtime.GOOS {
	case "android":
		// hardcode data dir, should get it from getFilesDir android api
		home = "/storage/emulated/0/Android/data/com.detour2/files/"
		path = filepath.Join(home, "profiles.json")
	case "ios":
		home = "???"
	default:
		home, err = os.UserHomeDir()
		if err != nil {
			return "", nil
		}
		path = filepath.Join(home, ".detour", "profiles.json")
		err = utils.EnsureDir(path)
	}

	return path, err
}
