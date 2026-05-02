# Moodle Services reference

Use these commands when you need quick reminders or output paths.

## Handy commands
- `moodle login`
- `moodle list courses --json`
- `moodle list files <course-id> --json`
- `moodle list timetable --json` (lectures only; no exam deadlines)
- `moodle download file <course-id> --all --output-dir <path>`
- `moodle export course <course-id> --output-dir <path>`
- `moodle skill` (print the bundled skill text)
- `moodle skill --install` (install skill to codex/opencode/claude-code/gemini-cli)
- `moodle logs` (tail debug log; use `--error` for error log)

## Data locations (defaults)
- Config: `~/.moodle/config.json`
- Session cookies: `~/.moodle/session.json`
- SQLite cache: `~/.moodle/cache.db`
- File cache: `~/.moodle/files/`
- Export: `~/Downloads/moodle/`
- Debug log: `~/.moodle/cli.log`
- Error log: `~/.moodle/error.log`

## Notes
- Project status: scaffold in progress (see README).
- Prefer JSON outputs for parsing.
