package repositories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// perPage is the page size used for paginated API requests (maximum allowed by GitLab and GitHub)
const perPage = 100

// ListOptions controls which repositories are returned by the providers.
type ListOptions struct {
	IncludeArchived bool // Include archived repositories
	IncludeForks    bool // Include forks (GitHub only)
}

// GitlabManager handles GitLab API operations for fetching groups and projects.
type GitlabManager struct {
	client *gitlab.Client
}

// NewGitlab creates a new GitlabManager with the provided token and instance URL.
func NewGitlab(token, url string) (*GitlabManager, error) {
	if token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}
	if url == "" {
		return nil, fmt.Errorf("GitLab URL is required")
	}

	apiURL := strings.TrimSuffix(url, "/") + "/api/v4"
	log.Debug().Msgf("Initializing GitLab client with API URL: %s", apiURL)

	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(apiURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GitlabManager{client: client}, nil
}

// GetGroupProjects fetches all projects from the specified groups and their subgroups.
// Projects are returned grouped by their namespace (e.g. "group/subgroup").
func (gm *GitlabManager) GetGroupProjects(ctx context.Context, groups []string, opts ListOptions) ([]*Group, error) {
	if len(groups) == 0 {
		return nil, fmt.Errorf("no groups specified")
	}

	byNamespace := map[string]*Group{}
	seen := map[string]bool{}

	for _, name := range groups {
		group, err := gm.resolveGroup(ctx, name)
		if err != nil {
			return nil, err
		}

		log.Debug().Msgf("Fetching projects for group %s (ID: %d)", group.FullPath, group.ID)
		projects, err := gm.listProjects(ctx, group, opts)
		if err != nil {
			return nil, err
		}

		for _, project := range projects {
			if project.SSHURLToRepo == "" {
				log.Debug().Msgf("Skipping project %s: no SSH URL available", project.PathWithNamespace)
				continue
			}
			// The same project is returned twice when both a group and its subgroup are requested
			if seen[project.SSHURLToRepo] {
				continue
			}
			seen[project.SSHURLToRepo] = true

			namespace := path.Dir(project.PathWithNamespace)
			if project.Namespace != nil && project.Namespace.FullPath != "" {
				namespace = project.Namespace.FullPath
			}
			if byNamespace[namespace] == nil {
				byNamespace[namespace] = &Group{Name: namespace}
			}
			byNamespace[namespace].Projects = append(byNamespace[namespace].Projects, Project{URL: project.SSHURLToRepo})
		}
	}

	result := sortGroups(byNamespace)
	log.Debug().Msgf("Found %d projects in %d groups", len(seen), len(result))
	return result, nil
}

// resolveGroup finds a group by its full path (e.g. "group/subgroup").
// When no group has that path, it falls back to searching for a group with exactly that name or path.
func (gm *GitlabManager) resolveGroup(ctx context.Context, name string) (*gitlab.Group, error) {
	group, _, err := gm.client.Groups.GetGroup(name, nil, gitlab.WithContext(ctx))
	if err == nil {
		return group, nil
	}
	if !isStatus(err, http.StatusNotFound) {
		return nil, fmt.Errorf("failed to fetch group '%s': %w", name, err)
	}

	log.Debug().Msgf("Group with path %s not found, searching by name", name)
	candidates, _, err := gm.client.Groups.ListGroups(&gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{PerPage: perPage},
		Search:      gitlab.Ptr(name),
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to search for group '%s': %w", name, err)
	}

	var matches []*gitlab.Group
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Name, name) || strings.EqualFold(candidate.Path, name) || strings.EqualFold(candidate.FullPath, name) {
			matches = append(matches, candidate)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("group '%s' not found or not accessible", name)
	case 1:
		return matches[0], nil
	default:
		paths := make([]string, 0, len(matches))
		for _, match := range matches {
			paths = append(paths, match.FullPath)
		}
		return nil, fmt.Errorf("group name '%s' is ambiguous, use the full path: %s", name, strings.Join(paths, ", "))
	}
}

// listProjects fetches all projects of a group, including projects in all of its subgroups.
func (gm *GitlabManager) listProjects(ctx context.Context, group *gitlab.Group, opts ListOptions) ([]*gitlab.Project, error) {
	options := &gitlab.ListGroupProjectsOptions{
		ListOptions:      gitlab.ListOptions{PerPage: perPage, Page: 1},
		IncludeSubGroups: gitlab.Ptr(true),
		Simple:           gitlab.Ptr(true),
	}
	if !opts.IncludeArchived {
		options.Archived = gitlab.Ptr(false)
	}

	var all []*gitlab.Project
	for {
		projects, resp, err := gm.client.Groups.ListGroupProjects(group.ID, options, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to list projects for group %s: %w", group.FullPath, err)
		}
		all = append(all, projects...)
		log.Debug().Msgf("Fetched %d projects from page %d of group %s", len(projects), options.Page, group.FullPath)

		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}

	return all, nil
}

// isStatus reports whether err is a GitLab API error with the given HTTP status code.
func isStatus(err error, status int) bool {
	var apiErr *gitlab.ErrorResponse
	return errors.As(err, &apiErr) && apiErr.Response != nil && apiErr.Response.StatusCode == status
}
