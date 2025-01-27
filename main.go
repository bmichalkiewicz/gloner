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

var version string

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	config.Init()

	if len(version) == 0 {
		version = "develop"
	}

	app := &cli.Command{
		Name:    facts.GetHomeDirectory(),
		Usage:   "Tool to clone repositories in a standardized way, similar to package managmement in Go.",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "destination",
				Aliases: []string{"d"},
				Usage:   "Repositories path",
				Value:   filepath.Join(facts.GetHomeDirectory(), "git"),
			},
		},
		Commands: []*cli.Command{
			cmd.Gitlab(),
			cmd.Clone(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application encountered an error")
	}
}
