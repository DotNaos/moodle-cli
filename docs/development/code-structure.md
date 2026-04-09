# Code structure

Use this page when you need a quick file map before editing.

## Top level

- `cmd/`
  Program entrypoint.
- `internal/`
  Application code.
- `scripts/`
  Install helpers.
- `packaging/`
  macOS and Windows packaging helpers.
- `assets/`
  Project images.

## `internal/`

- `internal/cli/`
  Cobra commands, TUI, browser actions, API startup, auth bootstrap, and command tests.
- `internal/api/`
  HTTP router and API tests.
- `internal/moodle/`
  Moodle client, login flow, sessions, scraping, files, timetable, downloads, and tests.
- `internal/config/`
  Default paths and config loading.
- `internal/update/`
  Release lookup and self-update logic.
- `internal/version/`
  Version metadata.
