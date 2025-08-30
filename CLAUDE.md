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

### Package Structure
- **`main.go`**: CLI application entry point using `urfave/cli/v3` framework
- **`cmd/`**: Command implementations (`gitlab.go`, `clone.go`)
- **`config/`**: Configuration management with TOML support
- **`git/`**: Git operations and repository cloning logic
- **`repositories/`**: GitLab API integration and repository management
- **`facts/`**: Application constants and utility functions
- **`exec/`**: Command execution wrapper

### Core Functionality
The application provides two main commands:

1. **`gitlab`** - Bulk clone repositories from GitLab groups
   - Supports nested group traversal
   - Concurrent cloning with progress spinners
   - Uses GitLab API v4 client
   
2. **`clone`** - Clone individual repositories
   - Validates SSH URLs
   - Creates structured directory paths

### Configuration System
- Configuration stored in `~/.config/gloner/config.toml`
- Auto-generated TOML config with GitLab URL and token settings
- Default GitLab URL: `https://gitlab.com`
- Supports custom GitLab instances via `-u/--url` flag

### Path Resolution Logic
Repositories are cloned using Go-style paths extracted from SSH URLs:
- Input: `git@github.com:user/repo.git`
- Output: `~/git/github.com/user/repo`

The `git/git.go:getPathFromSSH()` function handles URL parsing and path construction.

### Concurrency Model
- **Group-level concurrency**: Each GitLab group is processed in parallel
- **Project-level concurrency**: Within each group, repositories are cloned concurrently with CPU-based limits
- **Concurrency control**: Uses semaphores to limit simultaneous git clone operations (`runtime.NumCPU() * 2`)
- **Progress indication**: `ysmrr` spinner library provides real-time visual feedback for each group
- **Error handling**: Thread-safe error collection using channels and mutex protection

### Dependencies
- **CLI Framework**: `urfave/cli/v3` for command structure
- **GitLab API**: `gitlab.com/gitlab-org/api/client-go` for repository fetching
- **Logging**: `rs/zerolog` with console output
- **Config**: `pelletier/go-toml/v2` for TOML parsing
- **UI**: `chelnak/ysmrr` for progress spinners

### Error Handling Patterns
- Configuration errors are fatal and logged with `log.Fatal()`
- CLI command errors bubble up through return values
- Concurrent operations collect errors and use `errors.Join()` for aggregation
- Already-cloned repositories are detected and skipped (not errors)