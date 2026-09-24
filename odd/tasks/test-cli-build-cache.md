# Feature: test-cli-build-cache (issue #33)

Stop rebuilding the CLI binary cold on every CLI-process test in `cmd/tests`.

Root cause: `setUpCLIEnv` sets `XDG_CACHE_HOME` to a fresh `t.TempDir()`, and the
spawned `go build` inherits it. On Linux `os.UserCacheDir()` honors
`$XDG_CACHE_HOME`, so every build gets an empty default `GOCACHE` — a full cold
compile (~11s x ~25 tests ≈ 292s of CI).

Design (from the issue):
- Capture the real `GOCACHE` once at package level, before any `t.Setenv` runs,
  and pass it explicitly in the `cmd.Env` of every spawned `go build`.
- Memoize one build per distinct version marker ("dev", "v0.1.0", "v9.9.9") in a
  shared temp dir; each test copies the binary into its own `t.TempDir()`
  (copy required: `TestUpdateCLIReplacesBinary` mutates the binary it runs).
- Keep `XDG_CACHE_HOME` isolation untouched. Test-only change.

## Tasks

- [x] task-1: Shared build helper with anchored GOCACHE and per-marker memoization (`cmd/tests`)
- [x] task-2: Rewire existing build call sites (status/start/update/icon_chrome) onto the shared helper
- [x] task-3: Verify: go vet + full `go test ./cmd/tests/...` green, build count drops to ~3
- [ ] task-4: Work-unit commit on feature branch; record commit identity

## Evidence

- task-1: `cmd/tests/clibuild_test.go` (new): `realGOCACHE` captured at package init (before any `t.Setenv`); per-ldflags memoization (mutex + per-key `sync.Once`, errors and output cached); `runSharedBuild` anchors `GOCACHE` in the child env only when captured, never touches `XDG_CACHE_HOME`; one `os.MkdirTemp(dir, "config-")` subdir per configuration so no build clobbers another memoized artifact; `sharedDflowCLI(t, ldflags)` hands out a fresh 0o755 copy into the caller's `t.TempDir()` (isolation: `TestUpdateCLIReplacesBinary` mutates the binary it runs); `TestMain` removes the shared dir after `m.Run()`.
- task-2: `status_cli_test.go`/`update_cli_test.go` helpers now delegate to `sharedDflowCLI` (marker semantics preserved); `icon_chrome_test.go` duplicate wrapper deleted (2 call sites → `buildDflowCLI`); `start_cli_test.go` inline build replaced with `buildDflowCLI(t)`. Net: 12 insertions, 62 deletions across the 4 rewired files; exactly one `go build` spawn site remains, keyed on ldflags ("", "-X main.version=v0.1.0", "-X main.version=v9.9.9") = at most 3 builds per run.
- task-3: `go vet ./cmd/...` clean; `go test -count=1 ./cmd/tests/...` → ok, 27.8s; `gofmt -l cmd/tests/` clean (writer run) and re-verified independently (vet + count=1 suite, ok 27.8s). CI-side Linux gain (~292s → ~60-80s) is by construction, not observable on this warm macOS host.
- task-4: (pending — commit identity recorded in a follow-up `docs(odd)` commit)
