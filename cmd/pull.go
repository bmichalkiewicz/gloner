package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmichalkiewicz/gloner/git"
	"github.com/urfave/cli/v3"
)

// Pull returns a CLI command that updates all repositories below the destination directory.
func Pull() *cli.Command {
	return &cli.Command{
		Name:      "pull",
		Usage:     "Update all cloned repositories (git pull --ff-only)",
		ArgsUsage: "[PREFIX...]",
		Description: "Without arguments every repository below the destination directory is updated.\n" +
			"Prefixes limit the update to matching paths, e.g. 'gloner pull github.com/my-org'.",
		Flags: []cli.Flag{dryRunFlag, concurrencyFlag},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			base, err := destination(cmd)
			if err != nil {
				return err
			}

			repos, err := git.FindRepositories(base)
			if err != nil {
				return err
			}

			dryRun := cmd.Bool("dry-run")
			byParent := map[string]*batch{}
			for _, dir := range repos {
				rel, err := filepath.Rel(base, dir)
				if err != nil {
					return fmt.Errorf("failed to resolve %s: %w", dir, err)
				}
				rel = filepath.ToSlash(rel)
				if !matchesPrefix(rel, cmd.Args().Slice()) {
					continue
				}

				parent := filepath.ToSlash(filepath.Dir(rel))
				if byParent[parent] == nil {
					byParent[parent] = &batch{name: parent}
				}
				byParent[parent].jobs = append(byParent[parent].jobs, job{
					name: rel,
					run: func(ctx context.Context) (git.Status, error) {
						if dryRun {
							return git.StatusPlanned, nil
						}
						changed, err := git.Pull(ctx, dir)
						if err != nil {
							return 0, err
						}
						if changed {
							return git.StatusUpdated, nil
						}
						return git.StatusUpToDate, nil
					},
				})
			}

			batches := make([]batch, 0, len(byParent))
			for _, b := range byParent {
				batches = append(batches, *b)
			}
			sort.Slice(batches, func(i, j int) bool { return batches[i].name < batches[j].name })

			return runBatches(ctx, batches, int(cmd.Int("concurrency")), dryRun)
		},
	}
}

// matchesPrefix reports whether path is equal to or below one of the prefixes.
// An empty prefix list matches everything.
func matchesPrefix(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, prefix := range prefixes {
		prefix = strings.Trim(filepath.ToSlash(prefix), "/")
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
