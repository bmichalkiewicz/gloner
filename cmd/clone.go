package cmd

import (
	"context"
	"gloner/git"
	"gloner/repositories"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func Clone() *cli.Command {
	return &cli.Command{
		Name:  "clone",
		Usage: "Clone a repository",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Usage:    "The SSH URL of the repository to clone.",
				Aliases:  []string{"u"},
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			path := cmd.String("destination")
			url := cmd.String("url")

			if err := repositories.Validate(url); err != nil {
				return err
			}

			log.Info().Msgf("Using output path: %s", path)

			alreadyCloned, err := git.Clone(url, path)
			if alreadyCloned {
				log.Warn().Msg("Project is already cloned")
			}
			if err != nil {
				return err
			}

			return nil
		},
	}
}
