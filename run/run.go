package run

import (
	"encoding/json"
	"os"

	"github.com/observerss/detour2-fyne/profile"
	"github.com/observerss/detour2-fyne/utils"
)

func LoadRun() (*Run, error) {
	path, err := GetRunPath()
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Run{}, nil
		}
		return nil, err
	}
	run := &Run{}
	err = json.Unmarshal(content, &run)
	if err != nil {
		return nil, err
	}
	normalizeRun(run)
	return run, nil
}

func SaveRun(run *Run) error {
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	path, err := GetRunPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func GetRunPath() (string, error) {
	return utils.AppConfigFile("run.json")
}

func normalizeRun(run *Run) {
	if run == nil {
		return
	}
	if run.Socks5Port == "3333" {
		run.Socks5Port = DefaultSocks5Port
	}
	if run.LocalPort == "3333" {
		run.LocalPort = DefaultSocks5Port
	}
	if run.HTTPPort == "33331" {
		run.HTTPPort = DefaultHTTPPort
	}
	if run.ProfileName == "default" || run.ProfileName == "" {
		run.ProfileName = profile.DefaultProfileName
	}
}
