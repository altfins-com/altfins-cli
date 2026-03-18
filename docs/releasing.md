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

Documented install commands:

```bash
brew install altfins-com/tap/af
```

or:

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

## Deferred Channels

Scoop and Winget are intentionally not part of the current public README until they are actually published and verified.

If either channel is added later:

1. add or restore the GoReleaser config
2. verify the install flow from a clean environment
3. update README and this document in the same change
