# AGENTS.md

Guidance for coding agents working on wallpapers.

## Project Overview

Wallpaper serving web application and image uploader written in Go (`github.com/icco/wallpapers`).

## Commands (Taskfile)

Run via `task <name>`:
- `task build` — Build all binaries
- `task test` — Run all unit tests (`go test -v ./...`)
- `task lint` — Run `golangci-lint run`
- `task fmt` / `task vet` — Format and vet Go code
- `task check` — Run fmt, vet, lint, and test together
- `task server` — Run web server locally (`go run ./cmd/server`)
- `task uploader` — Run image uploader CLI (`go run ./cmd/uploader`)

## Architecture & Layout

- `cmd/server/` — HTTP web service serving wallpapers.
- `cmd/uploader/` — CLI tool for processing and uploading wallpapers to storage.
- `lib/` — Image metadata, storage handling, and API routes.

## Conventions

- Follow icco Go conventions (`github.com/icco/gutil` for logging and utilities).
- PR titles and commits must follow Conventional Commits with lowercase subjects.
- Ensure `task check` passes before submitting PRs.
