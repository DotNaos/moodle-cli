# moodle-cli

CLI for FHGR Moodle with caching, course exports, and file downloads.

## Goals
- Login via Playwright (SSO‑friendly)
- List courses, files, deadlines
- Cache Moodle tree in SQLite
- Cache downloads and avoid re‑downloading
- Export full course (zip or file tree)

## Data locations (defaults)
- Config: `~/.moodle-cli/config.json`
- Session cookies: `~/.moodle-cli/session.json`
- SQLite cache: `~/.moodle-cli/cache.db`
- File cache: `~/.moodle-cli/files/`
- Export: `~/Downloads/moodle/`

## Planned Commands
- `moodle login`
- `moodle courses --json`
- `moodle files <course-id> --json`
- `moodle deadlines --json`
- `moodle download course <id> --zip|--files`
- `moodle export course <id> --format=folder|zip`

## Status
Scaffold in progress.
