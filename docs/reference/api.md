# API endpoints

Use this page when you want the exact HTTP endpoints exposed by `moodle serve`.

## Base URL

The default server address is `http://127.0.0.1:8080`.

## Built-in API reference

Open the live reference in your browser:

- `http://127.0.0.1:8080/docs`
- `http://127.0.0.1:8080/scalar`

The raw OpenAPI document is available at:

- `http://127.0.0.1:8080/openapi.json`

## Endpoints

- `GET /healthz`
  Returns `{"status":"ok"}` when the server can use a valid Moodle session.
- `GET /api/courses`
  Returns your enrolled courses as JSON.
- `GET /api/courses/{courseID}/resources`
  Returns files and resources for one course as JSON.
- `POST /api/cli/...`
  Runs the matching non-interactive CLI command through the API.

## CLI command endpoints

Non-interactive commands are exposed under `/api/cli/...` by default.
Commands are only skipped when they are explicitly marked as CLI-only, for example shell completion generation and `serve`.

Examples:

- `POST /api/cli/version`
- `POST /api/cli/list/courses`
- `POST /api/cli/config/show`
- `POST /api/cli/open/course`

Send the remaining CLI arguments and flags in JSON:

```json
{"arguments":["current","--open"]}
```

The command path itself is already part of the URL, so only send the extra arguments after it.

## Quick check

```sh
open http://127.0.0.1:8080/docs
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/api/courses
curl http://127.0.0.1:8080/api/courses/18236/resources
curl -X POST http://127.0.0.1:8080/api/cli/version
curl -X POST http://127.0.0.1:8080/api/cli/list/courses
```

## Error shape

Errors are returned as JSON:

```json
{"error":"..."}
```
