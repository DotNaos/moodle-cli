# moodle-cli

CLI for FHGR Moodle with caching, course exports, and file downloads.

## Quickstart

### Schnell installieren ohne Go
Direkt aus den Releases:
- macOS: `.dmg`
- Windows: `.exe` installer
- Linux: `.tar.gz`

macOS / Linux:
```sh
curl -fsSL https://raw.githubusercontent.com/DotNaos/moodle-cli/main/scripts/install.sh | bash
```

Windows PowerShell:
```powershell
irm https://raw.githubusercontent.com/DotNaos/moodle-cli/main/scripts/install.ps1 | iex
```

Optional kannst du eine feste Version erzwingen:
```sh
VERSION=v1.2.3 curl -fsSL https://raw.githubusercontent.com/DotNaos/moodle-cli/main/scripts/install.sh | bash
```

### Mit Go installieren
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

### Erste Einrichtung
1. Configure credentials:
```sh
moodle config set \
  --school <school-id> \
  --username <username> \
  --password <password> \
  --calendar-url <ics-url>
```
Note: `--calendar-url` is optional (only needed for timetable).
2. Login (re-run when session expires):
```sh
moodle login
```
On first login, the CLI automatically installs the required Playwright driver and Chromium runtime.
3. Check the installed version:
```sh
moodle version
```
4. List courses:
```sh
moodle list courses --json
```
5. List files in a course:
```sh
moodle list files <course-id|name|current|0> --json
```
6. Open a course or resource in your browser:
```sh
moodle open course <course-id|name|current|0>
moodle open current current
moodle open resource <course-id|name|current|0> <resource-id|name|current|0>
```
Note: `moodle` is available on PATH in this workspace; avoid sourcing `~/.zshrc` from non-interactive shell commands because it loads interactive prompt setup.

### Updates
- `moodle update --check` checks whether a newer stable release is available.
- `moodle update` downloads and installs the latest stable release automatically.
- The CLI also checks roughly once per day in interactive terminals and prints a short hint if a newer release exists.

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
- CLI state: `~/.moodle-cli/state.json`
- Output: `~/Downloads/moodle/`

## Commands
- `moodle version`
- `moodle update --check`
- `moodle update`
- `moodle login`
- `moodle list courses --json`
- `moodle list files <course-id|name|current|0> --json`
- `moodle list timetable --json`
- `moodle list current current --json`
- `moodle open course <course-id|name|current|0>`
- `moodle open current current`
- `moodle open resource <course-id|name|current|0> <resource-id|name|current|0>`
- `moodle download file <course-id|name|current|0> <resource-id|name|current|0> --output-dir <path>`
- `moodle download file <course-id|name|current|0> --all --output-dir <path>`
- `moodle export course <course-id|name|current|0> --output-dir <path>`
- `moodle print current current`
- `moodle print course-page <course-id|name|current|0>`
- `moodle print course <course-id|name|current|0> <resource-id|name|current|0>`

By default, scraped course and resource names are cleaned up for easier matching and output. Use `--unsanitized` to preserve the raw Moodle names.

For best PDF OCR in `moodle print`, install `tesseract` and `pdftoppm` (Poppler).

## Skill (moodle-cli)
- Path: `skills/moodle-cli`
- Install via skills.sh: point it at `./skills/moodle-cli`

## Status
Scaffold in progress.
