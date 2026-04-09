# Default paths and environment variables

Use this page when you want the exact files and environment variables used by the CLI.

## Default paths

- Config: `~/.moodle-cli/config.json`
- Session cookies: `~/.moodle-cli/session.json`
- SQLite cache: `~/.moodle-cli/cache.db`
- File cache: `~/.moodle-cli/files/`
- CLI state: `~/.moodle-cli/state.json`
- Output: `~/Downloads/moodle/`

## Environment variables

- `MOODLE_CLI_HOME`
  Changes the base directory for config, session, cache, and state files.
- `MOODLE_CLI_EXPORT_DIR`
  Changes the default export directory.
- `MOODLE_USERNAME`
  Provides the username for automatic login.
- `MOODLE_PASSWORD`
  Provides the password for automatic login.
- `OS_STUDY_USERNAME`
  Alternative username variable used by the same login flow.
- `OS_STUDY_PASSWORD`
  Alternative password variable used by the same login flow.
