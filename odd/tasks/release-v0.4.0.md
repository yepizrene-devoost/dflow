# release-v0.4.0

Prepare and publish `v0.4.0`: everything merged into `develop` since the
`v0.3.0` tag is unreleased.

## Issues

Closes #37 (`chore(release): ship v0.4.0`)

Branch: `release/v0.4.0` (base: `develop`)

## Goal

Turn the ~30 commits merged into `develop` since `v0.3.0` into a published
release: a curated `CHANGELOG.md` section for `v0.4.0`, its `HISTORY.md` entry,
a `develop` merge, the promotion PR into `main`, the tag on `main`, and the
published GitHub release.

## Design decisions (resolved 2026-09-25)

- **The version is `v0.4.0` — a minor, not a major and not a patch.** #37
  proposed it, but its own comment says this repository documents no semver
  policy, so the maintainer picks the string. The maintainer asked for the
  technical read and confirmed it. The reasoning:

  - **Not major.** `1.0.0` is not "a bigger change", it is a promise of surface
    stability, and there is no break to justify it. Verified against the tree,
    not the issue text: `cmd/` only gained surface (`dflow agent`, plus
    `--agents`, `--install`, `--local`, `--json`, `--path`, `--force`), no
    command or flag was removed, and the `--json` contracts are unchanged.
  - **Not patch.** `pkg/flow` — the `.dflow.yaml` schema — has an empty diff, so
    there is no config-compatibility change either; but `v0.3.1` would still
    understate the range, because it adds a command and an install capability
    rather than correcting behavior.
  - **Minor.** Under `0.y.z` the leading `0` already says the surface may still
    break, the middle digit carries features, the last one carries fixes. This
    range is additive with no contract break: features.
  - The precedent of `v0.1.2` shipping 11 `feat` commits on a patch bump is a
    description of what happened, not a policy; repeating it would only
    perpetuate the inconsistency.

- **No `Changed` entry for `Agent.DisplayName`.** #37 asks for at least one
  `Changed` entry naming a `pkg/*` symbol removed in this range and points at
  `Agent.DisplayName`, dropped in `46a3f04`. It does not qualify: the field was
  **added in `5812032` and removed in `46a3f04`, both inside `v0.3.0..HEAD`**,
  and `pkg/agent` did not exist at `v0.3.0` (the whole package is 2,588 inserted
  lines with zero net deletions across the range). A symbol that never existed
  in a published version cannot break a consumer of a published version, so an
  entry naming it would document an intra-release edit as a user-facing change.
  The maintainer chose "no entry" when asked. `pkg/agent` is documented as
  `Added`, which is what the diff says. The general rule `7cd7c28` wrote into
  `RELEASING.md` step 3 stays as it is: this release simply has no commit that
  satisfies it.

- **The curation is the deliverable, not the git-cliff output.** `make changelog`
  writes a gitignored `CHANGELOG.draft.md`; the durable artifact is the curated
  `## 📦` section, because `make release` extracts it into the published release
  body and validates its version against the tag.

- **Intra-release hardening is not a `Fixed` item.** The two safety behaviors of
  the new `pkg/agent` path — mode preservation on rewrite and atomic writes —
  were drafted under `Fixed` and were moved to `Added` as properties of the
  brand-new command. The reason is the same one that produced the `DisplayName`
  decision: neither defect was ever published, so a `Fixed` entry would report a
  change no reader of the previous release could have experienced, and would
  imply `dflow agent` existed at `v0.3.0` when it did not. This is not a
  stylistic preference — checked against the file, every other `Fixed` entry in
  this repository documents a change against the previous tag: `dflow delete`
  idempotency and the `init` re-init refusal were real fixes because
  `cmd/commands/delete.go` and `cmd/commands/init.go` both exist at `v0.2.0`,
  and the `Makefile` "no longer injects the latest release tag" entry corrects
  `VERSION ?= $(shell git describe --tags --abbrev=0)`, which `git show
  v0.2.0:Makefile` shows was already there. The two remaining `Fixed` entries
  here (`make release` fail-fast, the read-only release-checks CI job) are
  genuine, because the release path and its gaps shipped in `v0.3.0`.

## Curation corrections after verification

The curation was written by a delegated `gentle-ai-worker` and then verified by
an independent `gentle-ai-verify` pass (read-only, source only), which found
four unsupported claims. All four were corrected before this stage closed, and
the runtime plan below was read back with `go run . agent … --json`, which is
strictly read-only by construction (`cmd/commands/agent.go` returns from the
`FormatJSON` branch before any write, reference or skill).

