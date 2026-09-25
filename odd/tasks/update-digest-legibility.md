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
- Review outcome: APPROVED — lineage `review-4a42927ff1af3792`, 4/4 lenses,
  authority burned (consumed revision `sha256:20cd0ec4da355d791bed1f47a8025b3db22b44d66411980fcd050913688a4f94`).
  One non-blocking advisory: `R2-version-heading-drop-doc-overstates-formatting`
  (notes.go:57, informational).

## Work unit 2: digest presentation spacing (screenshot feedback 2026-09-25)

The approved candidate still renders as a wall of text against the real v0.3.0
body: section blocks and the closing URL run together. Presentation spacing
only — the content rules from work unit 1 are unchanged:

- Blank line before each `▸ ` section heading except the first (separates
  Added/Changed/Fixed blocks).
- Blank line before the `release notes: <url>` closing line.
- Separator lines are presentation, not content: exempt from the 15-line and
  1200-char caps and never trigger the ellipsis.
- `reportNotesSummary` prints separator lines without the two-space indent.

### Tasks

- [x] T6: pin separator emission in `cmd/selfupdate/notes_test.go` (tests first)
      — RED observed (blank separator missing, `got "  "` on the caller pin);
      negative case pinned: a dropped heading leaves no dangling blank line.
- [x] T7: implement separators in `cmd/selfupdate/notes.go` — `contentKept`
      counter (caps count content only), separator inserted after the bounds so
      dropped headings never dangle.
- [x] T8: caller indent skip in `cmd/commands/update.go` + CLI pin in
      `cmd/tests/update_cli_test.go` — separator printed verbatim (no trailing
      whitespace), blank before the release URL, JSON paths verified human-only;
      three pins proven load-bearing by re-arming RED.
- [x] T9: full verification and work-unit commits — combined with T13 into one
      commit (see the note under T13).

## Work unit 3: terminal word-wrap for icon lines (screenshot feedback 2)

The provenance warning after the banner (~230 chars in one `utils.Warn` at
`cmd/commands/update.go:146`) hard-wraps at the terminal edge mid-word
("A b / inary") with the continuation at column 0, under the icon. Same class of
problem anywhere an icon line exceeds the width. Presentation only:

- Wrap inside `printWithIcon` (`cmd/utils/messages.go`) — the single rendering
  point for every icon-prefixed line: break at word boundaries, never mid-word,
  continuation lines hang-indent aligned to the content column (4 spaces,
  matching `%-3s `).
- Width from `term.GetSize` on stdout (dependency already present via
  `cmd/utils/tty.go`); only when stdout is a TTY — piped/scripted output stays
  unwrapped for copy-paste fidelity. Fallback width 80.
- A single token wider than the width (a URL) is not broken: it gets its own
  line overflowing.
- Blank line between the provenance-warning block and the report body
  ("current version:") so the alert reads as its own paragraph.
- Digest bullet lines printed via `utils.Plain` keep the 100-rune clamp from
  work unit 1; they are not re-wrapped.

### Tasks (work unit 3)

- [x] T10: pin wrapping in `cmd/utils` tests (word boundaries, hanging indent,
      TTY-only, width fallback) — tests first — RED observed as build failure
      (`undefined: stdoutWidth`, `undefined: wrapIconMessage`), then behavioral
      GREEN after implementation; 7 new tests.
- [x] T11: implement the wrap helper and wire it into `printWithIcon` —
      `stdoutWidth` seam in `cmd/utils/tty.go`, greedy word-boundary fill with
      4-space hanging indent, 80-column floor, overlong tokens overflow alone;
      `Plain`/`Notice`/`Prompt` untouched.
- [x] T12: blank line after the provenance warning in the check report and the
      startup notice; CLI pin update — the startup notice renders a different
      message via `utils.Notice` as the run's last line, so no separation applies
      there (single Warn render site confirmed at `update.go:146`); pin proven
      load-bearing by neutralizing the blank line.
