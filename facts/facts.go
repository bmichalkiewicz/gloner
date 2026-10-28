// Package facts provides application constants and utility functions
// for retrieving system information like home directory and application name.
package facts

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Application constants
const (
	applicationName = "gloner"
)

// GetApplicationName returns the name of the application
func GetApplicationName() string {
	return applicationName
}

// GetHomeDirectory returns the user's home directory path.
// Returns empty string if the home directory cannot be determined.
func GetHomeDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Warn().Err(err).Msg("Could not determine home directory")
		return ""
	}

	return homeDir
}

// ExpandHome replaces a leading "~" in path with the user's home directory.
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}

	homeDir := GetHomeDirectory()
	if homeDir == "" {
		return path
	}
	return filepath.Join(homeDir, strings.TrimPrefix(path, "~"))
}
