# Moodle CLI

<div align="center">
<img src="assets/moodle_cli_icon.png" alt="moodle-cli icon" width="250" />
</div>

## Index

- [Moodle CLI](#moodle-cli)
  - [Index](#index)
  - [Abstract](#abstract)
  - [Getting started](#getting-started)
  - [Guides](#guides)
  - [Reference](#reference)
  - [Troubleshooting](#troubleshooting)
  - [Development](#development)

## Abstract

CLI and JSON API for FHGR Moodle.
The fastest way to use it is to start the API in Docker and log in at server start.
FHGR is the current default. Multi-school support is not active.

## Getting started

1. Export your Moodle login.

```sh
export MOODLE_USERNAME="<username>"
export MOODLE_PASSWORD="<password>"
```

2. Start the API in Docker.

```sh
docker run --rm -p 8080:8080 \
  -e MOODLE_USERNAME="$MOODLE_USERNAME" \
  -e MOODLE_PASSWORD="$MOODLE_PASSWORD" \
  ghcr.io/dotnaos/moodle-cli:latest serve --addr :8080
```

3. Check that it is up.

```sh
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/api/courses
```

`/healthz` should return `{"status":"ok"}`.

4. Keep that terminal open while you use the API.

5. If you want a saved Docker session later, continue in [Run the API with Docker and a saved session](docs/guides/run-api-with-docker-saved-session.md).

6. If you want the local CLI instead of Docker, continue in [Use the CLI locally](docs/guides/install.md).

## Guides

- [Docs index](docs/README.md)
- [Run the API with Docker and a fresh login](docs/guides/run-api-with-docker-throwaway.md)
- [Run the API with Docker and a saved session](docs/guides/run-api-with-docker-saved-session.md)
- [Run the API with Docker Compose](docs/guides/run-api-with-docker-compose.md)
- [Use the CLI locally](docs/guides/install.md)
- [Run the API locally](docs/guides/run-api-locally.md)
- [Update the CLI](docs/guides/update.md)

## Reference

- [CLI commands](docs/reference/cli.md)
- [API endpoints](docs/reference/api.md)
- [Default paths and environment variables](docs/reference/paths.md)

## Troubleshooting

- [Login and config problems](docs/troubleshooting/login-and-config.md)
- [Docker session and credential problems](docs/troubleshooting/docker-sessions.md)

## Development

- [Internals](docs/development/internals.md)
- [Architecture](docs/development/architecture.md)
- [Code structure](docs/development/code-structure.md)
- [Release workflow](docs/development/release-workflow.md)
