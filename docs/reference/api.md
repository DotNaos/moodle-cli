# API endpoints

Use this page when you want the exact HTTP endpoints exposed by `moodle serve`.

## Base URL

The default server address is `http://127.0.0.1:8080`.

## Endpoints

- `GET /healthz`
  Returns `{"status":"ok"}` when the server can use a valid Moodle session.
- `GET /api/courses`
  Returns your enrolled courses as JSON.
- `GET /api/courses/{courseID}/resources`
  Returns files and resources for one course as JSON.

## Quick check

```sh
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/api/courses
curl http://127.0.0.1:8080/api/courses/18236/resources
```

## Error shape

Errors are returned as JSON:

```json
{"error":"..."}
```
