// Package main provides the entry point for the gloner CLI application.
// Gloner is a tool for cloning Git repositories into structured directory paths.
package main

import (
	"context"
	"gloner/cmd"
	"gloner/config"
	"gloner/facts"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// version is set during build time using ldflags
var version string

func main() {
	// Configure structured logging with human-readable output
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	})

	// Initialize configuration system
	config.Init()

	// Set default version if not provided during build
	if len(version) == 0 {
		version = "develop"
	}

	// Create CLI application
	app := &cli.Command{
		Name:    facts.GetApplicationName(),
		Usage:   "Clone repositories into structured paths, similar to Go package management",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "destination",
				Aliases: []string{"d"},
				Usage:   "Base directory for cloning repositories",
				Value:   getDefaultDestination(),
			},
		},
		Commands: []*cli.Command{
			cmd.Gitlab(), // GitLab group cloning command
			cmd.Clone(),  // Single repository cloning command
		},
	}

	// Run the CLI application
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application encountered a fatal error")
	}
}

// getDefaultDestination returns the default directory for cloning repositories
func getDefaultDestination() string {
	homeDir := facts.GetHomeDirectory()
	if homeDir == "" {
		// Fallback to current directory if home directory cannot be determined
		wd, err := os.Getwd()
		if err != nil {
			return "./repositories"
		}
		return filepath.Join(wd, "repositories")
	}
	return filepath.Join(homeDir, "git")
}
