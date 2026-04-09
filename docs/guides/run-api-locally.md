# Run the API locally

Use this page when you already logged in and want a local JSON API.

## Steps

1. Start the server.

```sh
moodle serve --addr :8080
```

2. Check that it is up.

```sh
curl http://127.0.0.1:8080/healthz
```

`/healthz` should return `{"status":"ok"}`.

3. Fetch your courses.

```sh
curl http://127.0.0.1:8080/api/courses
```

4. Keep the terminal open while you use the API.

## Fresh login on server start

Use this when you want a throwaway login and do not want to reuse a saved session:

```sh
moodle serve --addr :8080 \
  --username "<username>" \
  --password "<password>"
```

If you want the same flow in Docker, continue in [Run the API with Docker and a fresh login](run-api-with-docker-throwaway.md).
