package selfupdate

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxReleaseNotesLines bounds how many changelog lines a terminal summary may
// keep. It is a readability bound for a person scanning an update notice, not a
// guarantee about the changelog format: a release that ships more headings than
// this still gets summarized, just with its tail replaced by one ellipsis line.
const maxReleaseNotesLines = 15

// maxReleaseNotesChars bounds the cumulative size of a summary, counted in
// runes of rendered line content. Like maxReleaseNotesLines it is a
// terminal-friendly bound rather than a format guarantee: it keeps a release
// with a few enormous paragraphs from flooding the screen, and the dropped tail
// is represented by one ellipsis line. Counting runes rather than bytes keeps
// the bound meaningful for changelogs containing non-ASCII text.
const maxReleaseNotesChars = 1200

// releaseNotesEllipsis is the exact line appended when the summarizer had to
// drop content, so a reader can tell a complete summary from a clipped one.
const releaseNotesEllipsis = "…"

// releaseNotesHeadingMarker prefixes a Markdown section heading once its hashes
// are stripped, so "### Added" reaches the terminal as "▸ Added": the marker
// keeps the title visually distinct without the literal hashes a terminal would
// print.
const releaseNotesHeadingMarker = "▸ "

// releaseNotesBulletMarker replaces a Markdown list item's dash, so "- item"
// reads as "• item" instead of as a line that happens to start with a hyphen.
const releaseNotesBulletMarker = "• "

// SummarizeReleaseNotes turns a GitHub release body — the raw Markdown
// GoReleaser publishes from CHANGELOG.md — into the short list of lines a user
// can scan directly in a terminal.
//
// The function is pure and deterministic: it does no I/O, never detects the
// terminal width and never emits ANSI escape codes, so the same body always
// yields the same summary and callers can test it without a TTY. It returns nil
// for an empty or whitespace-only body, and otherwise one string per rendered
// line under these rules:
//
//   - Lines are split on "\n" and whitespace-only lines are skipped, which
//     collapses the blank runs Markdown uses to separate sections.
//   - Each kept line has its trailing whitespace (including a CRLF's "\r")
//     trimmed, while leading whitespace is preserved so list indentation keeps
//     its shape.
//   - Terminal rendering: the returned lines are terminal-shaped rather than
//     the body's raw Markdown, because markup such as "# Changelog" or "### Added"
//     is noise when there is no renderer between the string and the screen. This
//     is deliberately a light in-house rendering, not a Markdown renderer: there
//     is no styling, no line wrapping, no nested-list or emphasis handling, and
//     inline code keeps its backticks because they already read as "this is code".
//   - A top-level title line ("# " after trimming) is dropped outright: the
//     release body's "# Changelog" duplicates the header the command prints
//     above the summary, so repeating it would spend the reader's first line on
//     a title they just read. Because this removes formatting rather than
//     content, it never contributes to the ellipsis below.
//   - A section heading ("##" or deeper after trimming) loses its hashes and the
//     whitespace after them, and the remainder is prefixed with
//     releaseNotesHeadingMarker: "## 📦 v0.2.0 – Finish Automation" renders as
//     "▸ 📦 v0.2.0 – Finish Automation" and "### Added" as "▸ Added". Heading
//     depth is not preserved — every level gets the same single marker, so an
//     indented heading does not keep its indentation.
//   - A list item ("- " after trimming) swaps its dash for
//     releaseNotesBulletMarker while keeping its leading indentation: "- item"
//     renders as "• item" and "  - nested" as "  • nested".
//   - Marker detection runs on the trimmed line, so an indented bullet is still
//     recognized as one. Any other line passes through as it is: "---text" has
//     no space after the dash, so it is not a list item and stays unchanged.
//   - At most maxReleaseNotesLines content lines and maxReleaseNotesChars
//     cumulative runes are kept; the first line that would exceed either bound
//     is dropped along with everything after it. Both bounds count the final
//     rendered strings — a heading's marker included — because they exist to
//     limit what the terminal actually shows.
//   - When any non-blank content was dropped by those bounds, one
//     releaseNotesEllipsis line is appended so the summary visibly ends early. A
//     title removed by the "# " rule is not dropped content: the ellipsis marks
//     truncation only.
//
// A non-blank body whose every line was formatting — for example a body that is
// only "# Changelog" — has nothing to render; it returns an empty, non-nil
// summary so nil keeps meaning exactly "blank body".
func SummarizeReleaseNotes(body string) []string {
	var kept []string
	total := 0
	dropped := false
	sawContent := false

	for _, line := range strings.Split(body, "\n") {
		// A whitespace-only line is not content: skipping it collapses the
		// blank runs between Markdown sections without emitting empty strings.
		if strings.TrimSpace(line) == "" {
			continue
		}
		sawContent = true

		// Render first, then measure: the caps bound the terminal lines, so they
		// are applied to the string the reader will actually see.
		rendered, isContent := renderReleaseNoteLine(strings.TrimRightFunc(line, unicode.IsSpace))
		if !isContent {
			continue
		}

		// Stop before the line that would break either bound. Because the
		// current line is non-blank, reaching here means real content is being
		// dropped, which is exactly when the ellipsis line is owed.
		if len(kept) >= maxReleaseNotesLines {
			dropped = true
			break
		}
		if total+utf8.RuneCountInString(rendered) > maxReleaseNotesChars {
			dropped = true
			break
		}

		kept = append(kept, rendered)
		total += utf8.RuneCountInString(rendered)
	}

	if !sawContent {
		return nil
	}

	if dropped {
		kept = append(kept, releaseNotesEllipsis)
	}
	if kept == nil {
		// Every line was formatting that rendering removed. The body was not
		// blank, so the result must not be nil: callers use nil to mean "no
		// release notes at all" and an empty slice to mean "nothing worth
		// printing".
		kept = []string{}
	}
	return kept
}

// renderReleaseNoteLine turns one already trailing-trimmed release-note line
// into its terminal shape and reports whether it should be kept at all.
//
// It is the whole per-line half of the rendering contract: dropping a redundant
// title, re-marking a heading, re-marking a bullet, or leaving the line alone.
// It takes the trimmed line precisely so detection works on indented content
// ("  - nested" is still a bullet) while the bullet's indentation is preserved.
func renderReleaseNoteLine(line string) (string, bool) {
	content := strings.TrimSpace(line)

	// The release body's top title repeats the "what's new in vX:" header the
	// command prints, so it is dropped rather than rendered.
	if strings.HasPrefix(content, "# ") {
		return "", false
	}

	// "##" or deeper is a section heading: strip the hashes plus the whitespace
	// after them and mark the remainder. The marker replaces the depth, so the
	// original indentation is not carried over.
	if strings.HasPrefix(content, "##") {
		remainder := strings.TrimLeftFunc(strings.TrimLeft(content, "#"), unicode.IsSpace)
		return releaseNotesHeadingMarker + remainder, true
	}

	// A bullet keeps its indentation and only swaps the marker.
	if strings.HasPrefix(content, "- ") {
		indent := line[:len(line)-len(strings.TrimLeftFunc(line, unicode.IsSpace))]
		return indent + releaseNotesBulletMarker + content[len("- "):], true
	}

	return line, true
}
