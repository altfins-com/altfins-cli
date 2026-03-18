# Releasing and Distribution

`af` is documented as a package-manager-first CLI.

## Public Install Contract

The public onboarding path is:

1. Install via Homebrew
2. Run `af auth set`
3. Start with `af quota all`, `af markets search`, or `af tui markets`

The README should only document release channels that are live and verified.

## Current Channels

### Homebrew

GoReleaser publishes a formula to the tap repository configured in `.goreleaser.yml`:

- Tap repo: `altfins-com/homebrew-tap`
- Tap alias: `altfins-com/tap`
- Formula: `af`

Primary documented install command:

```bash
brew install altfins-com/tap/af
```

Fallback explicit tap flow:

```bash
brew tap altfins-com/tap
brew install af
```

### GitHub Releases

GitHub Releases are the fallback install path when Homebrew is unavailable or users want a direct binary download.

README should describe this generically:

- download the archive for the target platform
- extract it
- move `af` into `PATH`

Avoid hardcoding archive filenames in README unless artifact naming is intentionally frozen.

## Prepared But Not Yet Public

### Winget

The repo now includes a `winget` release configuration in `.goreleaser.yml`, but public README onboarding should still wait until the first package is merged into `microsoft/winget-pkgs`.

Target public install command after publication:

```powershell
winget install --id altfins.af -e --source winget
```

The `winget` path is intentionally opt-in on release:

- default releases keep `winget` upload disabled
- set `WINGET_PUBLISH=1` to generate manifests, push to a fork of `winget-pkgs`, and open a draft PR
- optionally set `WINGET_OWNER` if the fork owner is not the authenticated GitHub user

See [docs/winget.md](winget.md) for the exact flow.

## Deferred Channels

Scoop and Winget are intentionally not part of the current public README until they are actually published and verified.

If either channel is added later:

1. publish and verify the channel
2. verify the install flow from a clean environment
3. update README and this document in the same change
