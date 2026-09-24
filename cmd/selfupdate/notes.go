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
// runes of trimmed line content. Like maxReleaseNotesLines it is a
// terminal-friendly bound rather than a format guarantee: it keeps a release
// with a few enormous paragraphs from flooding the screen, and the dropped tail
// is represented by one ellipsis line. Counting runes rather than bytes keeps
// the bound meaningful for changelogs containing non-ASCII text.
const maxReleaseNotesChars = 1200

// releaseNotesEllipsis is the exact line appended when the summarizer had to
// drop content, so a reader can tell a complete summary from a clipped one.
const releaseNotesEllipsis = "…"

// SummarizeReleaseNotes turns a GitHub release body — the raw Markdown
// GoReleaser publishes from CHANGELOG.md — into the short list of lines a user
// can scan directly in a terminal.
//
// The function is pure and deterministic: it does no I/O and never detects the
// terminal width, so the same body always yields the same summary and callers
// can test it without a TTY. It returns nil for an empty or whitespace-only
// body, and otherwise one string per rendered line under these rules:
//
//   - Lines are split on "\n" and whitespace-only lines are skipped, which
//     collapses the blank runs Markdown uses to separate sections.
//   - Each kept line has its trailing whitespace (including a CRLF's "\r")
//     trimmed, while leading whitespace is preserved so list indentation keeps
//     its shape.
//   - At most maxReleaseNotesLines content lines and maxReleaseNotesChars
//     cumulative runes are kept; the first line that would exceed either bound
//     is dropped along with everything after it.
//   - When any non-blank content was dropped, one releaseNotesEllipsis line is
//     appended so the summary visibly ends early.
func SummarizeReleaseNotes(body string) []string {
	var kept []string
	total := 0
	dropped := false

	for _, line := range strings.Split(body, "\n") {
		// A whitespace-only line is not content: skipping it collapses the
		// blank runs between Markdown sections without emitting empty strings.
		if strings.TrimSpace(line) == "" {
			continue
		}

		trimmed := strings.TrimRightFunc(line, unicode.IsSpace)

		// Stop before the line that would break either bound. Because the
		// current line is non-blank, reaching here means real content is being
		// dropped, which is exactly when the ellipsis line is owed.
		if len(kept) >= maxReleaseNotesLines {
			dropped = true
			break
		}
		if total+utf8.RuneCountInString(trimmed) > maxReleaseNotesChars {
			dropped = true
			break
		}

		kept = append(kept, trimmed)
		total += utf8.RuneCountInString(trimmed)
	}

	if dropped {
		kept = append(kept, releaseNotesEllipsis)
	}
	return kept
}
