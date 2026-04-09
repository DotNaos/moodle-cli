# Internals

Use this page when you want the shortest map of how the project works inside.

## Request flow

1. `cmd/moodle/main.go` starts the CLI.
2. `internal/cli` parses commands, loads paths, and decides which flow to run.
3. `internal/config` resolves config, session, cache, state, and output paths.
4. `internal/moodle` owns login, session validation, scraping, downloads, and course data.
5. `internal/api` exposes the HTTP router used by `moodle serve`.

## Packaging flow

- `Dockerfile` builds the Linux container image.
- `scripts/` contains install scripts.
- `packaging/` contains macOS and Windows packaging helpers.
