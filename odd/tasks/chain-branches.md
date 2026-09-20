# chain-branches — feature tracking

Branch: `feature/implement-chain-branches` (base: develop)
Issues: #8 (MVP: start --from + non-interactive), #9 (rebase follow-up)

## Goal

Issue #8: add `--from <branch>` to `dflow start` so a work branch can be based
on another work branch (chained/stacked), plus `--push`/`--no-push` for
non-interactive use by agents.

## Tasks (issue #8)

1. [x] Implement `--from`, `--push`, `--no-push` in `cmd/commands/start.go`
2. [x] Write tests (`cmd/tests/start_cmd_test.go`)
3. [x] `go test ./...` + `go build`
4. [x] Review diff (left uncommitted per request)

## Evidence

- `cmd/commands/start.go`: DisableFlagParsing removed; `--from`/`--push`/`--no-push` added.
- `cmd/tests/start_cmd_test.go`: 4 tests added, all passing.
- Checks: gofmt clean, `go build ./...` OK, `go test ./...` green.
- Commits deferred: changes left uncommitted for review.
