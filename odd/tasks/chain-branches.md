# chain-branches — feature tracking

Branch: `feature/implement-chain-branches` (base: develop)
Issues: #8 (MVP: start --from + non-interactive), #9 (rebase follow-up)
Status: #8 implementation, documentation and CLI coverage complete; integration pending; #9 pending

## Goal

Issue #8: add `--from <branch>` to `dflow start` so a work branch can be based
on another work branch (chained/stacked), plus `--push`/`--no-push` for
non-interactive use by agents.

## Tasks (issue #8)

1. [x] Implement `--from`, `--push`, `--no-push` in `cmd/commands/start.go`
2. [x] Write tests (`cmd/tests/start_cmd_test.go`)
3. [x] `go test ./...` + `go build`
4. [x] Review diff (left uncommitted per request)

## Closure follow-up (issue #8)

Authorized scope: README documentation and CLI-process regression coverage only.
Route: one delegated writer (multiple non-trivial files); no production behavior changes.
Estimated scope: under 250 added/deleted lines; delivery strategy: single bounded slice.
TDD configuration: not found in project tracking/configuration; this follow-up tests existing behavior, so passing baseline coverage is expected (no fabricated RED).
Checks: `go test -count=1 ./cmd/tests`, `go test -count=1 ./...`, `go build ./...`, `git diff --check`.
Commit/push/merge/issue closure: not authorized for this follow-up.

5. [x] Document `--from`, `--push`, and `--no-push` with README examples; explicitly require a push-choice flag for non-interactive use. Evidence: examples and prompt/no-TTY-detection wording inspected; `git diff --check` passed.
6. [x] Add real CLI-process tests with non-TTY stdin for explicit push choices and a remote-only parent; assert branch ancestry and remote effects. Evidence: `go test -count=1 ./cmd/tests` passed; binary built once, bounded CLI commands, parent refs/object absent before invocation. Initial single-branch clone fixture failed because its fetch refspec excluded the parent; standard clone before parent publication passes without production edits.
7. [x] Run checks and review the new bounded candidate. Independent verification passed: focused `TestStartCLI`, full test suite, build and diff check. RDD `review-c6e3935be010f3a8` approved and acknowledged (4/4 lenses; one non-blocking advisory `R3-push-choice-contract`, README.md:159-161). This completion entry is post-review bookkeeping; the approved candidate was `sha256:5c35b9560d77aa46b8c640d81c7066db82f55758381f6beffb7b7d66026ef965`.

## Evidence

- `cmd/commands/start.go`: DisableFlagParsing removed; `--from`/`--push`/`--no-push` added.
- `cmd/tests/start_cmd_test.go`: 6 tests added, all passing.
- Checks: gofmt clean, `go build ./...` OK, `go test ./...` green.
- Commit: 2427ec6 `feat(start): support chained branches and non-interactive push`
- Commit: 6831d26 `fix(gitutils): add checkout-only branch helper and reuse it in pull branch`
- Commit: f778cf2 `fix(start): use checkout helper and validate push flags before mutation`
- RDD: lineage review-66f23b55e8f5509d approved (authority burned); corrected `R3-checkout-remote-exists-before-fetch`
