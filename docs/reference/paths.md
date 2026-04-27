# Default paths and environment variables

Use this page when you want the exact files and environment variables used by the CLI.

## Default paths

- Config: `~/.moodle/config.json`
- Session cookies: `~/.moodle/session.json`
- Mobile token session: `~/.moodle/mobile-session.json`
- SQLite cache: `~/.moodle/cache.db`
- File cache: `~/.moodle/files/`
- CLI state: `~/.moodle/state.json`
- Debug log: `~/.moodle/cli.log`
- Error log: `~/.moodle/error.log`
- Output: `~/Downloads/moodle/`

## Environment variables

- `MOODLE_HOME`
  Changes the base directory for config, session, cache, and state files.
- `MOODLE_CLI_HOME`
  Legacy fallback for older installs. Prefer `MOODLE_HOME` for new setups.
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

## Legacy migration

Older versions used `~/.moodle-cli`. New installs use `~/.moodle` so the CLI and future Moodle apps can share the same data folder.

To copy old data into the new location without deleting the old folder, run:

```sh
moodle config migrate-home
```
