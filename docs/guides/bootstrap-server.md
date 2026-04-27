# Bootstrap a server with one command

Goal: install the `moodle` command on a server without typing your Moodle password there.

Use two terminals:

- Mac terminal: create the install command.
- Server terminal: paste and run it.

Do not use `sudo` unless you want the install under `/root`.

## 1. Mac: check the account

```sh
moodle --json list courses
```

OK: this shows the courses for the account you want on the server.

Wrong account:

```sh
rm -f ~/.moodle/mobile-session.json
moodle login
moodle --json list courses
```

## 2. Mac: copy the install command

```sh
moodle bootstrap server --copy
```

OK: the command says the server install command was copied.

The clipboard now contains a secret. Paste it only into the server terminal.

If clipboard copy does not work:

```sh
moodle bootstrap server --print
```

Copy only the printed `curl ... --payload '...'` command.

## 3. Server: preflight

Run this as the server user who should own the install:

```sh
whoami
docker --version
docker info >/dev/null && echo "docker ok"
curl -fsSL https://raw.githubusercontent.com/DotNaos/moodle-cli/main/scripts/install-docker.sh >/dev/null && echo "github ok"
```

OK:

- `whoami` is the right server user.
- `docker ok` appears.
- `github ok` appears.

Stop if Docker or GitHub access fails. Fix that first.

## 4. Server: paste the copied command

Paste the real command from your Mac into the server terminal.

It looks like this, but with a long secret payload:

```sh
curl -fsSL https://raw.githubusercontent.com/DotNaos/moodle-cli/main/scripts/install-docker.sh | bash -s -- --payload '<secret-bootstrap-payload>'
```

OK: the installer finishes with:

```text
Moodle CLI is installed.
```

## 5. Server: open a new shell

Open a new server terminal, then run:

```sh
command -v moodle
moodle --json list courses
```

OK:

- `command -v moodle` prints a path under `~/.local/bin`.
- `moodle --json list courses` prints the course list.

If `moodle` is not found:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Then retry:

```sh
command -v moodle
moodle --json list courses
```

## 6. Done

Server state is here:

```sh
~/.moodle
```

Keep that folder private and persistent.

## If the secret command leaked

Revoke the old Moodle mobile session in Moodle if possible.

Then remove the server token:

```sh
rm -f ~/.moodle/mobile-session.json
```

Create a fresh command on your Mac:

```sh
moodle bootstrap server --copy
```

Run the fresh command on the server.
