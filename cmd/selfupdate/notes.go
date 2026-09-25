package selfupdate

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxReleaseNotesLines bounds how many physical lines a terminal summary may
// render. It is a readability bound for a person scanning an update notice, not
// a guarantee about the changelog format: a release that ships more content than
// this still gets summarized, just with its tail replaced by one ellipsis line.
//
// It counts physical lines, not source lines: a bullet that wraps at a narrow
// width spends as many of these lines as the terminal will show, which is what
// the reader actually has to scan. A source line's wrapped lines are kept or
// dropped together, so a bullet is never half-rendered. Separator lines are
// presentation, not content, and never spend one of these lines.
const maxReleaseNotesLines = 40

// maxReleaseNotesChars bounds the cumulative size of a summary, counted in
// runes of rendered line content. Like maxReleaseNotesLines it is a
// terminal-friendly bound rather than a format guarantee: it keeps a release
// with a few enormous paragraphs from flooding the screen, and the dropped tail
// is represented by one ellipsis line. Counting runes rather than bytes keeps
// the bound meaningful for changelogs containing non-ASCII text, and separators
// — the empty string — add nothing to it.
const maxReleaseNotesChars = 4000

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

// releaseNotesBulletContinuation indents a wrapped bullet's continuation lines
// by the marker's width, so the continuation hangs under the item's text instead
// of under the bullet. It is derived from the marker rather than written as
// literal spaces so the two widths cannot drift apart.
var releaseNotesBulletContinuation = strings.Repeat(" ", utf8.RuneCountInString(releaseNotesBulletMarker))

// releaseNotesVersionToken recognizes the semantic-version token ("v0.3.0" or
// "0.3.0") that a release's own version heading carries. A heading matching it
// names the release the reader is already looking at, so it is dropped instead
// of repeated under the caller's "what's new in vX:" line. The optional leading
// "v" covers both spellings a maintainer may use, the word boundaries keep a
// version glued to surrounding text ("path0.3.0") from matching, and matching
// anywhere in the heading tolerates the emoji and decoration wrapped around it.
var releaseNotesVersionToken = regexp.MustCompile(`\bv?\d+\.\d+\.\d+\b`)

// releaseNoteLineKind is the class of a raw release-note line, decided from the
// raw text before anything is rendered.
//
// Deciding the class first is what keeps the rendering rules independent: the
// separator rule keys on "this raw line is a section heading", not on the
// rendered output starting with a marker, and the wrap rule keys on "this raw
// line is a bullet, whose continuations hang under the item" rather than on the
// number of physical lines a rendered string happens to carry. Looking for the
// marker in the output would be a second, indirect statement of the same rule,
// and a plain line that happens to start with "▸ " would be misclassified as a
// heading by it.
type releaseNoteLineKind int

const (
	// releaseNotePlain is any content line that is not a heading or a bullet. It
	// is wrapped at the content width and otherwise passes through as it is.
	releaseNotePlain releaseNoteLineKind = iota
	// releaseNoteHeading is a version-free Markdown section heading. It renders
	// behind releaseNotesHeadingMarker and opens a new block with a separator
	// unless it is the summary's first kept line.
	releaseNoteHeading
	// releaseNoteBullet is a Markdown list item. It swaps its dash for
	// releaseNotesBulletMarker and wraps with its continuations hanging under the
	// item's text.
	releaseNoteBullet
	// releaseNoteDropped is a line rendering removes entirely: the top title or a
	// section heading carrying the release's own version. Dropping it removes
	// formatting rather than content, so it never spends a cap, never owes the
	// reader an ellipsis, and never opens a section block.
	releaseNoteDropped
)

