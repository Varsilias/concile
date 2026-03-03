package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func ResolvePath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path, err
		}
		return filepath.Join(home, path[1:]), nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}

	return filepath.Abs(path)
}
