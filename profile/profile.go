package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var EmptyProfile = map[string]*Profile{}

func LoadProfiles() (map[string]*Profile, error) {
	path, err := GetProfilePath()
	if err != nil {
		return EmptyProfile, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return EmptyProfile, nil
	}
	profs := make(map[string]*Profile, 0)
	err = json.Unmarshal(content, &profs)
	if err != nil {
		return EmptyProfile, nil
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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil
	}
	path := filepath.Join(home, ".detour", "profiles.json")
	ensureDir(path)
	return path, err
}

func ensureDir(fileName string) {
	dirName := filepath.Dir(fileName)
	if _, serr := os.Stat(dirName); serr != nil {
		merr := os.MkdirAll(dirName, os.ModePerm)
		if merr != nil {
			panic(merr)
		}
	}
}
