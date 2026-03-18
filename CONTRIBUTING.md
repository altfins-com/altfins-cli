# Contributing

This document is for contributors working on the source code of `af`.

## Local Development

Prerequisites:

- Go
- Homebrew only if you want to test packaging locally

Install dependencies:

```bash
go mod tidy
```

Run the test suite:

```bash
go test ./...
```

Build the project:

```bash
go build ./...
```

Useful commands:

```bash
go run . auth status -o json
ALTFINS_API_KEY=demo-key go run . markets search --symbols BTC,ETH --dry-run -o json
go run . commands -o json
```

## Project Layout

- `cmd/` contains the public command surface
- `internal/altfins/` contains the typed API client
- `internal/app/` contains shared output and runtime helpers
- `internal/tui/` contains Bubble Tea TUI models and views
- `openapi/altfins-openapi.json` is the vendored API contract snapshot

## Release Notes

Public install docs are package-manager-first. If you change release channels, artifact names, or package manager configuration, update both:

- `README.md`
- `docs/releasing.md`

README install commands are a public contract and should only document channels that are actually published.

For Windows package-manager publication details, see [docs/winget.md](docs/winget.md).
