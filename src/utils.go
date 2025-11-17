package main

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandTilde expands ~ to home directory path.
func ExpandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(homeDir, path[1:])
}
