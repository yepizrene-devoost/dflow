package selfupdate

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestSummarizeReleaseNotes covers the line-shaping rules a reader depends on:
// an empty or whitespace-only body summarizes to nothing, otherwise each
// rendered line is cleaned of trailing whitespace and blank runs are collapsed,
// while list indentation is left alone.
func TestSummarizeReleaseNotes(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		want    []string
		wantNil bool
	}{
		{name: "empty body", body: "", wantNil: true},
		{name: "whitespace-only body", body: "   \n\t\n\r\n", wantNil: true},
		{
			name: "drops the title and renders the bullets",
			body: "# v0.3.0\n\n- fixed a bug\n- added a feature",
			want: []string{"• fixed a bug", "• added a feature"},
		},
		{
			name: "collapses blank lines",
			body: "first\n\n\n   \nsecond",
			want: []string{"first", "second"},
		},
		{
			name: "trims trailing whitespace",
			body: "change one   \nchange two\t",
			want: []string{"change one", "change two"},
		},
		{
			name: "strips CRLF carriage returns",
			body: "first\r\nsecond\r\n",
			want: []string{"first", "second"},
		},
		{
			name: "preserves bullet indentation",
			body: "- top\n  - nested\n    - deeper",
			want: []string{"• top", "  • nested", "    • deeper"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want nil", tc.body, got)
				}
				return
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", tc.body, got, tc.want)
			}
		})
	}
}

// TestSummarizeReleaseNotesRendersTerminalShapedLines pins the Markdown-to-
// terminal rendering: the redundant top title disappears, headings lose their
// hashes behind a marker, bullets swap their dash while keeping indentation, and
// everything else — inline code included — passes through untouched.
func TestSummarizeReleaseNotesRendersTerminalShapedLines(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "h2 heading",
			body: "## 📦 v0.2.0 – Finish Automation",
			want: []string{"▸ 📦 v0.2.0 – Finish Automation"},
		},
		{
			name: "h3 heading",
			body: "### Added",
			want: []string{"▸ Added"},
		},
		{
			name: "deeper heading keeps one marker",
			body: "#### Fixed",
			want: []string{"▸ Fixed"},
		},
		{
			name: "indented heading is detected and renders at the marker",
			body: "  ## Added",
			want: []string{"▸ Added"},
		},
		{
			name: "heading and bullets together",
			body: "## 📦 v0.2.0\n### Added\n- `dflow finish` command",
			want: []string{"▸ 📦 v0.2.0", "▸ Added", "• `dflow finish` command"},
		},
		{
			name: "indented bullet keeps its indentation",
			body: "- top\n  - nested",
			want: []string{"• top", "  • nested"},
		},
		{
			name: "dash without a space is not a bullet",
			body: "---text",
			want: []string{"---text"},
		},
		{
			name: "inline code keeps its backticks",
			body: "Run `dflow update --check` first.",
			want: []string{"Run `dflow update --check` first."},
		},
		{
			name: "plain text passes through",
			body: "A release with no markup at all.",
			want: []string{"A release with no markup at all."},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", tc.body, got, tc.want)
			}
		})
	}
}

// TestSummarizeReleaseNotesRendersARealReleaseBody pins the whole contract
// against the shape GoReleaser actually publishes, the one a maintainer saw
// rendered as raw markup: the duplicated top title, the version heading, the
// section headings and the bullet list.
func TestSummarizeReleaseNotesRendersARealReleaseBody(t *testing.T) {
	body := "# Changelog\n" +
		"\n" +
		"## 📦 v0.2.0 – Finish Automation\n" +
		"\n" +
		"### Added\n" +
		"\n" +
		"- `dflow finish` command to close a feature branch\n" +
		"- `dflow update` command to install the latest release\n" +
		"\n" +
		"### Fixed\n" +
		"\n" +
		"- keep the current binary when a download fails\n"

	got := SummarizeReleaseNotes(body)

	want := []string{
		"▸ 📦 v0.2.0 – Finish Automation",
		"▸ Added",
		"• `dflow finish` command to close a feature branch",
		"• `dflow update` command to install the latest release",
		"▸ Fixed",
		"• keep the current binary when a download fails",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SummarizeReleaseNotes(release body) = %#v, want %#v", got, want)
	}
}

// TestSummarizeReleaseNotesDropsTheH1Title pins the top-title rule together
// with its bookkeeping consequence: the dropped title is formatting, not
// truncation, so a body that fits entirely must render without an ellipsis even
// though a line was removed.
func TestSummarizeReleaseNotesDropsTheH1Title(t *testing.T) {
	body := "# Changelog\n\n### Added\n\n- `dflow finish` command\n"

	got := SummarizeReleaseNotes(body)

	want := []string{"▸ Added", "• `dflow finish` command"}
	if !slices.Equal(got, want) {
		t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", body, got, want)
	}
	if slices.Contains(got, "…") {
		t.Fatalf("summary = %#v, want no ellipsis: dropping the title is formatting, not truncation", got)
	}
}

// TestSummarizeReleaseNotesKeepsNilReservedForBlankBodies pins what an
// all-formatting body returns. Rendering removes the only line, but the body was
// not blank, so nil keeps meaning exactly "blank body" and callers can tell the
// two cases apart.
func TestSummarizeReleaseNotesKeepsNilReservedForBlankBodies(t *testing.T) {
	got := SummarizeReleaseNotes("# Changelog")

	if got == nil {
		t.Fatal("SummarizeReleaseNotes(\"# Changelog\") = nil, want an empty, non-nil summary")
	}
	if len(got) != 0 {
		t.Fatalf("SummarizeReleaseNotes(\"# Changelog\") = %#v, want no lines", got)
	}
}

