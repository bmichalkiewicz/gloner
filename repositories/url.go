// Package repositories provides functionality for repository management,
// including URL parsing and data structures for organizing repositories into groups.
package repositories

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	hostPattern    = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	segmentPattern = regexp.MustCompile(`^[a-zA-Z0-9_.~-]+$`)
)

// Group represents a collection of repositories, typically from a GitLab group or GitHub organization.
type Group struct {
	Name     string    `json:"name"`     // Group name
	Projects []Project `json:"projects"` // List of projects in the group
}

// Project represents a single repository with its clone URL.
type Project struct {
	URL string `json:"url"` // Clone URL of the repository
}

// ParseURL extracts the hostname and repository path from a clone URL.
// Supported formats:
//   - git@hostname:owner/repo.git (scp-like SSH)
//   - ssh://git@hostname[:port]/owner/repo.git
//   - https://hostname/owner/repo.git
//
// The returned path has no ".git" suffix and never contains "." or ".." segments,
// so it is safe to join with a base directory.
func ParseURL(raw string) (host string, path string, err error) {
	if raw == "" {
		return "", "", fmt.Errorf("repository URL cannot be empty")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", "", fmt.Errorf("invalid repository URL %s: %w", raw, err)
		}
		switch u.Scheme {
		case "ssh", "https", "http":
		default:
			return "", "", fmt.Errorf("unsupported URL scheme %q in %s", u.Scheme, raw)
		}
		host, path = u.Hostname(), u.Path
	} else {
		// scp-like syntax: [user@]host:path, the user part is required to tell it apart from local paths
		at := strings.Index(raw, "@")
		colon := strings.Index(raw, ":")
		if at < 0 || colon < at {
			return "", "", fmt.Errorf("invalid repository URL format. Expected: git@hostname:owner/repo.git, got: %s", raw)
		}
		host, path = raw[at+1:colon], raw[colon+1:]
	}

	if !hostPattern.MatchString(host) {
		return "", "", fmt.Errorf("invalid hostname %q in %s", host, raw)
	}

	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	if path == "" {
		return "", "", fmt.Errorf("empty repository path in %s", raw)
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." || !segmentPattern.MatchString(segment) {
			return "", "", fmt.Errorf("invalid path segment %q in %s", segment, raw)
		}
	}

	return host, path, nil
}

// Validate checks if the provided URL is a supported Git repository URL.
func Validate(url string) error {
	_, _, err := ParseURL(url)
	return err
}
