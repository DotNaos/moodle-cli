# Use the CLI locally

Use this page when you want the local `moodle` command instead of Docker.

## Default path

1. Install `moodle`.

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/DotNaos/moodle-services/main/scripts/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/DotNaos/moodle-services/main/scripts/install.ps1 | iex
```

2. Open a new shell.

3. Save your login once.

```sh
moodle config set \
  --username "<username>" \
  --password "<password>"
```

4. Log in.

```sh
moodle login
```

You should see `session saved to ...`.

5. Check that it works.

```sh
moodle --json list courses
```

6. If you want the local API next, continue in [Run the API locally](run-api-locally.md).

## Packaged releases

Use this when you do not want the install script:

- macOS: download the `.dmg`
- Windows: download the `.exe`
- Linux: download the `.tar.gz`

All release files are attached to the GitHub releases page.

## Build from source

Use this when you want a local Go build:

```sh
git clone https://github.com/DotNaos/moodle-services.git
cd moodle-services
go install ./cmd/moodle
```

If your Go bin is not on `PATH`, add it first:

```sh
export PATH="$PATH:$HOME/go/bin"
```
