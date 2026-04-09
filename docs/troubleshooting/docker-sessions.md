# Docker session and credential problems

Use this page when Docker login and API runs do not behave the way you expect.

## I logged in once, but the next `docker run` cannot see the session

Separate `docker run` commands do not share their filesystem by default.

Use the same mounted folder in both runs:

```sh
docker run --rm \
  -v ${HOME}/.moodle-cli:/data \
  -e MOODLE_CLI_HOME=/data \
  ghcr.io/dotnaos/moodle-cli:latest login \
  --username "$MOODLE_USERNAME" \
  --password "$MOODLE_PASSWORD"

docker run --rm -p 8080:8080 \
  -v ${HOME}/.moodle-cli:/data \
  -e MOODLE_CLI_HOME=/data \
  ghcr.io/dotnaos/moodle-cli:latest serve --addr :8080
```

## I do not want to save a session at all

Start `serve` with a fresh login:

```sh
docker run --rm -p 8080:8080 \
  ghcr.io/dotnaos/moodle-cli:latest serve --addr :8080 \
  --username "$MOODLE_USERNAME" \
  --password "$MOODLE_PASSWORD"
```

## I prefer environment variables over flags

Use this version:

```sh
docker run --rm -p 8080:8080 \
  -e MOODLE_USERNAME="$MOODLE_USERNAME" \
  -e MOODLE_PASSWORD="$MOODLE_PASSWORD" \
  ghcr.io/dotnaos/moodle-cli:latest serve --addr :8080
```

## I am using Docker Compose

Use the compose guide here: [Run the API with Docker Compose](../guides/run-api-with-docker-compose.md).
