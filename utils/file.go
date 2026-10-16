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

// WriteFileAtomic writes data to a temp file in the same directory and renames
// it over path, so a crash mid-write never leaves a truncated config behind.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func EnsureDir(fileName string) error {
	dirName := filepath.Dir(fileName)
	if _, serr := os.Stat(dirName); serr != nil {
		merr := os.MkdirAll(dirName, 0o700)
		if merr != nil {
			logger.Error.Println(merr)
			return merr
		}
	}
	return nil
}