// SummarizeReleaseNotes turns a GitHub release body — the raw Markdown
// GoReleaser publishes from CHANGELOG.md — into the short list of lines a user
// can scan directly in a terminal.
//
// The function is pure and deterministic: it does no I/O, never detects the
// terminal width and never emits ANSI escape codes, so the same body and width
// always yield the same summary and callers can test it without a TTY. It
// returns nil for an empty or whitespace-only body, and otherwise one string per
// rendered physical line under these rules:
//
//   - contentWidth is the rune width available to a rendered line's text, after
//     whatever indent the caller adds around the returned strings. A value of
//     zero or less means the width was not measured — stdout is a pipe, a file
//     or a command substitution — and nothing is wrapped or cut by width, so the
//     caller renders the historical one-line-per-source-line summary. The
//     function never guesses a width; measuring a terminal belongs to the
//     caller.
//   - Lines are split on "\n" and whitespace-only lines are skipped, which
//     collapses the blank runs Markdown uses to separate sections.
//   - Each kept line has its trailing whitespace (including a CRLF's "\r")
//     trimmed, while leading whitespace is preserved so list indentation keeps
//     its shape.
//   - Terminal rendering: the returned lines are terminal-shaped rather than
//     the body's raw Markdown, because markup such as "# Changelog" or "### Added"
//     is noise when there is no renderer between the string and the screen. This
//     is deliberately a light in-house rendering, not a Markdown renderer: there
//     is no styling, no nested-list or emphasis handling, and inline code keeps
//     its backticks because they already read as "this is code".
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
//   - A bullet's item and a plain content line wider than contentWidth are
//     wrapped at word boundaries by the same greedy fill the icon helpers use
//     (wrapReleaseNoteText), so a break always replaces a space and no word is
//     ever split. A bullet's continuations are indented with
//     releaseNotesBulletContinuation, hanging under the item's text; a plain
//     line's continuations start at column zero, and a wrapped plain line is
//     re-flowed the way an icon message is: whitespace runs become single
//     spaces. A single token wider than the width — a URL, a path, a hash — is
//     left intact and overflows its line rather than being cut. Headings are not
//     wrapped: a heading is a short title, and it is left intact even when it
//     overflows. Wrapping re-flows text; it never removes any.
//   - A section heading that is not the summary's first kept line is preceded by
//     one empty-string separator line, so consecutive sections such as
//     "### Added" and "### Fixed" read as separate blocks. The first heading gets
//     none: the caller prints "what's new in vX:" directly above the summary, so
//     a blank line there would open with a gap rather than the first line of
//     content. A separator is presentation rather than content: it never spends
//     maxReleaseNotesLines or maxReleaseNotesChars and never contributes to the
//     ellipsis below, and because it is empty it carries no indentation or
//     trailing whitespace.
//   - At most maxReleaseNotesLines physical rendered lines and maxReleaseNotesChars
//     cumulative runes are kept. The first source line whose rendered lines would
//     exceed either bound is dropped along with everything after it, so a source
//     line's wrapped lines are kept or dropped together and a bullet is never
//     half-rendered. Both bounds count the final rendered strings — a heading's
//     marker included — because they exist to limit what the terminal actually
//     shows.
//   - When any non-blank content was dropped by those bounds, one
//     releaseNotesEllipsis line is appended so the summary visibly ends early. A
//     line removed by either formatting rule above is not dropped content: the
//     ellipsis marks truncation only, and there is no ellipsis anywhere else.
//
// A non-blank body whose every line was formatting — for example a body that is
// only "# Changelog", or only a version heading — has nothing to render; it
// returns an empty, non-nil summary so nil keeps meaning exactly "blank body".
func SummarizeReleaseNotes(body string, contentWidth int) []string {
	var kept []string
	total := 0
	physicalKept := 0
	dropped := false
	sawContent := false

	for _, line := range strings.Split(body, "\n") {
		// A whitespace-only line is not content: skipping it collapses the
		// blank runs between Markdown sections without emitting empty strings.
		if strings.TrimSpace(line) == "" {
			continue
		}
		sawContent = true

		// Classify and render first, then measure: the caps bound the terminal
		// lines, so they are applied to the strings the reader will actually see.
		rendered, kind := renderReleaseNoteLine(strings.TrimRightFunc(line, unicode.IsSpace), contentWidth)
		if kind == releaseNoteDropped {
			continue
		}

		// Stop before the source line whose physical lines would break either
		// bound, counting content only: a separator is presentation, so it
		// neither spends a line of the budget nor pushes real content behind the
		// ellipsis. The whole wrapped group is kept or dropped together, so no
		// bullet is ever half-rendered. Because the current line is non-blank,
		// reaching here means real content is being dropped, which is exactly
		// when the ellipsis line is owed.
		if physicalKept+len(rendered) > maxReleaseNotesLines {
			dropped = true
			break
		}
		groupRunes := 0
		for _, renderedLine := range rendered {
			groupRunes += utf8.RuneCountInString(renderedLine)
		}
		if total+groupRunes > maxReleaseNotesChars {
			dropped = true
			break
		}

		// A section heading opens a new block, so one blank line separates it from
		// the block above — unless it is the first line the summary keeps, where
		// the caller's own header already provides the separation. The separator
		// is added after the bounds have passed, so a heading that was dropped
		// never leaves a dangling blank line behind it.
		if kind == releaseNoteHeading && len(kept) > 0 {
			kept = append(kept, releaseNotesSeparator)
		}

		kept = append(kept, rendered...)
		physicalKept += len(rendered)
		total += groupRunes
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

// renderReleaseNoteLine classifies one already trailing-trimmed release-note
// line and renders it to its terminal lines.
//
// It is the whole per-line half of the rendering contract, in two ordered steps:
// decide the class from the raw text, then render per class. Dropping a redundant
// title or release version heading, re-marking a heading, re-marking and wrapping
// a bullet, or leaving the line alone are all keyed on that class, so the caller
// never has to re-derive the class from the rendered output. It takes the
// trimmed line precisely so detection works on indented content ("  - nested" is
// still a bullet) while the bullet's indentation is preserved.
//
// A dropped line returns releaseNoteDropped with no lines, which is how callers
// tell removed formatting apart from content the caps truncated.
//
// A heading is dropped in two cases, both of which remove formatting rather than
// content and so never owe the reader an ellipsis: the top-level title ("# "),
// which repeats the caller's own header, and a section heading carrying a
// releaseNotesVersionToken, which repeats the version the caller just named.
func renderReleaseNoteLine(line string, contentWidth int) ([]string, releaseNoteLineKind) {
	content := strings.TrimSpace(line)

	// The release body's top title repeats the "what's new in vX:" header the
	// command prints, so it is dropped rather than rendered.
	if strings.HasPrefix(content, "# ") {
		return nil, releaseNoteDropped
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
			return nil, releaseNoteDropped
		}
		return []string{releaseNotesHeadingMarker + remainder}, releaseNoteHeading
	}

	// A bullet keeps its indentation, swaps the marker, and wraps with its
	// continuations hanging under the item's text.
	if strings.HasPrefix(content, "- ") {
		indent := line[:len(line)-len(strings.TrimLeftFunc(line, unicode.IsSpace))]
		return wrapReleaseNoteBullet(indent, content[len("- "):], contentWidth), releaseNoteBullet
	}

	return wrapReleaseNoteText(line, contentWidth), releaseNotePlain
}

// wrapReleaseNoteBullet renders one bullet's item to its physical lines: the
// first line carries the bullet marker, each continuation carries
// releaseNotesBulletContinuation instead, and the original indentation is kept
// on every line so a nested bullet's continuations stay nested.
//
// The marker and the continuation indent are not part of the wrap budget — the
// item's text is wrapped at the width left after the marker — so the rendered
// bullet is never wider than contentWidth. A non-positive contentWidth means the
// width was not measured and the whole item is rendered on one line.
func wrapReleaseNoteBullet(indent, item string, contentWidth int) []string {
	textWidth := contentWidth - utf8.RuneCountInString(releaseNotesBulletMarker)
	chunks := wrapReleaseNoteText(item, textWidth)

	lines := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		if i == 0 {
			lines = append(lines, indent+releaseNotesBulletMarker+chunk)
			continue
		}
		lines = append(lines, indent+releaseNotesBulletContinuation+chunk)
	}
	return lines
}

