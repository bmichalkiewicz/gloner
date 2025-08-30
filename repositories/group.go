// Package repositories provides functionality for repository management,
// including URL validation and data structures for organizing repositories into groups.
package repositories

import (
	"fmt"
	"regexp"
	"sync"
)

// sshURLPattern defines the regex pattern for validating SSH URLs
// Format: git@hostname:owner/repo.git or git@hostname:group/subgroup/repo.git
var sshURLPattern = `^git@([a-zA-Z0-9.-]+):([a-zA-Z0-9_.-]+)(/[a-zA-Z0-9_./-]*)?\.git$`
var sshRegex *regexp.Regexp
var regexOnce sync.Once

// Group represents a collection of repositories, typically from a GitLab group or organization.
type Group struct {
	Name     string    `yaml:"name" json:"name"`         // Group name
	Projects []Project `yaml:"projects" json:"projects"` // List of projects in the group
}

// Project represents a single repository with its SSH URL.
type Project struct {
	URL string `yaml:"url" json:"url"` // SSH URL of the repository
}

// Validate checks if the provided URL is a valid SSH URL for Git repositories.
// It expects the format: git@hostname:owner/repo.git
func Validate(url string) error {
	if url == "" {
		return fmt.Errorf("SSH URL cannot be empty")
	}

	// Compile regex only once using sync.Once for thread safety
	regexOnce.Do(func() {
		sshRegex = regexp.MustCompile(sshURLPattern)
	})

	if !sshRegex.MatchString(url) {
		return fmt.Errorf("invalid SSH URL format. Expected: git@hostname:owner/repo.git, got: %s", url)
	}

	return nil
}

// Decode extracts repository information from an SSH URL and creates a Group.
// This is useful for creating a single-project group from a repository URL.
func Decode(url string) (*Group, error) {
	// Validate the SSH URL first
	if err := Validate(url); err != nil {
		return nil, fmt.Errorf("failed to decode URL: %w", err)
	}

	// Use the compiled regex from Validate function
	regexOnce.Do(func() {
		sshRegex = regexp.MustCompile(sshURLPattern)
	})

	submatches := sshRegex.FindStringSubmatch(url)
	if len(submatches) < 3 {
		return nil, fmt.Errorf("failed to extract repository information from SSH URL: %s", url)
	}

	// Extract components: [full_match, hostname, owner, path_suffix]
	hostname := submatches[1]
	owner := submatches[2]
	path := owner
	if len(submatches) > 3 && submatches[3] != "" {
		// If there's a path component, combine owner with path
		path = owner + submatches[3]
	}

	// Create a group name from hostname and path
	groupName := fmt.Sprintf("%s/%s", hostname, path)

	return &Group{
		Name: groupName,
		Projects: []Project{
			{
				URL: url,
			},
		},
	}, nil
}
