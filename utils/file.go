package utils

import (
	"os"
	"path/filepath"

	"github.com/observerss/detour2/logger"
)

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
