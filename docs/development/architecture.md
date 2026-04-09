# Architecture

Use this page when you want the high-level boundaries before changing code.

## Layers

- CLI layer
  Lives in `internal/cli`. It owns command flags, user flows, and API startup.
- Config layer
  Lives in `internal/config`. It owns where local files live.
- Moodle layer
  Lives in `internal/moodle`. It owns the Moodle-specific logic: login, sessions, scraping, files, and timetable data.
- API layer
  Lives in `internal/api`. It turns the Moodle client into a small JSON API.

## Dependency direction

- `cmd/moodle` depends on `internal/cli`.
- `internal/cli` depends on `internal/config`, `internal/moodle`, and `internal/api`.
- `internal/api` depends on a small client interface and on Moodle data types.
- `internal/moodle` does not depend on the CLI layer.

Keep new code inside the narrowest layer that matches its job.
