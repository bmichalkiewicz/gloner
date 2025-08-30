// Package git provides functionality for Git operations, primarily repository cloning.
// It handles SSH URL parsing and creates directory structures following Go package conventions.
package git

import (
	"errors"
	"fmt"
	"gloner/exec"
	"gloner/repositories"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Clone attempts to clone a repository from the given SSH URL to the specified base path.
// It creates a directory structure based on the repository's hostname and path.
// Returns true if the repository already exists, false if it was newly cloned.
// Returns an error if the operation fails.
func Clone(url string, path string) (alreadyExists bool, err error) {
	if err := repositories.Validate(url); err != nil {
		return false, fmt.Errorf("invalid SSH URL: %w", err)
	}

	// Resolve repository path from SSH URL
	repoPath, err := getPathFromSSH(url)
	if err != nil {
		return false, fmt.Errorf("failed to parse repository path: %w", err)
	}
	dir := filepath.Join(path, repoPath)

	log.Debug().Msgf("Target directory: %s", dir)

	// Check if repository directory already exists
	if _, err := os.Stat(dir); err == nil {
		log.Debug().Msgf("Directory already exists: %s", dir)
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("failed to check directory existence: %w", err)
	}

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(dir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}
	log.Debug().Msgf("Created parent directories: %s", parentDir)

	// Execute git clone command
	args := []string{"clone", url, dir}
	log.Debug().Msgf("Executing: git %s", strings.Join(args, " "))

	_, err = exec.New().Silent().Go("git", args...)
	if err != nil {
		return false, fmt.Errorf("git clone failed for %s: %w", url, err)
	}

	log.Debug().Msgf("Successfully cloned %s to %s", url, dir)

	return false, nil
}

// getPathFromSSH extracts the repository path from an SSH URL.
// Converts git@hostname:owner/repo.git to hostname/owner/repo.
// This creates a directory structure similar to Go's package system.
func getPathFromSSH(sshURL string) (string, error) {
	parts := strings.SplitN(sshURL, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid SSH URL format: expected 'git@hostname:path', got %s", sshURL)
	}

	// Extract hostname (remove git@ prefix)
	host := strings.TrimPrefix(parts[0], "git@")
	if host == parts[0] {
		return "", fmt.Errorf("SSH URL must start with 'git@': %s", sshURL)
	}

	// Extract repository path (remove .git suffix)
	repoPath := strings.TrimSuffix(parts[1], ".git")
	if repoPath == "" {
		return "", fmt.Errorf("empty repository path in SSH URL: %s", sshURL)
	}

	// Combine hostname and repository path
	finalPath := filepath.Join(host, repoPath)
	log.Debug().Msgf("Parsed SSH URL %s to path %s", sshURL, finalPath)
	return finalPath, nil
}
