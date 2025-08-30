// Package config handles application configuration management.
// It provides functionality for creating, reading, and validating TOML configuration files.
package config

import (
	"errors"
	"fmt"
	"gloner/facts"
	"io"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog/log"
)

// Settings holds the global application configuration
var Settings *Config

// Config represents the application configuration structure
type Config struct {
	Path   string        `toml:"path"`   // Default path for cloning repositories
	Gitlab GitlabConfig `toml:"gitlab"` // GitLab-specific configuration
}

// GitlabConfig holds GitLab-specific settings
type GitlabConfig struct {
	Token string `toml:"token"` // GitLab API token
	URL   string `toml:"url"`   // GitLab instance URL
}

// Init initializes the configuration system by creating the config file if it doesn't exist
// and loading it into the global Settings variable.
func Init() {
	filepath, err := CreateConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create configuration file")
	}

	log.Debug().Msgf("Using configuration file: %s", filepath)

	configFile, err := os.Open(filepath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to open configuration file: %s", filepath)
	}
	defer func() {
		if closeErr := configFile.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Msg("Failed to close configuration file")
		}
	}()

	byteValue, err := io.ReadAll(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read configuration file")
	}

	if err := toml.Unmarshal(byteValue, &Settings); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse TOML configuration file")
	}

	// Validate configuration after loading
	if err := ValidateConfig(); err != nil {
		log.Fatal().Err(err).Msg("Configuration validation failed")
	}

	log.Info().Msg("Configuration loaded successfully")
}

// CreateConfig ensures the configuration file exists in the user's config directory.
// It creates the directory structure and config file with default values if they don't exist.
// Returns the full path to the configuration file.
func CreateConfig() (string, error) {
	homeDir := facts.GetHomeDirectory()
	if homeDir == "" {
		return "", fmt.Errorf("could not determine home directory")
	}

	configDir := filepath.Join(homeDir, ".config", facts.GetApplicationName())
	log.Debug().Msgf("Configuration directory: %s", configDir)

	// Create configuration directory if it doesn't exist
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	configFilePath := filepath.Join(configDir, "config.toml")
	log.Debug().Msgf("Configuration file path: %s", configFilePath)

	// Check if configuration file exists, create it if it doesn't
	if _, err = os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		log.Info().Msg("Configuration file not found, creating with default values")

		file, err := os.Create(configFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to create config file %s: %w", configFilePath, err)
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				log.Warn().Err(closeErr).Msg("Failed to close config file after creation")
			}
		}()

		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(getDefaultConfig()); err != nil {
			return "", fmt.Errorf("failed to encode default configuration: %w", err)
		}

		log.Info().Msgf("Created configuration file: %s", configFilePath)
	} else if err != nil {
		return "", fmt.Errorf("failed to check config file existence: %w", err)
	}

	return configFilePath, nil
}

// ValidateConfig validates the loaded configuration for required fields and valid values
func ValidateConfig() error {
	if Settings == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Validate GitLab URL if provided
	if Settings.Gitlab.URL != "" {
		if !isValidURL(Settings.Gitlab.URL) {
			return fmt.Errorf("invalid GitLab URL: %s", Settings.Gitlab.URL)
		}
	}

	log.Debug().Msg("Configuration validation passed")
	return nil
}

// getDefaultConfig returns the default configuration values
func getDefaultConfig() map[string]any {
	return map[string]any{
		"gitlab": map[string]any{
			"url":   "https://gitlab.com",
			"token": "",
		},
	}
}

// isValidURL performs basic URL validation
func isValidURL(url string) bool {
	// Basic validation - should start with http:// or https://
	return len(url) > 0 && (len(url) > 8 && (url[:7] == "http://" || url[:8] == "https://"))
}
