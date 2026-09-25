# Feature: update digest legibility (issue #30)

`dflow update --check` renders the what's-new digest from the release body via
`selfupdate.SummarizeReleaseNotes` (`cmd/selfupdate/notes.go`). Against real
curated bodies (v0.3.0) the digest wraps mid-word and repeats the release's own
version heading right under the caller's `what's new in vX:` line.

Branch: `feature/update-digest-legibility` (base `develop`, per `.dflow.yaml`).

## Design decisions

- **Version heading drop rule**: a heading (any depth — `##` or deeper) whose
  text contains a semver token (`v?X.Y.Z`) is the release's own version heading
  and is dropped like the H1 title (formatting removal — it never contributes to
  the ellipsis). Section headings without a version token (`### Added`,
  `### Fixed`) keep the `▸ ` marker. Wider than the issue's `##` example on
  purpose: a `### Fixed in v0.4.1` heading is equally redundant, and the
  behavior is pinned by test.
- **Item clamp**: bullet content is clamped to `maxReleaseNotesItemChars` runes;
  a clipped item ends with `…`. Sentence-first cutting was considered and
  rejected: real curated bullets are single long sentences, so the budget does
  the work and the rule stays deterministic. The full release URL remains the
  closing line, so clipping loses no essential information.
- Headings are not clamped (they are short and are already bounded by the
  global line/char caps); only bullet content clamps. Indentation is preserved.

## Tasks

- [x] T1: pin the new contract in tests first (drop version heading, clamp
      bullets) in `cmd/selfupdate/notes_test.go` — RED observed in two stages
      (build failure on `maxReleaseNotesItemChars`, then 13 behavioral failures
      including the old `▸ 📦 v0.3.0` rendering and the unclamped 101-rune item).
- [x] T2: implement the renderer changes in `cmd/selfupdate/notes.go` —
      `maxReleaseNotesItemChars = 100`, `releaseNotesVersionToken`, and
      `clampReleaseNoteItem` (budget includes the ellipsis: 99 text runes + `…`).
- [x] T3: update the CLI contract pins in `cmd/tests/update_cli_test.go` — new
      end-to-end pin `TestUpdateCLICheckHumanDigestDropsTheVersionHeadingAndClampsDenseBullets`
      over a curated v0.3.0-shaped body; proven load-bearing by re-arming RED
      with each rule neutralized. No existing CLI pin referenced the old behavior.
- [x] T4: full verification (`go test ./...`, `go vet ./...`) and evidence —
      `go test -count=1 ./...` all green, `go vet` clean, `gofmt -l` clean.

## Work-unit commit identity

- `2bbe807` feat(update): clamp digest items and drop the version heading
  (cmd/selfupdate/notes.go, cmd/selfupdate/notes_test.go,
  cmd/tests/update_cli_test.go, odd/tasks/update-digest-legibility.md)
- Review declaration: this commit is the frozen candidate for the native RDD
  review; the expected outcome is an ordinary review over the diff against
  `faceb0b` (develop at branch time).
