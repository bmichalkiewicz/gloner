package cmd

import (
	"context"
	"fmt"
	"gloner/config"
	"gloner/git"
	"gloner/repositories"
	"runtime"
	"sync"

	"github.com/chelnak/ysmrr"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// Gitlab returns a CLI command for cloning repositories from GitLab groups.
// It supports concurrent fetching of groups and projects, with progress tracking.
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

			// Validate and set authentication token
			if err := validateAndSetToken(token); err != nil {
				return err
			}

			log.Info().Msgf("Using output path: %s", path)

			// Initialize GitLab client with validation
			gm, err := repositories.Init(config.Settings.Gitlab.Token, config.Settings.Gitlab.URL)
			if err != nil {
				return fmt.Errorf("failed to initialize GitLab client with URL %s: %w", config.Settings.Gitlab.URL, err)
			}

			log.Info().Msgf("Fetching repositories for groups: %v", groups)

			// Fetch group projects
			groupProjects, err := gm.GetGroupProjects(groups)
			if err != nil {
				return fmt.Errorf("failed to fetch group projects: %w", err)
			}

			// Clone repositories with concurrent processing
			return cloneGroupProjects(groupProjects, path)
		},
	}
}

// validateAndSetToken validates and sets the GitLab authentication token
func validateAndSetToken(token string) error {
	if config.Settings.Gitlab.Token == "" {
		if token == "" {
			return fmt.Errorf("GitLab token is required: use --token flag or set it in config file")
		}
		config.Settings.Gitlab.Token = token
	} else {
		log.Info().Msg("Using token from config file")
	}
	return nil
}

// cloneGroupProjects clones all projects from groups with concurrent processing
func cloneGroupProjects(groups []*repositories.Group, basePath string) error {
	if len(groups) == 0 {
		log.Warn().Msg("No groups found to process")
		return nil
	}

	// Initialize progress tracking
	sm := ysmrr.NewSpinnerManager()
	sm.Start()
	defer sm.Stop()

	var (
		wg         sync.WaitGroup
		errorChan  = make(chan error, len(groups)*10) // Buffer for potential errors
		maxWorkers = runtime.NumCPU() * 2             // Limit concurrent clones
		semaphore  = make(chan struct{}, maxWorkers)
	)

	// Process each group concurrently
	for _, group := range groups {
		wg.Add(1)
		go func(group *repositories.Group) {
			defer wg.Done()

			spinner := sm.AddSpinner(group.Name)
			spinner.UpdateMessagef("[%s] processing %d projects...", group.Name, len(group.Projects))

			if len(group.Projects) == 0 {
				spinner.ErrorWithMessagef("[%s] no projects found", group.Name)
				return
			}

			// Clone projects within group concurrently
			if err := cloneProjectsConcurrently(group.Projects, basePath, semaphore, errorChan); err != nil {
				spinner.ErrorWithMessagef("[%s] failed: %v", group.Name, err)
				return
			}

			spinner.CompleteWithMessagef("[%s] completed %d projects", group.Name, len(group.Projects))
		}(group)
	}

	// Wait for all groups to complete
	wg.Wait()
	close(errorChan)

	// Collect and return any errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		log.Warn().Msgf("Encountered %d errors during cloning", len(errors))
		return fmt.Errorf("cloning completed with %d errors: %w", len(errors), errors[0])
	}

	return nil
}

// cloneProjectsConcurrently clones projects within a group using controlled concurrency
func cloneProjectsConcurrently(projects []repositories.Project, basePath string, semaphore chan struct{}, errorChan chan<- error) error {
	var wg sync.WaitGroup

	for _, project := range projects {
		wg.Add(1)
		go func(project repositories.Project) {
			defer wg.Done()

			// Acquire semaphore to limit concurrent clones
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			alreadyCloned, err := git.Clone(project.URL, basePath)
			if alreadyCloned {
				log.Debug().Msgf("Repository %s already exists, skipping", project.URL)
				return
			}
			if err != nil {
				errorChan <- fmt.Errorf("failed to clone %s: %w", project.URL, err)
			}
		}(project)
	}

	wg.Wait()
	return nil
}
