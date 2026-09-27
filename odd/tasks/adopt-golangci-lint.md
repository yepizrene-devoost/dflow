# Adopt golangci-lint gate

Issue: #54
Branch: `feature/adopt-golangci-lint`

## Objective

Replace the lost Go Report Card signal with a separate, reproducible golangci-lint CI workflow and README badge, green without baselines or per-finding suppressions.

## Scope and decisions

Follow the approved issue's exact v2 configuration (standard linters plus bodyclose, gosec, noctx; gofmt and goimports), dedicated pinned workflow, badge, and 27 source fixes. Keep the existing build/test workflow and historical ODD snapshots unchanged. No timeout/cancellation plumbing or extra linters. One cohesive work unit: the gate and fixes must land together so CI starts green. Estimated authored diff ~160 lines plus this checklist, under the ~400-line delivery threshold. Route: delegated direct writer, because changes span multiple non-trivial files; parent owns branch, tracking, and delivery. TDD mode: not explicitly configured for this ODD task; runner `go test ./...`, supplemented by lint, build and vet. Baseline for lint is expected red until the 27 fixes are applied; record actual result, not a presumed one.

## Checklist

- [x] L1 — Compare issue #54 with the current tree and create a work branch from `develop`. Evidence: `feature/adopt-golangci-lint` at branch point `fc5689e02efaedc3db63630296f2111fb1939da6`; installed golangci-lint 2.11.4.
- [x] L2 — Add v2 lint config, standalone pinned CI workflow and README badge. Evidence: `.golangci.yml`, `.github/workflows/golangci-lint.yml`, README badge; build/test workflow unchanged.
- [x] L3 — Resolve all baseline findings and update two affected CLI assertions. Evidence: worker reports `golangci-lint run ./...` → `0 issues`; parent repeated → `0 issues`; no baseline or `nolint` added.
- [x] L4 — Run `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test ./...`, and diff hygiene. Evidence: worker and independent verifier each report lint `0 issues`, empty gofmt and diff check, successful build/vet and all Go package tests; parent repeated lint (`0 issues`) and diff check after adjusting the G115 rationale. GitHub CI and alternate Go toolchains not run.
- [x] L5 — Record work-unit identity and review declaration before any review freeze. Evidence: work-unit commit `6eb1585ffcb066dae9b1779ca25d38ca7912b1d9` (`chore(lint): add dedicated golangci quality gate`); this checklist closure belongs to the commit-identity exception follow-up. Native review is expected for the committed range from `fc5689e02efaedc3db63630296f2111fb1939da6` after this follow-up is committed; no verdict or approval is declared.

## Progress

Branch started clean; L2 and L3 implemented together by scoped writer. Independent verifier confirmed all six checks; noted a G115 rationale mismatch, corrected before parent repeated lint and diff hygiene. Native ASSESS was unassessable because untracked candidate paths require declaration, so its returned plan's independent verifier was used. L4 is complete. User authorized the work-unit commit to initiate RDD. Work-unit identity `6eb1585ffcb066dae9b1779ca25d38ca7912b1d9` is recorded in this checklist exception follow-up before the freeze. Review declaration: submit the final committed range against branch point `fc5689e02efaedc3db63630296f2111fb1939da6` for native RDD review; approval is not presumed. Delivery strategy: ask-on-risk; one cohesive work unit below 400 authored diff lines. Push, merge and issue closure require separate decisions.
