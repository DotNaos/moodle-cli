# Run the API with Docker and a saved session

Use this page when you want separate `docker run` commands to share the same session.

## Steps

1. Log in and save the session into a host folder.

```sh
docker run --rm \
  -v ${HOME}/.moodle:/data \
  -e MOODLE_HOME=/data \
  ghcr.io/dotnaos/moodle-cli:latest login \
  --username "$MOODLE_USERNAME" \
  --password "$MOODLE_PASSWORD"
```

2. Start the API with the same mounted folder.

```sh
docker run --rm -p 8080:8080 \
  -v ${HOME}/.moodle:/data \
  -e MOODLE_HOME=/data \
  ghcr.io/dotnaos/moodle-cli:latest serve --addr :8080
```

3. Check that it is up.

```sh
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/api/courses
```

4. Reuse the same `${HOME}/.moodle` folder for later runs.

If you do not want to keep a session, continue in [Run the API with Docker and a fresh login](run-api-with-docker-throwaway.md).
