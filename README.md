# 🚀 gloner

<img src="assets/logo.png" alt="gloner logo" width="120"/>

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bmichalkiewicz/gloner)](go.mod)
[![Release](https://img.shields.io/github/v/release/bmichalkiewicz/gloner)](https://github.com/bmichalkiewicz/gloner/releases)

**gloner** is a lightweight CLI tool for fetching and cloning repositories into user-specified locations.  
It organizes projects into a structured Go-style package tree 📦.

---

## ✨ Features

- 🌐 Fetch repositories from **GitLab** and **GitHub**
- 📂 Organize repos into structured paths
- ⚡ Simple & fast command-line interface

---

## 🛠️ Installation

```bash
go install github.com/bmichalkiewicz/gloner@latest
```

---

## 📜 Commands

### 🔑 GitLab
```bash
# Example:
gloner gitlab -g terraform -t "ghyr$jrnjyuehd2" -u gitlab.custom.com

# Help:
gloner gitlab --help
```

### Clone
```bash
# Example:
gloner clone -u git@github.com:bmichalkiewicz/gloner.git

# Help:
gloner clone --help
```
## ⚙️ Configuration

You can provide a configuration file at:

```
~/.config/gloner/config.toml
```

### Example `config.toml`:

```toml
[gitlab]
token = ''
url = 'https://gitlab.com'
```
---

## 📌 Roadmap

- [ ] Support for **Bitbucket**
- [ ] Interactive mode for selecting repositories
- [ ] Config file support (more options)

---

## 🤝 Contributing

Pull requests are welcome!  
For major changes, please open an issue first to discuss what you’d like to change.  

---

## 📄 License

[MIT](LICENSE)
