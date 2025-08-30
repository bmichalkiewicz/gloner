package cmd

import (
	"context"
	"fmt"
	"gloner/git"
	"gloner/repositories"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// Clone returns a CLI command for cloning a single repository.
// It validates the SSH URL format and creates the appropriate directory structure.
func Clone() *cli.Command {
	return &cli.Command{
		Name:  "clone",
		Usage: "Clone a repository",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Usage:    "The SSH URL of the repository to clone (e.g., git@github.com:user/repo.git)",
				Aliases:  []string{"u"},
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			path := cmd.String("destination")
			url := cmd.String("url")

			// Validate SSH URL format
			if err := repositories.Validate(url); err != nil {
				return fmt.Errorf("invalid repository URL: %w", err)
			}

			log.Info().Msgf("Cloning repository: %s", url)
			log.Info().Msgf("Destination path: %s", path)

			// Attempt to clone the repository
			alreadyCloned, err := git.Clone(url, path)
			if alreadyCloned {
				log.Info().Msg("Repository already exists at destination, skipping clone")
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to clone repository %s: %w", url, err)
			}

			log.Info().Msg("Repository cloned successfully")
			return nil
		},
	}
}
