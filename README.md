# gloner

![logo](assets/logo.png)

`gloner` is a simple tool for fetching and cloning repositories into user-specified locations.
It uses a Go package tree to organize projects into structured paths.

## Features

- Fetch repositories from GitLab and Github.

## Commands

```bash
# example: gloner gitlab -g terraform -t "ghyr$jrnjyuehd2" -u gitlab.custom.com
gloner gitlab --help

# example: gloner clone -u git@github.com:bmichalkiewicz/brr.git
gloner clone --help
```
