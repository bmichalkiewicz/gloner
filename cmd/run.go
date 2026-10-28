package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/bmichalkiewicz/gloner/config"
	"github.com/bmichalkiewicz/gloner/facts"
	"github.com/bmichalkiewicz/gloner/git"
	"github.com/bmichalkiewicz/gloner/repositories"
	"github.com/chelnak/ysmrr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// job is a single repository operation, e.g. cloning or pulling one repository.
type job struct {
	name string
	run  func(ctx context.Context) (git.Status, error)
}

// batch is a named set of jobs shown as one progress line, e.g. a GitLab group.
type batch struct {
	name string
	jobs []job
}

// summary counts job outcomes across all batches.
type summary struct {
	mu       sync.Mutex
	statuses map[git.Status]int
	errs     []error
}

func (s *summary) add(status git.Status, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.errs = append(s.errs, err)
		return
	}
	s.statuses[status]++
}

// statusText describes a job outcome for log output.
func statusText(status git.Status) string {
	switch status {
	case git.StatusCloned:
		return "cloned"
	case git.StatusExists:
		return "already exists"
	case git.StatusUpdated:
		return "updated"
	case git.StatusUpToDate:
		return "up to date"
	case git.StatusPlanned:
		return "planned"
	default:
		return "unknown"
	}
}

// Common flags shared by commands that clone or update repositories.
var (
	dryRunFlag = &cli.BoolFlag{
		Name:    "dry-run",
		Aliases: []string{"n"},
		Usage:   "Only print what would be done",
	}
	updateFlag = &cli.BoolFlag{
		Name:  "update",
		Usage: "Pull (fast-forward only) repositories that are already cloned",
	}
	depthFlag = &cli.IntFlag{
		Name:  "depth",
		Usage: "Create shallow clones with the given number of commits (0 = full clone)",
	}
	includeArchivedFlag = &cli.BoolFlag{
		Name:  "include-archived",
		Usage: "Also clone archived repositories",
	}
	concurrencyFlag = &cli.IntFlag{
		Name:    "concurrency",
		Aliases: []string{"j"},
		Usage:   "Maximum number of simultaneous git operations",
		Value:   runtime.NumCPU() * 2,
	}
)

// cloneFlags returns the flags used by commands that clone repositories.
func cloneFlags() []cli.Flag {
	return []cli.Flag{dryRunFlag, updateFlag, depthFlag, concurrencyFlag}
}

// gitOptions builds git clone options from the command flags.
func gitOptions(cmd *cli.Command) (git.Options, error) {
	depth := cmd.Int("depth")
	if depth < 0 {
		return git.Options{}, fmt.Errorf("--depth must not be negative")
	}
	return git.Options{
		Depth:  depth,
		Update: cmd.Bool("update"),
		DryRun: cmd.Bool("dry-run"),
	}, nil
}

// destination returns the base directory for repositories.
// Precedence: --destination flag, "path" in the config file, default (~/git).
func destination(cmd *cli.Command) (string, error) {
	path := cmd.String("destination")
	if !cmd.IsSet("destination") && config.Settings.Path != "" {
		path = config.Settings.Path
	}

	abs, err := filepath.Abs(facts.ExpandHome(path))
	if err != nil {
		return "", fmt.Errorf("invalid destination %s: %w", path, err)
	}
	return abs, nil
}

// cloneBatches turns repository groups into clone jobs.
func cloneBatches(groups []*repositories.Group, base string, opts git.Options) []batch {
	batches := make([]batch, 0, len(groups))
	for _, group := range groups {
		b := batch{name: group.Name}
		for _, project := range group.Projects {
			url := project.URL
			b.jobs = append(b.jobs, job{
				name: url,
				run: func(ctx context.Context) (git.Status, error) {
					return git.Clone(ctx, url, base, opts)
				},
			})
		}
		batches = append(batches, b)
	}
	return batches
}

