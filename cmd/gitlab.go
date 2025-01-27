package cmd

import (
	"context"
	"errors"
	"fmt"
	"gloner/config"
	"gloner/git"
	"gloner/repositories"
	"sync"

	"github.com/chelnak/ysmrr"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func Gitlab() *cli.Command {
	return &cli.Command{
		Name:  "gitlab",
		Usage: "Clone gitlab repositories",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "GitLab API token for authentication.",
			},
			&cli.StringSliceFlag{
				Name:     "groups",
				Aliases:  []string{"g"},
				Usage:    "A comma-separated list of GitLab group names to fetch repositories.",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "URL of the GitLab instance.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			groups := cmd.StringSlice("groups")
			token := cmd.String("token")
			path := cmd.String("destination")

			if config.Settings.Gitlab.Token == "" {
				if token == "" {
					return fmt.Errorf("required flag \"token\" not set")
				} else {
					config.Settings.Gitlab.Token = token
				}
			} else {
				log.Info().Msgf("Used token from config")
			}

			log.Info().Msgf("Using output path: %s", path)

			// Initialize GitLab client
			gm, err := repositories.Init(config.Settings.Gitlab.Token, config.Settings.Gitlab.URL)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to initialize GitLab client with URL: %s", config.Settings.Gitlab.URL)
				return err
			}

			log.Info().Msgf("Fetching repositories for groups: %v", groups)

			var wg sync.WaitGroup
			var mu sync.Mutex
			var combinedError []error

			g, err := gm.GetGroupProjects(groups)
			if err != nil {
				return fmt.Errorf("failed to fetch group projects: %w", err)
			}

			// Initialize spinners
			sm := ysmrr.NewSpinnerManager()
			sm.Start()
			defer sm.Stop()

			for _, group := range g {
				wg.Add(1)
				go func(group *repositories.Group) {
					defer wg.Done()

					spinner := sm.AddSpinner(group.Name)
					spinner.UpdateMessagef("[%s] processing...", group.Name)

					if len(group.Projects) == 0 {
						spinner.ErrorWithMessagef("[%s] no projects found", group.Name)
						return
					}

					for _, project := range group.Projects {
						alreadyCloned, err := git.Clone(project.URL, path)

						if alreadyCloned {
							continue
						}
						if err != nil {
							mu.Lock()
							combinedError = append(combinedError, fmt.Errorf("problems with %s: %v", project.URL, err))
							mu.Unlock()
							continue
						}
					}

					spinner.CompleteWithMessagef("[%s] done!", group.Name)
				}(group)
			}

			wg.Wait()

			if len(combinedError) > 0 {
				return errors.Join(combinedError...)
			}

			return nil
		},
	}
}
