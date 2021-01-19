package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

// NormalizePath returns an absolute path of given path.
func NormalizePath(path string) (string, error) {
	if len(path) >= 2 && path[:2] == "~/" {
		path = filepath.Join(userHome(), path[2:])
	}
	return filepath.Abs(path)
}

func userHome() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if d, p := os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"); d != "" && p != "" {
		return d + p
	}
	if p := os.Getenv("USERPROFILE"); p != "" {
		return p
	}
	switch runtime.GOOS {
	case "windows":
		return "C:/"
	default:
		return "/"
	}
}
