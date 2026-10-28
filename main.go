// Package main provides the entry point for the gloner CLI application.
// Gloner is a tool for cloning Git repositories into structured directory paths.
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bmichalkiewicz/gloner/cmd"
	"github.com/bmichalkiewicz/gloner/config"
	"github.com/bmichalkiewicz/gloner/facts"
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
	log.Logger = log.Logger.Level(zerolog.InfoLevel)

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
				Usage:   "Base directory for cloning repositories (overrides \"path\" in the config file)",
				Value:   getDefaultDestination(),
				Sources: cli.EnvVars("GLONER_DESTINATION"),
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug logging",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if cmd.Bool("debug") {
				log.Logger = log.Logger.Level(zerolog.DebugLevel)
			}
			// Loaded after flag parsing so --debug also covers configuration loading
			return ctx, config.Init()
		},
		Commands: []*cli.Command{
			cmd.Gitlab(), // GitLab group cloning command
			cmd.Github(), // GitHub organization/user cloning command
			cmd.Clone(),  // Repository cloning by URL
			cmd.Pull(),   // Update of already cloned repositories
		},
	}

	// Cancel running git operations and API requests on Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		stop()
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
