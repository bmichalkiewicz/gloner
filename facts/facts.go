// Package facts provides application constants and utility functions
// for retrieving system information like home directory and application name.
package facts

import (
	"github.com/mitchellh/go-homedir"
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
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Warn().Err(err).Msg("Could not determine home directory")
		return ""
	}

	log.Debug().Msgf("Home directory: %s", homeDir)
	return homeDir
}
