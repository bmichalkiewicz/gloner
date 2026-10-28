package cmd

import (
	"context"
	"fmt"

	"github.com/bmichalkiewicz/gloner/config"
	"github.com/bmichalkiewicz/gloner/repositories"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// Gitlab returns a CLI command for cloning repositories from GitLab groups,
// including all of their subgroups.
func Gitlab() *cli.Command {
	return &cli.Command{
		Name:  "gitlab",
		Usage: "Clone repositories from GitLab groups",
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "GitLab API token for authentication (overrides the config file)",
				Sources: cli.EnvVars("GITLAB_TOKEN"),
			},
			&cli.StringSliceFlag{
				Name:     "groups",
				Aliases:  []string{"g"},
				Usage:    "Comma-separated list of GitLab groups, by full path (e.g. group/subgroup) or name",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "URL of the GitLab instance (overrides the config file, default: " + config.DefaultGitlabURL + ")",
				Sources: cli.EnvVars("GITLAB_URL"),
			},
			includeArchivedFlag,
		}, cloneFlags()...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			token := firstNonEmpty(cmd.String("token"), config.Settings.Gitlab.Token)
			if token == "" {
				return fmt.Errorf("GitLab token is required: use --token, GITLAB_TOKEN or set it in the config file")
			}

			url, err := config.NormalizeURL(firstNonEmpty(cmd.String("url"), config.Settings.Gitlab.URL, config.DefaultGitlabURL))
			if err != nil {
				return fmt.Errorf("invalid GitLab URL: %w", err)
			}

			opts, err := gitOptions(cmd)
			if err != nil {
				return err
			}
			base, err := destination(cmd)
			if err != nil {
				return err
			}

			gm, err := repositories.NewGitlab(token, url)
			if err != nil {
				return fmt.Errorf("failed to initialize GitLab client with URL %s: %w", url, err)
			}

			groups := cmd.StringSlice("groups")
			log.Info().Msgf("Fetching repositories from %s for groups: %v", url, groups)

			groupProjects, err := gm.GetGroupProjects(ctx, groups, repositories.ListOptions{
				IncludeArchived: cmd.Bool("include-archived"),
			})
			if err != nil {
				return fmt.Errorf("failed to fetch group projects: %w", err)
			}

			log.Info().Msgf("Using output path: %s", base)
			return runBatches(ctx, cloneBatches(groupProjects, base, opts), int(cmd.Int("concurrency")), opts.DryRun)
		},
	}
}

// firstNonEmpty returns the first non-empty value.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
