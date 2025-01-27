package repositories

import (
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type GitlabManager struct {
	client *gitlab.Client
	opt    *gitlab.ListGroupProjectsOptions

	groups []*gitlab.Group
}

func Init(token, url string) (*GitlabManager, error) {
	g, err := gitlab.NewClient(token, gitlab.WithBaseURL(url+"/api/v4"))
	if err != nil {
		return nil, fmt.Errorf("problem with creating client: %v", err)
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

func (gb *GitlabManager) getNestedGroups(groups []string) error {
	for _, group := range groups {
		topGroups, _, err := gb.client.Groups.ListGroups(&gitlab.ListGroupsOptions{
			Search:       gitlab.Ptr(group),
			AllAvailable: gitlab.Ptr(false),
			TopLevelOnly: gitlab.Ptr(true),
		})
		if err != nil {
			return err
		}

		for _, topGroup := range topGroups {
			stack := []*gitlab.Group{topGroup}

			// looping until stack is 0
			for len(stack) > 0 {
				currentGroup := stack[len(stack)-1]

				// remove current group (last group added)
				stack = stack[:len(stack)-1]

				gb.groups = append(gb.groups, currentGroup)

				subGroups, _, err := gb.client.Groups.ListSubGroups(currentGroup.ID, &gitlab.ListSubGroupsOptions{})
				if err != nil {
					return err
				}

				// add subGroups to stack
				stack = append(stack, subGroups...)
			}
		}
	}

	return nil
}

func (gb *GitlabManager) GetGroupProjects(groups []string) ([]*Group, error) {
	g := []*Group{}

	// Fetch nested groups with IDs
	err := gb.getNestedGroups(groups)
	if err != nil {
		return nil, fmt.Errorf("problem with getting group IDs: %v", err)
	}

	var (
		mu        sync.Mutex // To safely update shared resources
		wg        sync.WaitGroup
		errorChan = make(chan error, 1) // Channel to capture the first error
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

			// Safely append to the configTemplate slice
			mu.Lock()
			g = append(g, &Group{
				Name:     strings.ReplaceAll(group.FullName, " ", ""),
				Projects: projects,
			})
			mu.Unlock()
		}(group)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errorChan)

	// Check for any errors
	if err := <-errorChan; err != nil {
		return nil, fmt.Errorf("problem with processing groups: %v", err)
	}

	return g, err
}

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
			// Check if the error is a 404 Group Not Found
			if _, ok := err.(*gitlab.ErrorResponse); ok && err.(*gitlab.ErrorResponse).Response.StatusCode == 404 {
				log.Warn().Msgf("group %s not found\n", group.Name)
				break // Exit the current group's loop and continue with the next group
			} else {
				// For all other errors, return immediately
				return nil, fmt.Errorf("issue with listing group projects: %w", err)
			}
		}
		for _, project := range projects {
			proj = append(proj, project.SSHURLToRepo)
		}

		if resp.NextPage == 0 {
			break
		}
		groupOptions.Page = resp.NextPage
	}

	return proj, nil
}