- [x] T13: full verification and work-unit commits — `go test -count=1 ./...`
      green, `go vet` clean, `gofmt` clean. Commit note: WU2 and WU3 hunks
      interleave in `cmd/commands/update.go` and `cmd/tests/update_cli_test.go`,
      so both units land as one commit to keep every commit green; the review
      was already planned as combined.

## Work unit 4: unified text measure — wrap replaces clamp (screenshot feedback 3)

Two connected symptoms against the real v0.3.0 body on a ~200-column terminal:
the digest still clamps bullets at 100 runes (mid-word `…` cuts, half the line
empty), while icon lines fill the full terminal width (~196 cols) — two
different text measures side by side read as misaligned, and a 196-column prose
line is unreadable. User decision: no more truncation; the full bullet text is
shown, wrapped.

Design:

- **One readable measure for the whole report**: effective content width =
  `min(measured − indent, 100)`, floor 80, TTY-only. Same measure for icon
  lines (printWithIcon) and digest lines (renderer).
- **The 100-rune item clamp is removed.** `SummarizeReleaseNotes` takes a
  content width parameter (stays pure): bullets and plain content lines wrap at
  word boundaries with hanging indent aligned under the bullet text; a width ≤ 0
  (non-TTY) means unbounded — no wrap, no clamp.
- **Flood caps move to physical lines (40) and 4000 chars**; the ellipsis marks
  dropped content only at those caps, never mid-word.
- Heading/bullet markers, H1 and version-heading drop, nil-vs-empty contract
  unchanged.
- Also fixes advisory `R2-stray-duplicate-work-unit-3-tasks-heading` (duplicate
  heading in this file) and, naturally, `R2-heading-class-sniffed-from-rendered-prefix`
  (heading detection moves before rendering).

### Tasks

- [x] T14: tests first — wrap-at-measure contract in `cmd/selfupdate/notes_test.go`
      and measure cap in `cmd/utils/messages_test.go` — RED observed (new
      signature + removed clamp constants), GREEN after implementation.
- [x] T15: renderer takes content width (`SummarizeReleaseNotes(body, contentWidth)`,
      pure, 0 = unbounded); exported `utils.TerminalWidth()` wrapping the
      `stdoutWidth` seam; `reportNotesSummary` passes
      `min(measured−2, maxDigestWidth=100)`, floor 40, 0 when unmeasured. Fill
      helper shared as a local `wrapReleaseNoteText` copy (cmd/selfupdate must
      not import cmd/utils).
- [x] T16: `printWithIcon` ceiling `maxWrapWidth=100` over the existing 80 floor;
      clamp tests replaced by wrap tests; heading class decided from the raw line
      (`releaseNoteLineKind`) before rendering — closes advisory
      `R2-heading-class-sniffed-from-rendered-prefix`.
- [x] T17: full verification — `go test ./...` green, `go vet` clean, `gofmt`
      clean; work-unit commit below; RDD review of the new candidate declared in
      the commit identity section.

## Work-unit commit identity (combined)

- `2bbe807` feat(update): clamp digest items and drop the version heading (WU1)
- `6171857` feat(update): space and wrap the update report for terminal
  legibility (WU2 + WU3; hunks interleave in two shared files, so one commit
  keeps every commit green — see T13)
- Review outcome: APPROVED — lineage `review-42908f606d1b3841`, 4/4 lenses,
  authority burned (consumed revision `sha256:254552772891af242587f99e4281973d4e1169a26f0255b17e947d677d36c358`).
  Non-blocking advisories carried into WU4: `R2-heading-class-sniffed-from-rendered-prefix`,
  `R2-stray-duplicate-work-unit-3-tasks-heading` (fixed in this file by that
  edit), plus WU1's `R2-version-heading-drop-doc-overstates-formatting`.
- WU4 commit identity: recorded as it lands.
- `806d4e9` feat(update): wrap the report on one shared text measure instead of
  clamping (WU4; renderer width param, clamp removed, caps 40 lines / 4000
  chars, icon measure ceiling 100, heading class decided before rendering)
- Review declaration: `806d4e9` is the frozen candidate for the native RDD
  review of work unit 4; expected outcome is an ordinary review over the diff
  against `eacde90`.
