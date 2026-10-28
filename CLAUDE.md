# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

`gloner` is a Go CLI tool for fetching and cloning repositories into structured paths, similar to Go's package management system. It organizes repositories using a directory structure like `~/git/hostname/owner/repo`.

## Common Development Commands

### Task Runner (Taskfile)
This project uses [Task](https://taskfile.dev/) for build automation. All commands are defined in `Taskfile.yml`:

```bash
# Build the application
task build

# Run tests
task test

# Run linter
task lint

# Format code
task fmt

# Run go vet
task vet

# Check for vulnerabilities
task vuln

# Install dependencies and tidy modules
task  # runs default task (go mod tidy)

# Install binary to /usr/local/bin
task install

# Upgrade all dependencies
task upgrade-deps
```

### Go Commands
```bash
# Build manually
go build -ldflags '-s -w' -o ./gloner

# Run tests
go test ./...

# Install dependencies
go mod tidy
```

## Architecture Overview

The module path is `github.com/bmichalkiewicz/gloner`. The version is injected with `-X main.version=...`.

### Package Structure
- **`main.go`**: CLI entry point using `urfave/cli/v3`; loads config in the `Before` hook (after `--debug` is parsed) and cancels the context on Ctrl+C
- **`cmd/`**: Commands (`gitlab.go`, `github.go`, `clone.go`, `pull.go`) and the shared job runner (`run.go`)
- **`config/`**: TOML configuration (`~/.config/gloner/config.toml`, mode 0600) and URL normalization
- **`git/`**: Clone/pull logic, target directory resolution, repository discovery
- **`repositories/`**: GitLab and GitHub API integration, clone URL parsing (`url.go`)
- **`facts/`**: Application constants, home directory helpers
- **`exec/`**: Command execution wrapper that captures stderr into errors

### Commands
1. **`gitlab`** - Clone all projects of GitLab groups. Groups are resolved by full path (fallback: exact name search) and listed with `include_subgroups`, paginated.
2. **`github`** - Clone repositories of GitHub organizations (fallback: users). Supports GitHub Enterprise.
3. **`clone`** - Clone repositories by URL (args or `--url`).
4. **`pull`** - `git pull --ff-only` every repository below the destination, optionally filtered by path prefix.

Shared flags: `--dry-run`, `--update`, `--depth`, `--concurrency`, plus `--include-archived` (gitlab/github).

### Configuration Precedence
Flag/env var > config file > default. Destination: `--destination`/`GLONER_DESTINATION` > `path` in config > `~/git`.

### Path Resolution Logic
`repositories.ParseURL()` accepts `git@host:path`, `ssh://user@host[:port]/path` and `https://host/path`,
strips `.git` and rejects `.`/`..` segments. `git.Dir()` joins it with the base directory:
- Input: `git@github.com:user/repo.git`
- Output: `~/git/github.com/user/repo`

### Concurrency Model
- `cmd/run.go:runBatches()` runs jobs grouped into batches (one per GitLab namespace / GitHub owner / parent directory)
- A shared semaphore limits simultaneous git operations (`--concurrency`, default `runtime.NumCPU() * 2`)
- Results and errors are collected under a mutex (no bounded channels), and a summary is logged at the end
- `ysmrr` spinners are used only on an interactive stderr without `--debug` or `--dry-run`; otherwise one log line per repository

### Dependencies
- **CLI Framework**: `urfave/cli/v3`
- **GitLab API**: `gitlab.com/gitlab-org/api/client-go`
- **GitHub API**: `google/go-github`
- **Logging**: `rs/zerolog` with console output
- **Config**: `pelletier/go-toml/v2`
- **UI**: `chelnak/ysmrr` for progress spinners

### Error Handling Patterns
- Configuration errors are returned from the `Before` hook and end the program
- CLI command errors bubble up through return values
- Every failed repository is logged; the command returns an error with the failure count
- Already-cloned repositories are skipped (or pulled with `--update`); an existing directory that is not a git repository is an error
- A failed or interrupted clone removes its partial directory
