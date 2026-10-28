# 🚀 gloner

<img src="assets/logo.png" alt="gloner logo" width="120"/>

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bmichalkiewicz/gloner)](go.mod)
[![Release](https://img.shields.io/github/v/release/bmichalkiewicz/gloner)](https://github.com/bmichalkiewicz/gloner/releases)

**gloner** is a lightweight CLI tool for fetching and cloning repositories into user-specified locations.  
It organizes projects into a structured Go-style package tree 📦.

---

## ✨ Features

- 🌐 Fetch repositories from **GitLab** groups (with all subgroups) and **GitHub** organizations or users
- 📂 Organize repos into structured paths: `~/git/<host>/<owner>/<repo>`
- ⚡ Concurrent cloning with progress per group and a summary at the end
- 🔄 Keep everything up to date with `gloner pull` or `--update`
- 🧪 `--dry-run`, shallow clones (`--depth`), archived/fork filters

---

## 🛠️ Installation

```bash
go install github.com/bmichalkiewicz/gloner@latest
```

---

## 📜 Commands

### 🦊 GitLab
Clones all projects of the given groups, including subgroups. Groups are given by full path
(`group/subgroup`) or by name.
```bash
gloner gitlab -g terraform -t "$GITLAB_TOKEN" -u gitlab.custom.com
gloner gitlab -g platform/infra,security --update     # pull repositories that already exist
gloner gitlab -g terraform --dry-run                  # only show what would be cloned
gloner gitlab -g terraform --include-archived --depth 1
```

### 🐙 GitHub
Clones repositories of organizations or users. Without a token only public repositories are visible.
```bash
gloner github -o kubernetes-sigs -t "$GITHUB_TOKEN"
gloner github -o my-org --include-forks --include-archived
gloner github -o my-org -u https://github.example.com  # GitHub Enterprise
```

### 📥 Clone
Clones repositories by URL. SSH (`git@host:owner/repo.git`, `ssh://git@host:2222/owner/repo.git`)
and HTTPS URLs are supported.
```bash
gloner clone git@github.com:bmichalkiewicz/gloner.git https://github.com/urfave/cli.git
gloner clone -u git@github.com:bmichalkiewicz/gloner.git
```

### 🔄 Pull
Updates (`git pull --ff-only`) every repository below the destination directory, optionally
limited to path prefixes.
```bash
gloner pull
gloner pull github.com/my-org gitlab.com/platform
```

### Common options

| Option | Description |
|---|---|
| `-d, --destination` | Base directory (default: `path` from config, then `~/git`, env `GLONER_DESTINATION`) |
| `-n, --dry-run` | Only print what would be done |
| `--update` | Pull repositories that are already cloned |
| `--depth N` | Shallow clone with N commits |
| `-j, --concurrency N` | Maximum number of simultaneous git operations (default: 2 × CPUs) |
| `--debug` | Debug logging |

Press `Ctrl+C` to stop; running clones are cancelled and partial directories removed.

## ⚙️ Configuration

The configuration file is created on first run at:

```
~/.config/gloner/config.toml
```

It contains tokens, so it is kept readable only by its owner (`0600`).
Command line flags and environment variables (`GITLAB_TOKEN`, `GITLAB_URL`, `GITHUB_TOKEN`/`GH_TOKEN`,
`GITHUB_URL`) override values from the file.

### Example `config.toml`:

```toml
# Base directory for cloned repositories (default: ~/git)
path = '~/git'

[gitlab]
token = ''
url = 'https://gitlab.com'

[github]
token = ''
# GitHub Enterprise URL, leave empty for github.com
url = ''
```
---

## 📌 Roadmap

- [ ] Support for **Bitbucket**
- [ ] Interactive mode for selecting repositories
- [x] Config file support (more options)
- [x] Support for **GitHub**

---

## 🤝 Contributing

Pull requests are welcome!  
For major changes, please open an issue first to discuss what you’d like to change.  

---

## 📄 License

[MIT](LICENSE)