// runBatches runs all jobs with at most concurrency simultaneous git operations,
// shows progress per batch and prints a summary. It returns an error if any job failed.
func runBatches(ctx context.Context, batches []batch, concurrency int, dryRun bool) error {
	if concurrency < 1 {
		return fmt.Errorf("--concurrency must be at least 1")
	}

	total := 0
	for _, b := range batches {
		total += len(b.jobs)
	}
	if total == 0 {
		log.Warn().Msg("No repositories found to process")
		return nil
	}

	// Spinners and log lines both write to stderr and would garble each other,
	// so spinners are only used on an interactive terminal without debug output.
	useSpinners := !dryRun && term.IsTerminal(int(os.Stderr.Fd())) && log.Logger.GetLevel() > zerolog.DebugLevel

	var sm ysmrr.SpinnerManager
	if useSpinners {
		sm = ysmrr.NewSpinnerManager(ysmrr.WithWriter(os.Stderr))
		sm.Start()
	}

	var (
		wg        sync.WaitGroup
		semaphore = make(chan struct{}, concurrency)
		result    = &summary{statuses: map[git.Status]int{}}
	)

	for _, b := range batches {
		if len(b.jobs) == 0 {
			continue
		}

		var spinner *ysmrr.Spinner
		if useSpinners {
			spinner = sm.AddSpinner(fmt.Sprintf("[%s] 0/%d", b.name, len(b.jobs)))
		}

		var (
			batchMu   sync.Mutex
			done      int
			failed    int
			batchWg   sync.WaitGroup
			batchName = b.name
			batchSize = len(b.jobs)
		)

		for _, j := range b.jobs {
			wg.Add(1)
			batchWg.Add(1)
			go func() {
				defer wg.Done()
				defer batchWg.Done()

				var (
					status git.Status
					err    error
				)
				select {
				case semaphore <- struct{}{}:
					status, err = j.run(ctx)
					<-semaphore
				case <-ctx.Done():
					err = ctx.Err()
				}
				if err != nil {
					err = fmt.Errorf("[%s] %s: %w", batchName, j.name, err)
				}
				result.add(status, err)

				if !useSpinners {
					switch {
					case err != nil:
						log.Error().Msg(err.Error())
					case status == git.StatusExists || status == git.StatusUpToDate:
						log.Debug().Msgf("[%s] %s: %s", batchName, j.name, statusText(status))
					default:
						log.Info().Msgf("[%s] %s: %s", batchName, j.name, statusText(status))
					}
					return
				}

				batchMu.Lock()
				defer batchMu.Unlock()
				done++
				if err != nil {
					failed++
				}
				spinner.UpdateMessagef("[%s] %d/%d", batchName, done, batchSize)
			}()
		}

		if useSpinners {
			wg.Add(1)
			go func() {
				defer wg.Done()
				batchWg.Wait()
				if failed > 0 {
					spinner.ErrorWithMessagef("[%s] %d of %d failed", batchName, failed, batchSize)
				} else {
					spinner.CompleteWithMessagef("[%s] %d done", batchName, batchSize)
				}
			}()
		}
	}

	wg.Wait()
	if useSpinners {
		sm.Stop()
		for _, err := range result.errs {
			log.Error().Msg(err.Error())
		}
	}

	s := result.statuses
	if dryRun {
		log.Info().Msgf("Dry run: %d planned, %d already present", s[git.StatusPlanned], s[git.StatusExists]+s[git.StatusUpToDate])
	} else {
		log.Info().Msgf("Done: %d cloned, %d updated, %d up to date, %d skipped, %d failed",
			s[git.StatusCloned], s[git.StatusUpdated], s[git.StatusUpToDate], s[git.StatusExists], len(result.errs))
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("interrupted: %w", err)
	}
	if len(result.errs) > 0 {
		return fmt.Errorf("%d of %d repositories failed", len(result.errs), total)
	}
	return nil
}
