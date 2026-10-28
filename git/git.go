// Package git provides functionality for Git operations: cloning and updating repositories.
// It creates directory structures following Go package conventions (host/owner/repo).
package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bmichalkiewicz/gloner/exec"
	"github.com/bmichalkiewicz/gloner/repositories"
	"github.com/rs/zerolog/log"
)

// Status describes the outcome of a Clone call.
type Status int

const (
	// StatusCloned means the repository was cloned
	StatusCloned Status = iota
	// StatusExists means the repository was already present and left untouched
	StatusExists
	// StatusUpdated means an existing repository received new commits
	StatusUpdated
	// StatusUpToDate means an existing repository was pulled but had no new commits
	StatusUpToDate
	// StatusPlanned means the repository would be cloned or pulled (dry run)
	StatusPlanned
)

// Options controls how repositories are cloned.
type Options struct {
	Depth  int  // Create a shallow clone with this many commits (0 = full clone)
	Update bool // Pull existing repositories instead of skipping them
	DryRun bool // Only report what would be done
}

// Dir returns the directory a repository URL is cloned into, e.g.
// git@github.com:user/repo.git with base ~/git gives ~/git/github.com/user/repo.
func Dir(url string, base string) (string, error) {
	host, repoPath, err := repositories.ParseURL(url)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, host, filepath.FromSlash(repoPath))

	// ParseURL rejects ".." segments, this is a second line of defence against escaping base
	rel, err := filepath.Rel(base, dir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("repository path %s escapes base directory %s", dir, base)
	}

	return dir, nil
}

// Clone clones a repository into its structured directory under base.
// Existing repositories are skipped, or pulled when opts.Update is set.
func Clone(ctx context.Context, url string, base string, opts Options) (Status, error) {
	dir, err := Dir(url, base)
	if err != nil {
		return 0, fmt.Errorf("invalid repository URL: %w", err)
	}

	log.Debug().Msgf("Target directory: %s", dir)

	exists, err := isRepository(dir)
	if err != nil {
		return 0, err
	}
	if exists {
		if !opts.Update {
			return StatusExists, nil
		}
		if opts.DryRun {
			return StatusPlanned, nil
		}
		changed, err := Pull(ctx, dir)
		if err != nil {
			return 0, err
		}
		if changed {
			return StatusUpdated, nil
		}
		return StatusUpToDate, nil
	}

	if opts.DryRun {
		return StatusPlanned, nil
	}

	parentDir := filepath.Dir(dir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return 0, fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	args := []string{"clone", "--quiet"}
	if opts.Depth > 0 {
		args = append(args, "--depth", strconv.Itoa(opts.Depth))
	}
	args = append(args, "--", url, dir)

	if _, err := exec.Run(ctx, "", "git", args...); err != nil {
		// A killed clone (e.g. Ctrl+C) leaves a partial directory that would be treated as cloned next time
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			log.Warn().Err(removeErr).Msgf("Failed to remove partial clone %s", dir)
		}
		return 0, fmt.Errorf("git clone failed for %s: %w", url, err)
	}

	log.Debug().Msgf("Successfully cloned %s to %s", url, dir)
	return StatusCloned, nil
}

// Pull fast-forwards an existing repository. It reports whether new commits were fetched.
func Pull(ctx context.Context, dir string) (bool, error) {
	before := head(ctx, dir)

	if _, err := exec.Run(ctx, dir, "git", "pull", "--ff-only", "--quiet"); err != nil {
		return false, fmt.Errorf("git pull failed in %s: %w", dir, err)
	}

	return head(ctx, dir) != before, nil
}

// head returns the current commit of a repository, or an empty string for empty repositories.
func head(ctx context.Context, dir string) string {
	out, err := exec.Run(ctx, dir, "git", "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// isRepository reports whether dir is a git repository. A directory that exists
// but is not a repository is an error, so it is never silently skipped or overwritten.
func isRepository(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%s exists and is not a directory", dir)
	}

	// .git is a directory for regular clones and a file for worktrees
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return false, fmt.Errorf("%s exists but is not a git repository", dir)
	}
	return true, nil
}

// FindRepositories returns all git repositories below base. Nested repositories
// (e.g. submodules) are not descended into.
func FindRepositories(base string) ([]string, error) {
	var repos []string

	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				log.Warn().Msgf("Skipping %s: permission denied", path)
				return fs.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			repos = append(repos, path)
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan %s: %w", base, err)
	}

	return repos, nil
}
