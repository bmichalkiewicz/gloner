// Package repositories provides GitLab API integration for fetching repository information.
// It handles group traversal, project listing, and concurrent processing.
package repositories

import (
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GitlabManager handles GitLab API operations for fetching groups and projects.
type GitlabManager struct {
	client *gitlab.Client                     // GitLab API client
	opt    *gitlab.ListGroupProjectsOptions  // Default options for listing projects
	groups []*gitlab.Group                   // Cached list of groups and subgroups
}

// Init creates and initializes a new GitlabManager with the provided token and URL.
// It validates the connection and sets up default options for API requests.
func Init(token, url string) (*GitlabManager, error) {
	if token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}
	if url == "" {
		return nil, fmt.Errorf("GitLab URL is required")
	}

	apiURL := strings.TrimSuffix(url, "/") + "/api/v4"
	log.Debug().Msgf("Initializing GitLab client with API URL: %s", apiURL)

	g, err := gitlab.NewClient(token, gitlab.WithBaseURL(apiURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GitlabManager{
		client: g,
		opt: &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: 50,
				Page:    1,
			},
			Archived: gitlab.Ptr(false),
		},
	}, nil
}

// getNestedGroups fetches all groups and their nested subgroups recursively.
// It uses a stack-based approach to traverse the group hierarchy.
func (gb *GitlabManager) getNestedGroups(groups []string) error {
	for _, groupName := range groups {
		log.Debug().Msgf("Searching for top-level groups matching: %s", groupName)
		topGroups, _, err := gb.client.Groups.ListGroups(&gitlab.ListGroupsOptions{
			Search:       gitlab.Ptr(groupName),
			AllAvailable: gitlab.Ptr(false),
			TopLevelOnly: gitlab.Ptr(true),
		})
		if err != nil {
			return fmt.Errorf("failed to search for group '%s': %w", groupName, err)
		}

		if len(topGroups) == 0 {
			log.Warn().Msgf("No top-level groups found matching: %s", groupName)
			continue
		}

		for _, topGroup := range topGroups {
			log.Debug().Msgf("Processing group: %s (ID: %d)", topGroup.FullName, topGroup.ID)
			stack := []*gitlab.Group{topGroup}

			// Process groups using depth-first traversal
			for len(stack) > 0 {
				// Pop current group from stack
				currentGroup := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				// Add current group to our list
				gb.groups = append(gb.groups, currentGroup)
				log.Debug().Msgf("Added group to processing list: %s", currentGroup.FullName)

				// Fetch subgroups and add them to stack
				subGroups, _, err := gb.client.Groups.ListSubGroups(currentGroup.ID, &gitlab.ListSubGroupsOptions{})
				if err != nil {
					log.Warn().Msgf("Failed to fetch subgroups for %s: %v", currentGroup.FullName, err)
					continue // Continue processing other groups
				}

				if len(subGroups) > 0 {
					log.Debug().Msgf("Found %d subgroups in %s", len(subGroups), currentGroup.FullName)
					// Add subgroups to stack for processing
					stack = append(stack, subGroups...)
				}
			}
		}
	}

	return nil
}

// GetGroupProjects fetches all projects from the specified groups and their subgroups.
// It returns a slice of Group structs containing project information.
func (gb *GitlabManager) GetGroupProjects(groups []string) ([]*Group, error) {
	if len(groups) == 0 {
		return nil, fmt.Errorf("no groups specified")
	}

	result := []*Group{}

	// Fetch nested groups with IDs
	log.Info().Msgf("Fetching nested groups for: %v", groups)
	err := gb.getNestedGroups(groups)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nested groups: %w", err)
	}

	if len(gb.groups) == 0 {
		return nil, fmt.Errorf("no groups found matching the specified names: %v", groups)
	}

	log.Info().Msgf("Processing %d groups (including subgroups)", len(gb.groups))

	var (
		mu        sync.Mutex                // Protects shared result slice
		wg        sync.WaitGroup            // Waits for all goroutines
		errorChan = make(chan error, len(gb.groups)) // Buffered channel for errors
	)

	// Goroutine to process each group
	for _, group := range gb.groups {
		wg.Add(1)

		go func(group *gitlab.Group) {
			defer wg.Done()

			p, err := gb.getProjects(group)
			if err != nil {
				// Capture the first error and send it to the error channel
				select {
				case errorChan <- err:
				default:
				}
				return
			}

			var projects []Project
			for _, project := range p {
				projects = append(projects, Project{URL: project})
			}

			if len(projects) > 0 {
				// Only add groups that have projects
				mu.Lock()
				result = append(result, &Group{
					Name:     sanitizeGroupName(group.FullName),
					Projects: projects,
				})
				mu.Unlock()
				log.Debug().Msgf("Added group %s with %d projects", group.FullName, len(projects))
			} else {
				log.Debug().Msgf("Skipping empty group: %s", group.FullName)
			}
		}(group)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errorChan)

	// Collect any errors that occurred
	var errors []error
	for len(errorChan) > 0 {
		errors = append(errors, <-errorChan)
	}

	if len(errors) > 0 {
		log.Warn().Msgf("Encountered %d errors while processing groups", len(errors))
		// Return partial results with first error
		return result, fmt.Errorf("encountered errors during processing: %w", errors[0])
	}

	log.Info().Msgf("Successfully processed %d groups with projects", len(result))
	return result, nil
}

// getProjects fetches all project SSH URLs from a specific GitLab group.
// It handles pagination and filters out archived projects.
func (gb *GitlabManager) getProjects(group *gitlab.Group) ([]string, error) {
	var proj []string

	groupOptions := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    gb.opt.Page,
			PerPage: gb.opt.PerPage,
		},
		Archived: gb.opt.Archived,
	}
	for {
		projects, resp, err := gb.client.Groups.ListGroupProjects(group.ID, groupOptions)
		if err != nil {
			// Handle API errors gracefully
			if apiErr, ok := err.(*gitlab.ErrorResponse); ok {
				if apiErr.Response.StatusCode == 404 {
					log.Warn().Msgf("Group %s (ID: %d) not found or not accessible", group.Name, group.ID)
					break // Continue with next page/group
				}
				if apiErr.Response.StatusCode == 403 {
					log.Warn().Msgf("Access denied to group %s (ID: %d)", group.Name, group.ID)
					break
				}
			}
			return nil, fmt.Errorf("failed to list projects for group %s (ID: %d): %w", group.Name, group.ID, err)
		}
		for _, project := range projects {
			if project.SSHURLToRepo != "" {
				proj = append(proj, project.SSHURLToRepo)
			} else {
				log.Debug().Msgf("Skipping project %s: no SSH URL available", project.Name)
			}
		}
		log.Debug().Msgf("Fetched %d projects from page %d of group %s", len(projects), groupOptions.Page, group.Name)

		if resp.NextPage == 0 {
			break
		}
		groupOptions.Page = resp.NextPage
	}

	log.Debug().Msgf("Total projects found in group %s: %d", group.Name, len(proj))
	return proj, nil
}

// sanitizeGroupName removes spaces and other characters that might cause issues in directory names
func sanitizeGroupName(name string) string {
	// Replace spaces and other problematic characters
	sanitized := strings.ReplaceAll(name, " ", "")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	return sanitized
}
