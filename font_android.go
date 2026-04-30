//go:build android
// +build android

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	files, err := os.ReadDir("/system/fonts")
	if err != nil {
		return
	}
	for _, name := range []string{"MiSansVF.ttf", "DroidSans.ttf", "Roboto-Regular.ttf", "NotoSerif-Regular.ttf"} {
		for _, f := range files {
			if strings.EqualFold(filepath.Ext(f.Name()), ".ttf") && strings.Contains(f.Name(), name) {
				os.Setenv("FYNE_FONT", fmt.Sprintf("/system/fonts/%s", f.Name()))
				return
			}
		}
	}
}
