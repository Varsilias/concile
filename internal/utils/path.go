package utils

import "path/filepath"

func ResolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	return filepath.Abs(path)
}
