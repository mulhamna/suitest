# Release pipeline postmortem

## What broke

The initial `v1.0.0` release created a GitHub Release and tag, but did not publish binary assets or update downstream package repos.

## Root cause

The repo used this flow:

1. `release-please` ran on pushes to `main`
2. `release-please` created a release tag using `GITHUB_TOKEN`
3. a separate `GoReleaser` workflow listened for `push` on `v*` tags

That failed because tags created by a workflow using `GITHUB_TOKEN` do not reliably trigger downstream tag-push workflows.

## Symptoms

- GitHub Release existed
- release notes existed
- no GoReleaser run started
- no release assets were uploaded
- no Homebrew formula update landed
- no Scoop manifest update landed
- duplicate root-level package files appeared in downstream repos and had to be cleaned up

## Fixes applied

### In `suitest`

- switched release automation to use `GITHUB_TOKEN`
- moved GoReleaser execution into the same release workflow, gated on `release_created == true`
- kept standalone GoReleaser as manual fallback via `workflow_dispatch`
- added basic CI for pull requests and pushes to `main`
  - `go vet ./...`
  - `go test ./...`
  - `go build ./...`

### In downstream repos

- removed duplicate root-level `suitest.rb` from `homebrew-tap`
- removed duplicate root-level `suitest.json` from `scoop-bucket`
- kept canonical package files at:
  - `Formula/suitest.rb`
  - `bucket/suitest.json`

## Current known-good state

Verified with `v1.0.1`:

- CI passed
- release workflow passed
- release assets were uploaded
- Homebrew formula updated in `Formula/`
- Scoop manifest updated in `bucket/`
- no duplicate root-level package files remained

## Rules of thumb for future repos

1. Do not chain release-critical workflows through tag pushes created by `GITHUB_TOKEN`
2. Prefer one release workflow that creates the release and publishes artifacts in the same run
3. Add basic CI before relying on release automation
4. Keep Homebrew formulas in `Formula/`
5. Keep Scoop manifests in `bucket/`
6. Treat formatting-only cleanup as a separate PR if the repo already has style debt

## Recommended baseline for future Go repos

- PR CI:
  - `go vet ./...`
  - `go test ./...`
  - `go build ./...`
- release:
  - `release-please` on `main`
  - GoReleaser in the same workflow after `release_created == true`
- package distribution:
  - Homebrew tap via GoReleaser `brews`
  - Scoop bucket via GoReleaser `scoops`
