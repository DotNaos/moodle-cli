# Run the API with Docker and a fresh login

Use this page when you want one container that logs in, serves the API, and then forgets the session when it stops.

## Default path

1. Start the API and pass the login at startup.

```sh
docker run --rm -p 8080:8080 \
  ghcr.io/dotnaos/moodle-services:latest serve --addr :8080 \
  --username "$MOODLE_USERNAME" \
  --password "$MOODLE_PASSWORD"
```

2. Check that it is up.

```sh
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/api/courses
```

3. Stop the container when you are done.

The session lives only inside that container run.

## Environment variable version

Use this when you prefer environment variables over flags:

```sh
docker run --rm -p 8080:8080 \
  -e MOODLE_USERNAME="$MOODLE_USERNAME" \
  -e MOODLE_PASSWORD="$MOODLE_PASSWORD" \
  ghcr.io/dotnaos/moodle-services:latest serve --addr :8080
```

If you want the session to survive across separate runs, continue in [Run the API with Docker and a saved session](run-api-with-docker-saved-session.md).
