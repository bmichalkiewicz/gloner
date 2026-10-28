package cmd

import (
	"context"
	"fmt"

	"github.com/bmichalkiewicz/gloner/repositories"
	"github.com/urfave/cli/v3"
)

// Clone returns a CLI command for cloning one or more repositories by URL.
func Clone() *cli.Command {
	return &cli.Command{
		Name:      "clone",
		Usage:     "Clone repositories by URL",
		ArgsUsage: "[URL...]",
		Description: "Accepts SSH (git@host:owner/repo.git, ssh://git@host:port/owner/repo.git)\n" +
			"and HTTPS (https://host/owner/repo.git) URLs, given as arguments or with --url.",
		Flags: append([]cli.Flag{
			&cli.StringSliceFlag{
				Name:    "url",
				Usage:   "The URL of the repository to clone (e.g., git@github.com:user/repo.git)",
				Aliases: []string{"u"},
			},
		}, cloneFlags()...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			urls := append(cmd.StringSlice("url"), cmd.Args().Slice()...)
			if len(urls) == 0 {
				return fmt.Errorf("no repository URL given: pass it as an argument or with --url")
			}

			for _, url := range urls {
				if err := repositories.Validate(url); err != nil {
					return fmt.Errorf("invalid repository URL: %w", err)
				}
			}

			opts, err := gitOptions(cmd)
			if err != nil {
				return err
			}
			base, err := destination(cmd)
			if err != nil {
				return err
			}

			group := &repositories.Group{Name: "clone"}
			for _, url := range urls {
				group.Projects = append(group.Projects, repositories.Project{URL: url})
			}

			return runBatches(ctx, cloneBatches([]*repositories.Group{group}, base, opts), int(cmd.Int("concurrency")), opts.DryRun)
		},
	}
}
