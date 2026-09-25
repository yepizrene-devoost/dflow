package selfupdate

import (
	"regexp"
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

// maxReleaseNotesItemChars bounds how many runes of a list item's text a bullet
// line may keep before it is clipped with an ellipsis. Like the two caps above it
// is a readability bound rather than a format guarantee: real curated release
// notes often carry one dense prose bullet per change, which wraps across several
// terminal lines and forces the reader to reassemble it mid-word, while the full
// release page is printed as the closing line of the report. Counting runes
// rather than bytes keeps the bound meaningful for non-ASCII notes, and the clamp
// applies to bullets only: headings and plain lines stay bounded by
// maxReleaseNotesLines and maxReleaseNotesChars alone.
const maxReleaseNotesItemChars = 100

// releaseNotesEllipsis is the exact line appended when the summarizer had to
// drop content, so a reader can tell a complete summary from a clipped one.
const releaseNotesEllipsis = "…"

// releaseNotesHeadingMarker prefixes a Markdown section heading once its hashes
// are stripped, so "### Added" reaches the terminal as "▸ Added": the marker
// keeps the title visually distinct without the literal hashes a terminal would
// print.
const releaseNotesHeadingMarker = "▸ "

// releaseNotesSeparator is the empty line emitted before each section heading
// but the summary's first, so a multi-section digest reads block by block instead
// of as one wall of text. It is an empty string rather than a padded or
// whitespace-bearing line, which is what keeps it free of trailing whitespace in
// the terminal, and it is presentation rather than content, so it is exempt from
// both summary caps and can never trigger the ellipsis.
const releaseNotesSeparator = ""

// releaseNotesBulletMarker replaces a Markdown list item's dash, so "- item"
// reads as "• item" instead of as a line that happens to start with a hyphen.
const releaseNotesBulletMarker = "• "

// releaseNotesVersionToken recognizes the semantic-version token ("v0.3.0" or
// "0.3.0") that a release's own version heading carries. A heading matching it
// names the release the reader is already looking at, so it is dropped instead
// of repeated under the caller's "what's new in vX:" line. The optional leading
// "v" covers both spellings a maintainer may use, the word boundaries keep a
// version glued to surrounding text ("path0.3.0") from matching, and matching
// anywhere in the heading tolerates the emoji and decoration wrapped around it.
var releaseNotesVersionToken = regexp.MustCompile(`\bv?\d+\.\d+\.\d+\b`)

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
//     releaseNotesHeadingMarker: "## What's changed" renders as
//     "▸ What's changed" and "### Added" as "▸ Added". Heading depth is not
//     preserved — every level gets the same single marker, so an indented
//     heading does not keep its indentation.
//   - A section heading whose text carries a version number is dropped outright,
//     exactly like the top title: the release body's own
//     "## 📦 v0.3.0 – Self-Update, Installers & CLI Contracts" duplicates the
//     "what's new in vX:" header the command prints, so rendering it would spend
//     the reader's first summary line repeating the version they just read. The
//     version is recognized as a releaseNotesVersionToken anywhere in the
//     heading, at any heading depth, while a version-free heading such as
//     "### Added" is still marked. Like the title rule this removes formatting
//     rather than content, so it never contributes to the ellipsis below.
//   - A list item ("- " after trimming) swaps its dash for
//     releaseNotesBulletMarker while keeping its leading indentation: "- item"
//     renders as "• item" and "  - nested" as "  • nested".
//   - A list item's text is then clamped to maxReleaseNotesItemChars runes,
//     ending in releaseNotesEllipsis when it had to be clipped: one dense prose
//     bullet wraps across several terminal lines and forces the reader to
//     reassemble it mid-word, while the full release page is printed as the
//     closing line. The clamp covers the item text only, not the marker or a
//     nested bullet's indentation, and it leaves a bullet at or under the budget
//     byte-for-byte unchanged. Headings and plain lines are not clamped; they
//     stay bounded by the caps below alone.
//   - Marker detection runs on the trimmed line, so an indented bullet is still
//     recognized as one. Any other line passes through as it is: "---text" has
//     no space after the dash, so it is not a list item and stays unchanged.
//   - A section heading that is not the summary's first kept line is preceded by
//     one empty-string separator line, so consecutive sections such as
//     "### Added" and "### Fixed" read as separate blocks. The first heading gets
//     none: the caller prints "what's new in vX:" directly above the summary, so
//     a blank line there would open with a gap rather than the first line of
//     content. A separator is presentation rather than content: it never spends
//     maxReleaseNotesLines or maxReleaseNotesChars and never contributes to the
//     ellipsis below, and because it is empty it carries no indentation or
//     trailing whitespace.
//   - At most maxReleaseNotesLines content lines and maxReleaseNotesChars
//     cumulative runes are kept; the first line that would exceed either bound
//     is dropped along with everything after it. Both bounds count the final
//     rendered strings — a heading's marker included — because they exist to
//     limit what the terminal actually shows. Each line is rendered, and a
//     bullet's item clamped, before these bounds measure it, so a clipped
//     bullet counts as the shorter text the reader actually sees.
//   - When any non-blank content was dropped by those bounds, one
//     releaseNotesEllipsis line is appended so the summary visibly ends early. A
//     line removed by either formatting rule above is not dropped content: the
//     ellipsis marks truncation only.
//
// A non-blank body whose every line was formatting — for example a body that is
// only "# Changelog", or only a version heading — has nothing to render; it
// returns an empty, non-nil summary so nil keeps meaning exactly "blank body".
func SummarizeReleaseNotes(body string) []string {
	var kept []string
	total := 0
	contentKept := 0
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

		// Stop before the line that would break either bound, counting content
		// lines only: a separator is presentation, so it neither spends a line of
		// the budget nor pushes real content behind the ellipsis. Because the
		// current line is non-blank, reaching here means real content is being
		// dropped, which is exactly when the ellipsis line is owed.
		if contentKept >= maxReleaseNotesLines {
			dropped = true
			break
		}
		if total+utf8.RuneCountInString(rendered) > maxReleaseNotesChars {
			dropped = true
			break
		}

		// A section heading opens a new block, so one blank line separates it from
		// the block above — unless it is the first line the summary keeps, where
		// the caller's own header already provides the separation. The separator
		// is added after the bounds have passed, so a heading that was dropped
		// never leaves a dangling blank line behind it.
		if contentKept > 0 && strings.HasPrefix(rendered, releaseNotesHeadingMarker) {
			kept = append(kept, releaseNotesSeparator)
		}

		kept = append(kept, rendered)
		contentKept++
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
// title or release version heading, re-marking a heading, re-marking and
// clamping a bullet, or leaving the line alone. It takes the trimmed line
// precisely so detection works on indented content ("  - nested" is still a
// bullet) while the bullet's indentation is preserved.
//
// Clamping a bullet's item belongs here rather than in SummarizeReleaseNotes
// because it changes the line's shape: shaping and measuring stay separate
// steps, so the caps there always measure the text the terminal will show.
// Dropping a heading returns false with an empty string, which is how callers
// tell removed formatting apart from content the caps truncated.
//
// A heading is dropped in two cases, both of which remove formatting rather than
// content and so never owe the reader an ellipsis: the top-level title ("# "),
// which repeats the caller's own header, and a section heading carrying a
// releaseNotesVersionToken, which repeats the version the caller just named.
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
		// A heading naming the release version repeats the "what's new in vX:"
		// line printed directly above the summary, so it is dropped like the top
		// title instead of being marked.
		if releaseNotesVersionToken.MatchString(remainder) {
			return "", false
		}
		return releaseNotesHeadingMarker + remainder, true
	}

	// A bullet keeps its indentation, only swaps the marker, and has its item
	// text clamped so a dense prose bullet does not wrap mid-word.
	if strings.HasPrefix(content, "- ") {
		indent := line[:len(line)-len(strings.TrimLeftFunc(line, unicode.IsSpace))]
		return indent + releaseNotesBulletMarker + clampReleaseNoteItem(content[len("- "):]), true
	}

	return line, true
}

// clampReleaseNoteItem returns the renderable text of one bullet's item,
// shortened to maxReleaseNotesItemChars runes with releaseNotesEllipsis as its
// final rune when it did not fit.
//
// The budget bounds the returned string, ellipsis included, so a clipped item is
// never wider than an item that fit exactly; appending the ellipsis to the full
// budget instead would make clipping cost an extra column. Items at or under the
// budget are returned unchanged. Rune slices rather than byte slices keep a
// multi-byte character from being split, matching how maxReleaseNotesChars
// counts its budget.
func clampReleaseNoteItem(item string) string {
	if utf8.RuneCountInString(item) <= maxReleaseNotesItemChars {
		return item
	}
	clipped := []rune(item)[:maxReleaseNotesItemChars-1]
	return string(clipped) + releaseNotesEllipsis
}
