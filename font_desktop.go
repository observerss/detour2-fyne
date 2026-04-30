//go:build !android && !ios
// +build !android,!ios

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/flopp/go-findfont"
)

func init() {
	fontPaths := findfont.List()
	for _, name := range []string{"msyh.ttc", "simsun.ttf", "simhei.ttf", "simkai.ttf"} {
		for _, path := range fontPaths {
			if strings.Contains(path, name) && strings.ToLower(filepath.Ext(path)) != ".ttc" {
				os.Setenv("FYNE_FONT", path)
				return
			}
		}
	}
}