| Claim as drafted | Correction | Deciding evidence |
| --- | --- | --- |
| "`dflow update --check` **and the update notice** render their report at one shared text measure" | attributed to the `dflow update` report alone | `cmd/utils/messages.go:137` — the startup notice is a single unwrapping `utils.Notice` line on stderr, and `git log v0.3.0..HEAD -- cmd/root/updatenotify.go` is empty |
| "`AGENTS.md` is **always** ensured"; "a run **never** drops a surprise `CLAUDE.md`" | both qualified by the selection | `agent --agents claude --json` plans `CLAUDE.md` (`create: true`) and no `AGENTS.md`; `pkg/agent/registry.go:183-192` |
| "The default targets the portable `.agents/skills` root that pi, codex and opencode all discover" | the default installs under the user's home; the shared root is the project one | `agent --install --json` → `~/.agents/skills/dflow` `[pi]`; `--install --agents all` → four distinct user roots; `--install --local --agents all` → `.agents/skills/dflow` `[pi, codex, opencode]` plus `.claude/skills/dflow` |
| "the skill is kept current, while **a hand-edited document is left untouched**" | names which artifact is which: a hand-edited `SKILL.md` **is** rewritten without `--force`; only the workflow document is protected | `pkg/agent/install.go:159-161` — "a stale copy is a bug rather than a local edit to protect" |

## Tasks

- [x] **WU1** — `make changelog`, then curate the draft into a
      `## 📦 v0.4.0 – <title>` section of `CHANGELOG.md`: prune internal noise
      (`docs(odd)` identity records, `chore(ci)`/`chore(release)` plumbing),
      reword for users, keep the human-readable voice, and give the removed
      `Agent.DisplayName` no entry per the decision above.
      Surfaces: `CHANGELOG.md`.

      Evidence: the section is `## 📦 v0.4.0 – Multi-Agent Install, Legible
      Digests & Release Safeguards`, inserted above `v0.3.0`, with `### Added`
      (5), `### Changed` (2) and `### Fixed` (2) and no `### Internal`. All 19
      draft lines were dispositioned: 9 `Added` lines kept, folded or split,
      and all 7 internal ones pruned or folded (the two `test(cmd)` commits and
      `e80b49f` are pruned as non-observable; the release-path commits are
      folded into the two `Fixed` bullets). The nine lines about the four
      interim digest iterations collapse into the two `Changed` bullets,
      because only the final rule is user-visible. `grep -nE
      'DisplayName|### Internal|#[0-9]+'` over the section is empty. The
      changelog contract passes — `make release-notes` then
      `head -n 1 RELEASE_NOTES.md | awk '{print $3}'` prints `v0.4.0`, and the
      CI assertion verbatim (`awk '/^## 📦/' …`, `.github/workflows/
      release-checks.yml:62-97`) prints `CHANGELOG contract OK: ## 📦 v0.4.0 –
      Multi-Agent Install, Legible Digests & Release Safeguards`. The generated
      `RELEASE_NOTES.md` was deleted afterwards: it is gitignored and never
      committed.
- [x] **WU2** — add the `v0.4.0` entry to `HISTORY.md` (tag link and date) and
      run `go test ./...`.
      Surfaces: `HISTORY.md`.

      Evidence: the entry sits above `v0.3.0` with a byte-identical heading, the
      `v0.4.0` release-tag link carrying the file's two-space hard break, `**Date:**
      2026-09-25`, and four full-sentence highlights that summarize rather than
      repeat the changelog; the second highlight was corrected in the same
      verification pass as WU1 (it claimed `CLAUDE.md` is maintained only when
      the project already has one, which `--agents claude` falsifies). `go test
      ./...` is green across every package, `cmd/tests` included (43.581s).

## Out of scope

- Any code change: this branch carries release documentation only, so the
  candidate is `CHANGELOG.md` + `HISTORY.md` + this document.
- Editing `cliff.toml`: its filters are the draft's shape, and the curation is
  where noise is pruned. Nothing in this range motivated a parser change.
- A `## Unreleased` section in `CHANGELOG.md`: it would add a state the runbook
  does not know about, and `make release` validates the newest `## 📦` section
  against the tag being published.
- Publishing, tagging, pushing, or opening the PR: each is its own decision at
  its own boundary, outside this branch's work.

## Commit identity

- WU1 + WU2 land as one work-unit commit — pending, recorded in the `docs(odd)`
  follow-up that closes this stage. They are one curation act over two files;
  the two stages were tracked separately for their surfaces, not for their
  commits.