// TestSummarizeReleaseNotesCapsTheRenderedLine fixes the order of rendering and
// measuring: the character bound counts the returned string, so a heading's
// marker is part of the budget. A heading rendering to exactly the cap fits, and
// one rune more clips the whole body behind the ellipsis.
func TestSummarizeReleaseNotesCapsTheRenderedLine(t *testing.T) {
	// "▸ " is two runes, so maxReleaseNotesChars-2 runes of heading text render to
	// exactly the cap.
	fits := "## " + strings.Repeat("x", maxReleaseNotesChars-2)
	wantFits := "▸ " + strings.Repeat("x", maxReleaseNotesChars-2)
	if got := SummarizeReleaseNotes(fits); len(got) != 1 || got[0] != wantFits {
		t.Fatalf("a heading rendering to exactly the cap = %#v, want it kept as %q", got, wantFits)
	}

	clips := "## " + strings.Repeat("x", maxReleaseNotesChars-1)
	if got := SummarizeReleaseNotes(clips); len(got) != 1 || got[0] != "…" {
		t.Fatalf("a heading rendering past the cap = %#v, want [\"…\"]", got)
	}
}

// TestSummarizeReleaseNotesCapsLineCount pins the line bound and the ellipsis
// line: a body with more lines than the cap keeps exactly the cap's worth and
// then announces the drop with one final ellipsis.
func TestSummarizeReleaseNotesCapsLineCount(t *testing.T) {
	lines := make([]string, 0, maxReleaseNotesLines+5)
	for i := 1; i <= maxReleaseNotesLines+5; i++ {
		lines = append(lines, fmt.Sprintf("line %02d", i))
	}

	got := SummarizeReleaseNotes(strings.Join(lines, "\n"))

	if len(got) != maxReleaseNotesLines+1 {
		t.Fatalf("len = %d, want %d (the %d-line cap plus the ellipsis)", len(got), maxReleaseNotesLines+1, maxReleaseNotesLines)
	}
	if got[0] != "line 01" {
		t.Fatalf("first line = %q, want %q", got[0], "line 01")
	}
	wantLastKept := fmt.Sprintf("line %02d", maxReleaseNotesLines)
	if got[maxReleaseNotesLines-1] != wantLastKept {
		t.Fatalf("last kept line = %q, want %q", got[maxReleaseNotesLines-1], wantLastKept)
	}
	if got[maxReleaseNotesLines] != "…" {
		t.Fatalf("final line = %q, want the ellipsis %q", got[maxReleaseNotesLines], "…")
	}
}

// TestSummarizeReleaseNotesKeepsTheCapWithoutEllipsis fixes the boundary: a
// body of exactly the cap's worth of lines is complete, so no ellipsis line is
// owed and none is added.
func TestSummarizeReleaseNotesKeepsTheCapWithoutEllipsis(t *testing.T) {
	lines := make([]string, 0, maxReleaseNotesLines)
	for i := 1; i <= maxReleaseNotesLines; i++ {
		lines = append(lines, fmt.Sprintf("line %02d", i))
	}

	got := SummarizeReleaseNotes(strings.Join(lines, "\n"))

	if len(got) != maxReleaseNotesLines {
		t.Fatalf("len = %d, want %d with no ellipsis", len(got), maxReleaseNotesLines)
	}
	if slices.Contains(got, "…") {
		t.Fatalf("summary = %#v, want no ellipsis for a body that exactly fits", got)
	}
}

// TestSummarizeReleaseNotesCapsTheCharacterBudget pins the character bound:
// lines are added until the next one would exceed the budget, at which point it
// and the rest are dropped behind a single ellipsis line. The 100-rune lines
// keep the arithmetic exact and below the line cap.
func TestSummarizeReleaseNotesCapsTheCharacterBudget(t *testing.T) {
	const lineLen = 100
	fit := maxReleaseNotesChars / lineLen
	line := strings.Repeat("x", lineLen)

	lines := make([]string, 0, fit+1)
	for i := 0; i <= fit; i++ {
		lines = append(lines, line)
	}

	got := SummarizeReleaseNotes(strings.Join(lines, "\n"))

	if len(got) != fit+1 {
		t.Fatalf("len = %d, want %d (%d full lines plus the ellipsis)", len(got), fit+1, fit)
	}
	for i := 0; i < fit; i++ {
		if got[i] != line {
			t.Fatalf("line %d = %q, want the full %d-rune line", i, got[i], lineLen)
		}
	}
	if got[fit] != "…" {
		t.Fatalf("final line = %q, want the ellipsis %q", got[fit], "…")
	}
}

// TestSummarizeReleaseNotesReportsAClippedOversizedLine records that an
// over-budget single line is not silently discarded: the body is non-blank, so
// the summary is the ellipsis line rather than nil.
func TestSummarizeReleaseNotesReportsAClippedOversizedLine(t *testing.T) {
	got := SummarizeReleaseNotes(strings.Repeat("x", maxReleaseNotesChars+1))

	if len(got) != 1 || got[0] != "…" {
		t.Fatalf("SummarizeReleaseNotes(oversized single line) = %#v, want [\"…\"]", got)
	}
}

// TestSummarizeReleaseNotesCountsCharactersNotBytes locks the unit of the
// character budget: a line of 700 two-byte runes is 1400 bytes but only 700
// characters, so it fits and must be kept whole.
func TestSummarizeReleaseNotesCountsCharactersNotBytes(t *testing.T) {
	line := strings.Repeat("é", 700)

	got := SummarizeReleaseNotes(line)

	if len(got) != 1 || got[0] != line {
		t.Fatalf("SummarizeReleaseNotes(700-rune line) = %#v, want it kept whole", got)
	}
}