// wrapReleaseNoteText breaks text into the physical lines a release-note line
// renders to, given the content width available to that text.
//
// The fill rule is the one cmd/utils.wrapIconMessage established for icon lines,
// and it is copied here deliberately rather than shared. cmd/selfupdate is a
// domain package and cmd/utils is the generic output layer: importing either
// from the other would couple layers that must not know each other (and the
// allowed edit surface for this change does not include a shared low-level text
// package). The two implementations must stay in sync; each is covered by its
// own package's tests, so a changed fill is a visible behavior change in both.
//
// It fills each line greedily from the words in order, so a break always replaces
// a space and no word is ever split across lines. A text that already fits is
// returned as a single element, unchanged; only a text that must wrap is
// re-flowed, and that re-flow normalises whitespace runs to a single space.
//
// The one exception is an overlong token: a single word wider than contentWidth
// is placed on its own line and overflows. A URL, a path or a hash is one value,
// and cutting it would corrupt it, so the terminal's own edge wrapping is the
// lesser evil.
//
// Widths are counted in runes, matching the rest of this package. A non-positive
// contentWidth is the unmeasured contract: the text is returned whole on one
// line, never wrapped one word per line and never cut.
func wrapReleaseNoteText(text string, contentWidth int) []string {
	if contentWidth < 1 {
		return []string{text}
	}
	if utf8.RuneCountInString(text) <= contentWidth {
		return []string{text}
	}

	var lines []string
	current := ""
	for _, word := range strings.Fields(text) {
		if current == "" {
			current = word
			continue
		}
		if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= contentWidth {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
