package repositories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/google/go-github/v93/github"
	"github.com/rs/zerolog/log"
)

// GithubManager handles GitHub API operations for fetching organization and user repositories.
type GithubManager struct {
	client *github.Client
}

// NewGithub creates a new GithubManager. The token is optional (only public repositories
// are visible without it) and url is only needed for GitHub Enterprise.
func NewGithub(token, url string) (*GithubManager, error) {
	var options []github.ClientOptionsFunc
	if token != "" {
		options = append(options, github.WithAuthToken(token))
	}
	if url != "" {
		apiURL := strings.TrimSuffix(url, "/") + "/api/v3/"
		uploadURL := strings.TrimSuffix(url, "/") + "/api/uploads/"
		log.Debug().Msgf("Initializing GitHub Enterprise client with API URL: %s", apiURL)
		options = append(options, github.WithEnterpriseURLs(apiURL, uploadURL))
	}

	client, err := github.NewClient(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return &GithubManager{client: client}, nil
}

// GetOwnerRepositories fetches all repositories of the given organizations or users.
// Each owner becomes one Group.
func (gm *GithubManager) GetOwnerRepositories(ctx context.Context, owners []string, opts ListOptions) ([]*Group, error) {
	if len(owners) == 0 {
		return nil, fmt.Errorf("no owners specified")
	}

	byOwner := map[string]*Group{}
	for _, owner := range owners {
		repos, err := gm.listRepositories(ctx, owner)
		if err != nil {
			return nil, err
		}

		group := &Group{Name: owner}
		for _, repo := range repos {
			switch {
			case repo.GetSSHURL() == "":
				continue
			case repo.GetArchived() && !opts.IncludeArchived:
				log.Debug().Msgf("Skipping archived repository %s", repo.GetFullName())
				continue
			case repo.GetFork() && !opts.IncludeForks:
				log.Debug().Msgf("Skipping fork %s", repo.GetFullName())
				continue
			}
			group.Projects = append(group.Projects, Project{URL: repo.GetSSHURL()})
		}

		if len(group.Projects) > 0 {
			byOwner[owner] = group
		}
	}

	return sortGroups(byOwner), nil
}

// listRepositories lists repositories of an organization, falling back to a user account
// when no organization with that name exists.
func (gm *GithubManager) listRepositories(ctx context.Context, owner string) ([]*github.Repository, error) {
	var all []*github.Repository

	orgOptions := &github.RepositoryListByOrgOptions{ListOptions: github.ListOptions{PerPage: perPage}}
	for {
		repos, resp, err := gm.client.Repositories.ListByOrg(ctx, owner, orgOptions)
		if err != nil {
			if isGithubStatus(err, http.StatusNotFound) && len(all) == 0 {
				log.Debug().Msgf("Organization %s not found, trying user", owner)
				return gm.listUserRepositories(ctx, owner)
			}
			return nil, fmt.Errorf("failed to list repositories for organization %s: %w", owner, err)
		}
		all = append(all, repos...)

		if resp.NextPage == 0 {
			return all, nil
		}
		orgOptions.Page = resp.NextPage
	}
}

// listUserRepositories lists repositories owned by a user.
func (gm *GithubManager) listUserRepositories(ctx context.Context, user string) ([]*github.Repository, error) {
	var all []*github.Repository

	options := &github.RepositoryListByUserOptions{
		Type:        "owner",
		ListOptions: github.ListOptions{PerPage: perPage},
	}
	for {
		repos, resp, err := gm.client.Repositories.ListByUser(ctx, user, options)
		if err != nil {
			if isGithubStatus(err, http.StatusNotFound) {
				return nil, fmt.Errorf("organization or user '%s' not found", user)
			}
			return nil, fmt.Errorf("failed to list repositories for user %s: %w", user, err)
		}
		all = append(all, repos...)

		if resp.NextPage == 0 {
			return all, nil
		}
		options.Page = resp.NextPage
	}
}

// isGithubStatus reports whether err is a GitHub API error with the given HTTP status code.
func isGithubStatus(err error, status int) bool {
	var apiErr *github.ErrorResponse
	return errors.As(err, &apiErr) && apiErr.Response != nil && apiErr.Response.StatusCode == status
}

// sortGroups returns groups ordered by name, with projects ordered by URL, for stable output.
func sortGroups(groups map[string]*Group) []*Group {
	result := make([]*Group, 0, len(groups))
	for _, group := range groups {
		sort.Slice(group.Projects, func(i, j int) bool { return group.Projects[i].URL < group.Projects[j].URL })
		result = append(result, group)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
