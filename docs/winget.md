# Winget Packaging

`af` is published under the package ID `altfins.af` in `winget`.

## Install

Install from the public catalogue.

```powershell
winget install --id altfins.af -e --source winget
```


## Package Identity

- Package identifier: `altfins.af`
- Package name: `altFINS CLI`
- Publisher: `altFINS`
- Source repository: `altfins-com/altfins-cli`
- Target catalog: `microsoft/winget-pkgs`

## Artifact Mapping

The package uses the Windows archives already produced by GoReleaser:

- `af_<version>_windows_amd64.zip`
- `af_<version>_windows_arm64.zip`

Each archive currently contains `af.exe` at the root, which makes it suitable for a portable `winget` package based on ZIP artifacts.

## Release Flow

The main release pipeline keeps Homebrew live by default and leaves `winget` upload disabled unless you opt in.

To publish a release and also open a `winget-pkgs` PR:

1. Make sure you have a GitHub fork of `microsoft/winget-pkgs`.
2. Set `WINGET_OWNER` to the fork owner if it is not your authenticated GitHub user.
3. Export `WINGET_PUBLISH=1`.
4. Run GoReleaser as usual.

Example:

```bash
export WINGET_PUBLISH=1
export WINGET_OWNER=feci
GITHUB_TOKEN="$(gh auth token)" goreleaser release --clean
```

With that flag enabled, GoReleaser will:

- generate the `winget` manifests from the Windows release assets
- push them to `<WINGET_OWNER>/winget-pkgs`
- open a PR into `microsoft/winget-pkgs:master`

## First-Time Setup

Create your fork once:

```bash
gh repo fork microsoft/winget-pkgs --clone=false --remote=false
```

If you want the fork under an organization instead of your user account, create or fork that repository there first and then set `WINGET_OWNER` for releases.

## Verification

Verify that Microsoft accepted the version manifest and that both architecture hashes match the release archives. Catalogue availability follows Microsoft review.
