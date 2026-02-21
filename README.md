# moodle-cli

CLI for FHGR Moodle with caching, course exports, and file downloads.

## Quickstart

### Step-by-step
1. Install Go: https://go.dev/doc/install
2. Clone the repo:
```sh
git clone https://github.com/DotNaos/moodle-cli.git
cd moodle-cli
```
3. Ensure your Go bin is on PATH:
```sh
export PATH="$PATH:$HOME/go/bin"
```
4. Build/install the CLI:
```sh
go install ./cmd/moodle
```
5. Install the skill:
```sh
npx skills add DotNaos/moodle-cli
```
6. Configure credentials:
```sh
moodle config set \
  --school <school-id> \
  --username <username> \
  --password <password> \
  --calendar-url <ics-url>
```
Note: `--calendar-url` is optional (only needed for timetable).
7. Login (re-run when session expires):
```sh
moodle login
```
On first login, the CLI automatically installs the required Playwright driver and Chromium runtime.
8. List courses:
```sh
moodle list courses --json
```
9. List files in a course:
```sh
moodle list files <course-id|name> --json
```

### Zsh completion
In your `.zshrc`:
```sh
autoload -Uz compinit && compinit
source <(moodle completion zsh)
```

## Goals
- Login via Playwright with username/password
- List courses, files, timetable events
- Cache Moodle tree in SQLite
- Cache downloads and avoid re‑downloading
- Export full course (zip or file tree)

## Data locations (defaults)
- Config: `~/.moodle-cli/config.json`
- Session cookies: `~/.moodle-cli/session.json`
- SQLite cache: `~/.moodle-cli/cache.db`
- File cache: `~/.moodle-cli/files/`
- Output: `~/Downloads/moodle/`

## Commands
- `moodle login`
- `moodle list courses --json`
- `moodle list files <course-id|name> --json`
- `moodle list timetable --json`
- `moodle download file <course-id|name> <resource-id|name> --output-dir <path>`
- `moodle download file <course-id|name> --all --output-dir <path>`
- `moodle export course <course-id|name> --output-dir <path>`
- `moodle print course <course-id|name> <resource-id|name>`

## Skill (moodle-cli)
- Path: `skills/moodle-cli`
- Install via skills.sh: point it at `./skills/moodle-cli`

## Status
Scaffold in progress.
