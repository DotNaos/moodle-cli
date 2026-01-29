# Moodle CLI reference

Repo: `/Users/oli/projects/active/moodle-cli`

## Goals / planned commands
- `moodle login`
- `moodle courses --json`
- `moodle files <course-id> --json`
- `moodle timetable --json` (lectures only; no exam deadlines)
- `moodle download course <id> --zip|--files`
- `moodle export course <id> --format=folder|zip`

## Data locations (defaults)
- Config: `~/.moodle-cli/config.json`
- Session cookies: `~/.moodle-cli/session.json`
- SQLite cache: `~/.moodle-cli/cache.db`
- File cache: `~/.moodle-cli/files/`
- Export: `~/Downloads/moodle/`

## Notes
- Project status: scaffold in progress (see README).
- Prefer JSON outputs for parsing.
