package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/observerss/detour2/logger"
)

const AppConfigDirName = "detour2-fyne"

func AppConfigFile(name string) (string, error) {
	dir, err := appConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	return path, EnsureDir(path)
}

func appConfigDir() (string, error) {
	if runtime.GOOS == "android" {
		return filepath.Join(androidAppFilesDir(), AppConfigDirName), nil
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, AppConfigDirName), nil
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".detour"), nil
	}
	return filepath.Join(os.TempDir(), AppConfigDirName), nil
}

func androidAppFilesDir() string {
	packageName := "github.com.observerss.detour2fyne"
	if content, err := os.ReadFile("/proc/self/cmdline"); err == nil {
		candidate := strings.Trim(string(content), "\x00")
		if candidate != "" && !strings.Contains(candidate, "/") {
			packageName = candidate
		}
	}
	return filepath.Join("/data/data", packageName, "files")
}

func EnsureDir(fileName string) error {
	dirName := filepath.Dir(fileName)
	if _, serr := os.Stat(dirName); serr != nil {
		merr := os.MkdirAll(dirName, os.ModePerm)
		if merr != nil {
			logger.Error.Println(merr)
			return merr
		}
	}
	return nil
}
