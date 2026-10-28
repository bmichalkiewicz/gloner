// Package config handles application configuration management.
// It provides functionality for creating, reading, and validating TOML configuration files.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bmichalkiewicz/gloner/facts"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog/log"
)

// DefaultGitlabURL is used when no GitLab URL is configured
const DefaultGitlabURL = "https://gitlab.com"

// Settings holds the global application configuration
var Settings = &Config{}

// Config represents the application configuration structure
type Config struct {
	Path   string       `toml:"path" comment:"Base directory for cloned repositories (default: ~/git)"`
	Gitlab GitlabConfig `toml:"gitlab"`
	Github GithubConfig `toml:"github"`
}

// GitlabConfig holds GitLab-specific settings
type GitlabConfig struct {
	Token string `toml:"token" comment:"GitLab API token (can also be set with GITLAB_TOKEN)"`
	URL   string `toml:"url" comment:"GitLab instance URL"`
}

// GithubConfig holds GitHub-specific settings
type GithubConfig struct {
	Token string `toml:"token" comment:"GitHub API token (can also be set with GITHUB_TOKEN or GH_TOKEN)"`
	URL   string `toml:"url" comment:"GitHub Enterprise URL, leave empty for github.com"`
}

// Init initializes the configuration system by creating the config file if it doesn't exist
// and loading it into the global Settings variable.
func Init() error {
	path, err := CreateConfig()
	if err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}

	log.Debug().Msgf("Using configuration file: %s", path)

	if err := ensurePrivate(path); err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %s: %w", path, err)
	}

	settings := &Config{}
	if err := toml.Unmarshal(data, settings); err != nil {
		return fmt.Errorf("failed to parse TOML configuration file %s: %w", path, err)
	}
	Settings = settings

	if err := ValidateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Debug().Msg("Configuration loaded successfully")
	return nil
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
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	configFilePath := filepath.Join(configDir, "config.toml")

	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		data, err := toml.Marshal(getDefaultConfig())
		if err != nil {
			return "", fmt.Errorf("failed to encode default configuration: %w", err)
		}

		// The file holds API tokens, so it must not be readable by other users
		if err := os.WriteFile(configFilePath, data, 0o600); err != nil {
			return "", fmt.Errorf("failed to create config file %s: %w", configFilePath, err)
		}

		log.Info().Msgf("Created configuration file: %s", configFilePath)
	} else if err != nil {
		return "", fmt.Errorf("failed to check config file existence: %w", err)
	}

	return configFilePath, nil
}

// ensurePrivate restricts permissions of the config file, which may contain tokens.
func ensurePrivate(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat configuration file %s: %w", path, err)
	}

	if info.Mode().Perm()&0o077 != 0 {
		log.Warn().Msgf("Configuration file %s is accessible by other users, restricting permissions to 0600", path)
		if err := os.Chmod(path, 0o600); err != nil {
			return fmt.Errorf("failed to change permissions of %s: %w", path, err)
		}
	}
	return nil
}

// ValidateConfig validates the loaded configuration for required fields and valid values
func ValidateConfig() error {
	if Settings == nil {
		return fmt.Errorf("configuration not loaded")
	}

	for name, value := range map[string]string{"gitlab.url": Settings.Gitlab.URL, "github.url": Settings.Github.URL} {
		if value == "" {
			continue
		}
		if _, err := NormalizeURL(value); err != nil {
			return fmt.Errorf("invalid %s: %w", name, err)
		}
	}

	return nil
}

// NormalizeURL validates an instance URL and adds the https:// scheme when it is missing,
// so both "gitlab.example.com" and "https://gitlab.example.com/" are accepted.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid URL %q: scheme must be http or https", raw)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid URL %q: missing host", raw)
	}

	return strings.TrimSuffix(u.String(), "/"), nil
}

// getDefaultConfig returns the default configuration values
func getDefaultConfig() *Config {
	return &Config{
		Gitlab: GitlabConfig{URL: DefaultGitlabURL},
	}
}
