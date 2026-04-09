# CLI commands

Use this page when you want the exact command for a common task.

## Core commands

- `moodle` opens the interactive view.
- `moodle login` creates or refreshes the saved session.
- `moodle serve --addr :8080` starts the local JSON API.
- `moodle update` installs the latest stable release.

## List data

```sh
moodle list courses --json
moodle list files <course-id|name|current|0> --json
```

## Open in your browser

```sh
moodle open course <course-id|name|current|0>
moodle open current current
moodle open resource <course-id|name|current|0> <resource-id|name|current|0>
```

## Print course content

```sh
moodle print course-page <course-id|name|current|0>
```

## Download files

```sh
moodle download file <course-id|name|current|0> <resource-id|name|current|0> --output-dir <path>
moodle export course <course-id|name|current|0> --output-dir <path>
```

## Shell completion

```sh
autoload -Uz compinit && compinit
source <(moodle completion zsh)
```
