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

## Command Safety Metadata

`af` exposes a machine-readable safety contract through `af commands -o json` (see the README "Agent Safety Contract"). Every new runnable command MUST declare its safety metadata. `cmd/safety_test.go` fails if any runnable leaf command is missing it.

When adding a command, annotate it in its constructor:

- Networked read command → `markRemoteQuery(cmd, method, path)`.
- TUI command → `markInteractive(cmd, path)`.
- Local read command → `markLocalRead(cmd)`.
- Anything else (local/remote write, or custom confirmation/force) → `annotateSafety(cmd, safetyMeta{...})` with the correct `operationType`, mutation flags, `dryRunSupported`, and any `confirmationFlags` / `forceSupported`.

Keep behavior and metadata consistent:

- If a command mutates local state, it should respect `--dry-run`.
- If it advertises `confirmationRequired`, it must enforce a confirmation flag and return `*app.ConfirmationRequiredError` (exit code `5`) when that flag is missing in non-interactive mode.
- Use `--force` only where there is a genuine overwrite/override semantic.

## Release Notes

Public install docs are package-manager-first. If you change release channels, artifact names, or package manager configuration, update both:

- `README.md`
- `docs/releasing.md`

README install commands are a public contract and should only document channels that are actually published.

For Windows package-manager publication details, see [docs/winget.md](docs/winget.md).
