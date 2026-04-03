# Releasing dflow

This document describes the release flow currently used by this repository.

## Release Overview

`dflow` uses:

- `develop` as the main integration branch
- `main` as the release branch
- `release/*` branches to prepare release notes and final publication changes
- `GoReleaser` to build binaries and publish GitHub releases

The GitHub release body is taken from `CHANGELOG.md`.

## Release Checklist

### 1. Finish the feature work

- Merge all work branches into `develop`
- Ensure `develop` is clean and up to date
- Run:

```bash
go test ./...
```

### 2. Create a release branch

Create a release branch from `develop`:

```bash
dflow start release v0.2.0
```

Adjust the version string to the release you are preparing.

### 3. Prepare release notes

On the release branch:

- Update `CHANGELOG.md` so it contains the notes for the version being released
- Update `HISTORY.md` with the new version, tag, and date
- Keep release-only documentation changes isolated in this branch

Run tests again:

```bash
go test ./...
```

### 4. Finish the release branch

From the `release/*` branch, run:

```bash
dflow finish
```

With the current repository configuration:

- `develop` is an `auto` target
- `main` is a `manual` target

That means `dflow finish` will merge the release branch back into `develop`,
but you still need a PR from `release/*` into `main`.

### 5. Merge to main

- Open a PR from the release branch to `main`
- Merge it
- Check out `main`
- Pull the merged changes

At this point, `main` should contain the exact state you want to publish.

### 6. Create the release tag

Create the tag from the current `main` commit:

```bash
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```

If you need to fix release metadata after tagging but before publishing:

```bash
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0
```

Then recreate the tag on the corrected commit.

### 7. Publish the release

Push `main` if needed, then run:

```bash
make release
```

The release target:

- loads `GITHUB_TOKEN` from `.env`
- runs `goreleaser release --clean --release-notes=CHANGELOG.md`

GoReleaser is configured to:

- use config format `version: 2`
- replace the GitHub release body if the release already exists
- attach `LICENSE`, `README.md`, and `CHANGELOG.md` to the archives

## Notes

- Always tag from `main`, never from `develop` or `release/*`
- Keep the working tree clean before tagging or publishing
- `CHANGELOG.md` is the published release body
- `HISTORY.md` is the long-form project history
- If `.env` contains old or unused tokens, remove them and rotate any exposed credentials
