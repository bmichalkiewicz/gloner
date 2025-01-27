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

var Settings *Config

type Config struct {
	Path   string `toml:"path"`
	Gitlab struct {
		Token string `toml:"token"`
		URL   string `toml:"url"`
	} `toml:"gitlab"`
}

// Init creating config if don't exists and read it into Settings
func Init() {
	filepath, err := CreateConfig()
	if err != nil {
		log.Fatal().Msgf("Error creating config: %v", err)
	}

	configFile, err := os.Open(filepath)
	if err != nil {
		log.Fatal().Msgf("Error opening config file: %v", err)
	}
	defer configFile.Close()

	byteValue, _ := io.ReadAll(configFile)
	if err := toml.Unmarshal(byteValue, &Settings); err != nil {
		log.Fatal().Msgf("Error parsing config file: %v", err)
	}
}

// CreateConfig ensures ~/.config/gloner/config.toml is created with the template and returns the filepath
func CreateConfig() (string, error) {
	homeDir := facts.GetHomeDirectory()
	configDir := filepath.Join(homeDir, ".config", facts.GetApplicationName())

	// Create ~/.config/gloner if it doesn't exists
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		return "", err
	}

	configFilePath := filepath.Join(configDir, "config.toml")

	// check if ~/.config/gloner/config.toml exists
	if _, err = os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {

		// if not, create it with the config.toml
		file, err := os.Create(configFilePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		encoder := toml.NewEncoder(file)

		if err := encoder.Encode(g); err != nil {
			return "", fmt.Errorf("while encoding the config.yaml : %w", err)
		}

	}

	return configFilePath, nil
}
