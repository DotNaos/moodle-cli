# moodle-cli

CLI for FHGR Moodle with caching, course exports, and file downloads.

## Goals
- Login via Playwright (SSO‑friendly)
- List courses, files, timetable events
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
- `moodle timetable --json`
- `moodle download course <id> --zip|--files`
- `moodle export course <id> --format=folder|zip`

## Skill (moodle-cli)
This repo bundles a Clawdbot skill at:
- `skills/moodle-cli`

**Install via skills.sh**
- `skills.sh` expects the structure `root/skills/<skill>`. Point it at `./skills/moodle-cli`.

## Status
Scaffold in progress.
