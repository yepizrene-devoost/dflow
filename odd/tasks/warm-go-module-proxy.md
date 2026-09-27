# Warm Go module proxy after release

Issue: #52
Branch: `feature/warm-go-module-proxy`

## Objective

Make each locally published release visible to the public Go module proxy and document how to retry and verify the post-publication step.

## Scope and constraints

- The repository publishes locally with `make release`, not a GitHub Actions release workflow.
- Provide an independently runnable Makefile target that the release recipe calls only after GoReleaser succeeds.
- Fetch the exact tagged module version from `proxy.golang.org`, not a local module cache or a direct VCS fallback. Preserve the meaning of a post-publish failure: publication may already have succeeded.
- Update `RELEASING.md`; do not publish a release or modify CI for this issue.
- Delivery strategy: ask-on-risk; forecast under 400 authored changed lines, one work unit. First review boundary: branch point `15b71efb84be2d4e9e2137636360e71b468b0a4b`.
- Route: parent owns tracking and state; scoped `gentle-ai-worker` writes the Makefile and guide (multi-file write trigger). Verification uses the worker's foreground commands; parent checks the result and follows native assessment/review routing.
- TDD mode: strict, from `~/.gentle-ai/state.json` (`strict_tdd: true`). Repository runner: `make test` (`go test ./...`). Observe RED for the missing isolated warm-up target, GREEN with an offline mocked positive path, then run the applicable suite. Do not claim a Go test guards Makefile recipe behavior.

## Tasks

- [x] W1: Inspect the publication path, resolve an independent post-release proxy warm-up design, and create the work branch. Evidence: Makefile release recipe, RELEASING.md, read-only exploration; `dflow start feat warm-go-module-proxy --no-push` completed on a clean branch.
- [x] W2: Implement a standalone warm-up Makefile target, reuse it after successful GoReleaser publication, and document manual retry and verification. RED: `make -n warm-module-proxy VERSION=v0.4.1` failed before the target existed. GREEN: mocked curl success captured the exact `.info` URL and expected flags; mocked curl failure exited nonzero; `VERSION=dev` failed before curl. `make -n warm-module-proxy VERSION=v0.4.1`, `make -n release VERSION=v0.4.1`, `make test` and `git diff --check -- Makefile RELEASING.md` passed; no live request or release was made. Native risk assessment/review status: pending.
- [x] W3: Read back the completed candidate and close pre-commit bookkeeping. Evidence: parent inspected the Makefile and release-guide diff; parent spot-check `make -n warm-module-proxy VERSION=v0.4.1` displayed the exact `.info` request; `git diff --check` passed. Freeze declaration: expect native review of the complete candidate (Makefile, RELEASING.md, this closed checklist); no review outcome is claimed here. Commit/staging and external delivery remain separate human decisions.
- [x] W4: Record work-unit commit identity in a short follow-up `docs(odd)` commit before the committed-range review. Work unit: `2ad7711f4e75b95e690117cf1472394be2f45f08` (`chore(release): warm public go module proxy after publishing`). This is the sole ODD commit-identity exception; no review result is claimed in the tree.

## Acceptance

- `make warm-module-proxy` addresses the tagged public module independently; an untagged/default `dev` invocation fails before any fetch.
- `make release` invokes the same target after and only after successful publishing. A failed warm-up reports a post-publish retry path.
- The release guide shows retry and public proxy/pkg.go.dev verification and acknowledges indexing delay.
- No tag, release, push, merge, or issue closure is performed as part of implementation.

## Progress

W1–W4 completed. The complete ODD checklist and work-unit identity are recorded before the committed-range freeze; the earlier working-tree lineage remains unapproved. No review outcome or delivery is claimed.
