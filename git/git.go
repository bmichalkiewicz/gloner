package git

import (
	"errors"
	"fmt"
	"gloner/exec"
	"gloner/repositories"
	"os"
	"path/filepath"
	"strings"
)

func Clone(url string, path string) (cloned bool, err error) {
	if err := repositories.Validate(url); err != nil {
		return false, err
	}

	// resolve repo path from ssh url
	repoPath, _ := getPathFromSSH(url)
	dir := filepath.Join(path, repoPath)

	// check if repo already exists
	if _, err := os.Stat(dir); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	// make sure parent dirs exist
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return false, err
	}

	args := []string{"clone"}

	args = append(args, url, dir)

	_, err = exec.New().Silent().Go("git", args...)
	if err != nil {
		return false, err
	}

	return false, nil
}

func getPathFromSSH(sshURL string) (string, error) {
	parts := strings.SplitN(sshURL, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("problem with getting path from SSH URL")
	}

	host := strings.TrimPrefix(parts[0], "git@")
	path := strings.TrimSuffix(parts[1], ".git")
	return filepath.Join(host, path), nil
}
