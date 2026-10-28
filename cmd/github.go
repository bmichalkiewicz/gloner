package cmd

import (
	"context"
	"fmt"

	"github.com/bmichalkiewicz/gloner/config"
	"github.com/bmichalkiewicz/gloner/repositories"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// Github returns a CLI command for cloning repositories of GitHub organizations or users.
func Github() *cli.Command {
	return &cli.Command{
		Name:  "github",
		Usage: "Clone repositories from GitHub organizations or users",
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "GitHub API token, required for private repositories (overrides the config file)",
				Sources: cli.EnvVars("GITHUB_TOKEN", "GH_TOKEN"),
			},
			&cli.StringSliceFlag{
				Name:     "owners",
				Aliases:  []string{"o"},
				Usage:    "Comma-separated list of GitHub organizations or users",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "URL of a GitHub Enterprise instance (overrides the config file)",
				Sources: cli.EnvVars("GITHUB_URL"),
			},
			includeArchivedFlag,
			&cli.BoolFlag{
				Name:  "include-forks",
				Usage: "Also clone forked repositories",
			},
		}, cloneFlags()...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			token := firstNonEmpty(cmd.String("token"), config.Settings.Github.Token)
			if token == "" {
				log.Warn().Msg("No GitHub token configured, only public repositories will be cloned")
			}

			url := firstNonEmpty(cmd.String("url"), config.Settings.Github.URL)
			if url != "" {
				normalized, err := config.NormalizeURL(url)
				if err != nil {
					return fmt.Errorf("invalid GitHub URL: %w", err)
				}
				url = normalized
			}

			opts, err := gitOptions(cmd)
			if err != nil {
				return err
			}
			base, err := destination(cmd)
			if err != nil {
				return err
			}

			gm, err := repositories.NewGithub(token, url)
			if err != nil {
				return err
			}

			owners := cmd.StringSlice("owners")
			log.Info().Msgf("Fetching repositories for: %v", owners)

			groups, err := gm.GetOwnerRepositories(ctx, owners, repositories.ListOptions{
				IncludeArchived: cmd.Bool("include-archived"),
				IncludeForks:    cmd.Bool("include-forks"),
			})
			if err != nil {
				return fmt.Errorf("failed to fetch repositories: %w", err)
			}

			log.Info().Msgf("Using output path: %s", base)
			return runBatches(ctx, cloneBatches(groups, base, opts), int(cmd.Int("concurrency")), opts.DryRun)
		},
	}
}
