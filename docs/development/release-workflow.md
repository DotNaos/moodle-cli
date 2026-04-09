# Release workflow

Use this page when you want to know how stable releases and container images are produced.

## Stable release path

1. Push the finished change to `main`.
2. The `Release From Main` workflow prepares the release.
3. The `Release` workflow publishes the GitHub release and container image.

## Release outputs

- Stable version tags such as `v0.1.11`
- `ghcr.io/dotnaos/moodle-cli:latest`
- Platform release files for macOS, Windows, and Linux

## Unstable channel

Pull requests can produce unstable builds for testing before merge.

## Local checks before release

```sh
go test ./...
docker build -t ghcr.io/dotnaos/moodle-cli .
```
