package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/observerss/detour2-fyne/utils"
)

func LoadRun() (*Run, error) {
	path, err := GetRunPath()
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	run := &Run{}
	err = json.Unmarshal(content, &run)
	if err != nil {
		return nil, err
	}
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
	var home, path string
	var err error
	switch runtime.GOOS {
	case "android":
		// hardcode data dir, should get it from getFilesDir android api
		home = "/storage/emulated/0/Android/data/com.detour2/files/"
		path = filepath.Join(home, "run.json")
	case "ios":
		home = "???"
	default:
		home, err = os.UserHomeDir()
		if err != nil {
			return "", nil
		}
		path = filepath.Join(home, ".detour", "run.json")
		utils.EnsureDir(path)
	}
	return path, err
}
