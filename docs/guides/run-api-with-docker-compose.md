# Run the API with Docker Compose

Use this page when you want the repo's `docker-compose.yml` to start the API.

## Steps

1. Export your Moodle login.

```sh
export MOODLE_USERNAME="<username>"
export MOODLE_PASSWORD="<password>"
```

2. Start the service.

```sh
docker compose up
```

3. Check that it is up.

```sh
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/api/courses
```

4. Stop it with `Ctrl+C` when you are done.
