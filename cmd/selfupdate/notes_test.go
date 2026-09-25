package selfupdate

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// Most of the tests below render with a content width of 0, the unmeasured
// contract a caller gets when stdout is not a terminal: at that width nothing
// wraps and nothing is cut by width, so those tests stay about line shaping.
// The tests named ...Wraps... or ...WithoutAMeasuredWidth... drive a measured
// width explicitly to pin the wrapping rules.

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
			want: []string{"• fixed a bug", "", "• added a feature"},
		},
		{
			name: "collapses blank lines",
			body: "first\n\n\n   \nsecond",
			// The source's blank run collapses to the one presentation blank line
			// every kept line after the first is spaced by.
			want: []string{"first", "", "second"},
		},
		{
			name: "trims trailing whitespace",
			body: "change one   \nchange two\t",
			want: []string{"change one", "", "change two"},
		},
		{
			name: "strips CRLF carriage returns",
			body: "first\r\nsecond\r\n",
			want: []string{"first", "", "second"},
		},
		{
			name: "preserves bullet indentation",
			body: "- top\n  - nested\n    - deeper",
			want: []string{"• top", "", "  • nested", "", "    • deeper"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body, 0)

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
// terminal rendering: the redundant top title and the release's own version
// heading disappear, version-free headings lose their hashes behind a marker,
// bullets swap their dash while keeping indentation, and everything else —
// inline code included — passes through untouched.
func TestSummarizeReleaseNotesRendersTerminalShapedLines(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "h2 heading",
			body: "## What's changed",
			want: []string{"▸ What's changed"},
		},
		{
			name: "h2 heading carrying the release version is dropped",
			body: "## 📦 v0.3.0 – Self-Update, Installers & CLI Contracts",
			want: []string{},
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
			want: []string{"▸ Added", "• `dflow finish` command"},
		},
		{
			name: "indented bullet keeps its indentation",
			body: "- top\n  - nested",
			want: []string{"• top", "", "  • nested"},
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
			got := SummarizeReleaseNotes(tc.body, 0)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", tc.body, got, tc.want)
			}
		})
	}
}

// TestSummarizeReleaseNotesRendersARealReleaseBody pins the whole contract
// against the shape GoReleaser actually publishes, the one a maintainer saw
// rendered as raw markup: the duplicated top title and the release's own version
// heading are both gone, while the section headings and the bullet list still
// render, spaced into an airy list by the generalized separator rule.
//
// The version heading is the legibility fix from issue #30: the command already
// prints "what's new in v0.2.0:" from the release tag, so rendering
// "▸ 📦 v0.2.0 – Finish Automation" right below it would spend the reader's first
// summary line repeating the version they just read.
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

	got := SummarizeReleaseNotes(body, 0)

	want := []string{
		"▸ Added",
		"• `dflow finish` command to close a feature branch",
		// The two bullets of one section are spaced, and the next heading keeps
		// its own separating blank line.
		"",
		"• `dflow update` command to install the latest release",
		"",
		"▸ Fixed",
		"• keep the current binary when a download fails",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SummarizeReleaseNotes(release body) = %#v, want %#v", got, want)
	}
	for _, line := range got {
		if strings.Contains(line, "📦") {
			t.Fatalf("summary = %#v, want the release's own version heading dropped: it repeats the version the command already printed", got)
		}
	}
}

// TestSummarizeReleaseNotesSpacesContentWithBlankLines pins the airy list the
// maintainer asked for on the follow-up to issue #30: a real multi-section body
// read as a wall of text, so every kept content line after the summary's first
// is preceded by one empty-string separator — which puts a blank between two
// bullets of the same section — with one exception: a line whose predecessor in
// the summary is a heading gets none, so a section's first bullet sits directly
// under its heading instead of floating away from it.
//
// Two consequences are deliberate and pinned below. The summary never opens with
// a blank line, because the caller prints "what's new in vX:" directly above it,
// so a blank there would open with a gap instead of the first line of content.
// And two consecutive headings render stacked without a blank between them: the
// exception welds every line to a heading before it, and a second heading has no
// content of its own, so it lands directly beneath the first — which matches how
// tightly a maintainer groups two adjacent section titles.
//
// The separator is the empty string rather than a padded line, so it carries no
// indentation and no trailing whitespace for a renderer to print.
func TestSummarizeReleaseNotesSpacesContentWithBlankLines(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "a heading-only body opens without a blank line",
			body: "### Added",
			want: []string{"▸ Added"},
		},
		{
			name: "content directly under the first heading needs no separator",
			body: "### Added\n\n- a short item",
			want: []string{"▸ Added", "• a short item"},
		},
		{
			name: "a plain first line is not preceded by a blank line",
			body: "intro\nmore prose",
			want: []string{"intro", "", "more prose"},
		},
		{
			name: "bullets in one section are separated by blank lines",
			body: "### Added\n- first item\n- second item",
			want: []string{"▸ Added", "• first item", "", "• second item"},
		},
		{
			name: "a heading after bullets keeps its separating blank line",
			body: "- item\n## Changed",
			want: []string{"• item", "", "▸ Changed"},
		},
		{
			name: "two consecutive headings stack without a blank between them",
			body: "## Changed\n## Fixed",
			want: []string{"▸ Changed", "▸ Fixed"},
		},
		{
			name: "a dropped version heading adds no separator of its own",
			body: "### Added\n- one item\n## 📦 v0.9.9 – Release\n- two item",
			want: []string{"▸ Added", "• one item", "", "• two item"},
		},
		{
			name: "a release body is spaced block by block",
			body: "# Changelog\n\n## 📦 v0.3.0 – Self-Update\n\n### Added\n\n- surface the update notification\n- summarize the release notes\n\n### Changed\n\n- rework the digest\n\n### Fixed\n\n- keep the current binary when a download fails\n",
			want: []string{
				"▸ Added",
				"• surface the update notification",
				"",
				"• summarize the release notes",
				"",
				"▸ Changed",
				"• rework the digest",
				"",
				"▸ Fixed",
				"• keep the current binary when a download fails",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body, 0)

			if !slices.Equal(got, tc.want) {
				t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", tc.body, got, tc.want)
			}
			for _, line := range got {
				if strings.TrimSpace(line) == "" && line != "" {
					t.Fatalf("separator line = %q, want the empty string: an indented or padded separator is trailing whitespace", line)
				}
			}
		})
	}
}

// TestSummarizeReleaseNotesKeepsALongBodyUnderTheCharacterBudgetWhole inverts
// the regression the removed physical-line cap caused. Counting rendered lines
// made a normal release overflow a forty-line budget as soon as lines wrapped,
// so the tail of the changelog disappeared behind an ellipsis for no legibility
// gain. The character budget is now the only flood guard, so both bodies below —
// one with more physical lines than the removed cap ever allowed, one that wraps
// well past it — must render completely, with no ellipsis and not a bullet lost.
//
// The bullets also make the airy spacing measurable here: every pair of bullets
// carries a separator between them, and none of those separators may spend a
// rune of the budget.
func TestSummarizeReleaseNotesKeepsALongBodyUnderTheCharacterBudgetWhole(t *testing.T) {
	cases := []struct {
		name         string
		body         string
		contentWidth int
		wantBullets  int
	}{
		{
			name:         "fifty-five bullets are more than the removed forty-line cap",
			body:         strings.TrimSpace(strings.Repeat("- item 01\n", 55)),
			contentWidth: 0,
			wantBullets:  55,
		},
		{
			name:         "forty bullets wrapped at a narrow width render eighty physical lines",
			body:         strings.TrimSpace(strings.Repeat("- alpha beta gamma\n", 40)),
			contentWidth: 12,
			wantBullets:  40,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body, tc.contentWidth)

			if slices.Contains(got, releaseNotesEllipsis) {
				t.Fatalf("summary = %#v, want no ellipsis: the body is far under the character budget, which is the only flood guard left", got)
			}
			markers := 0
			for _, line := range got {
				if strings.HasPrefix(line, releaseNotesBulletMarker) {
					markers++
				}
			}
			if markers != tc.wantBullets {
				t.Fatalf("summary = %#v, want all %d bullets rendered: only the character budget may clip content", got, tc.wantBullets)
			}
		})
	}
}

// TestSummarizeReleaseNotesSeparatorsNeverTruncate is the other half of the
// separator exemption: excluding separators from the character budget would
// still be wrong if their presence could make a complete summary look truncated.
// Seven one-item sections carry six blank lines and fourteen short content
// lines, far more than the removed line cap would have allowed, yet no content
// was dropped, so the summary must end without an ellipsis.
func TestSummarizeReleaseNotesSeparatorsNeverTruncate(t *testing.T) {
	var body strings.Builder
	for i := 1; i <= 7; i++ {
		fmt.Fprintf(&body, "## Section %d\n- item %d\n", i, i)
	}

	got := SummarizeReleaseNotes(body.String(), 0)

	// Seven headings plus seven bullets, plus the six separators between them.
	const wantLines = 20
	if len(got) != wantLines {
		t.Fatalf("len = %d, want %d (14 content lines plus 6 separators)", len(got), wantLines)
	}
	blanks := 0
	for _, line := range got {
		if line == "" {
			blanks++
		}
	}
	if blanks != 6 {
		t.Fatalf("found %d separators in %#v, want 6: one blank line between each of the 7 sections", blanks, got)
	}
	if slices.Contains(got, "…") {
		t.Fatalf("summary = %#v, want no ellipsis: nothing was dropped, so the separators must not read as truncation", got)
	}
}

// TestSummarizeReleaseNotesExemptsSeparatorsFromTheCharacterBudget pins the
// exemption. A separator is the empty string, so it adds nothing to the
// cumulative rune budget; the body below spends the whole
// maxReleaseNotesChars budget on content while carrying a separator between
// every pair of bullets, so an implementation that charged even one rune for a
// blank line would drop the tail behind an ellipsis.
func TestSummarizeReleaseNotesExemptsSeparatorsFromTheCharacterBudget(t *testing.T) {
	// "• " is two runes, so an item of 98 runes renders to exactly 100, and forty
	// such bullets spend the 4000-rune budget exactly. Those forty bullets also
	// carry thirty-nine separators, none of which may spend a rune.
	const contentLines = 40
	item := strings.Repeat("x", 98)
	line := "- " + item

	body := strings.Repeat(line+"\n", contentLines)

	// The rendered shape: a bullet, then a separator and a bullet for each of the
	// remaining thirty-nine.
	want := make([]string, 0, contentLines*2-1)
	for i := 0; i < contentLines; i++ {
		if i > 0 {
			want = append(want, releaseNotesSeparator)
		}
		want = append(want, releaseNotesBulletMarker+item)
	}

	total := 0
	for _, rendered := range want {
		total += utf8.RuneCountInString(rendered)
	}
	if total != maxReleaseNotesChars {
		t.Fatalf("the fixture spends %d content runes, want exactly the %d-rune budget", total, maxReleaseNotesChars)
	}

	got := SummarizeReleaseNotes(body, 0)

	if !slices.Equal(got, want) {
		t.Fatalf("SummarizeReleaseNotes(at the character budget) = %#v, want %#v", got, want)
	}
	if slices.Contains(got, "…") {
		t.Fatalf("summary = %#v, want no ellipsis: the content fit the budget exactly and the separators added nothing", got)
	}
}

// TestSummarizeReleaseNotesDropsTheH1Title pins the top-title rule together
// with its bookkeeping consequence: the dropped title is formatting, not
// truncation, so a body that fits entirely must render without an ellipsis even
// though a line was removed.
func TestSummarizeReleaseNotesDropsTheH1Title(t *testing.T) {
	body := "# Changelog\n\n### Added\n\n- `dflow finish` command\n"

	got := SummarizeReleaseNotes(body, 0)

	want := []string{"▸ Added", "• `dflow finish` command"}
	if !slices.Equal(got, want) {
		t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", body, got, want)
	}
	if slices.Contains(got, "…") {
		t.Fatalf("summary = %#v, want no ellipsis: dropping the title is formatting, not truncation", got)
	}
}

// TestSummarizeReleaseNotesDropsTheVersionHeading pins the issue #30 rule on
// its own: a section heading whose text carries a semantic-version token is the
// release's own version heading, so it is dropped exactly like the H1 title
// rather than rendered under the "what's new in vX:" line the caller prints.
//
// The rule is deliberately mechanical — any heading depth, an optional leading
// "v", the version anywhere in the text — so it does not depend on how a
// maintainer words or places the heading. A version-free heading is the control
// that proves the rule keyed on the version token and not on headings in
// general.
func TestSummarizeReleaseNotesDropsTheVersionHeading(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "the curated v0.3.0 heading disappears",
			body: "## 📦 v0.3.0 – Self-Update, Installers & CLI Contracts",
			want: []string{},
		},
		{
			name: "a bare version without the leading v also disappears",
			body: "## 0.3.0",
			want: []string{},
		},
		{
			name: "a deeper heading carrying a version disappears too",
			body: "### Fixed in v0.4.1",
			want: []string{},
		},
		{
			name: "a version-free heading keeps its marker",
			body: "### Added",
			want: []string{"▸ Added"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body, 0)

			if !slices.Equal(got, tc.want) {
				t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", tc.body, got, tc.want)
			}
			// Dropping a heading removes formatting, not content: the body was not
			// blank, so an all-dropped body is an empty, non-nil summary.
			if got == nil {
				t.Fatalf("SummarizeReleaseNotes(%q) = nil, want an empty, non-nil summary: the body was not blank", tc.body)
			}
		})
	}
}

// TestSummarizeReleaseNotesDropsTheVersionHeadingWithoutAnEllipsis pins the
// bookkeeping consequence of the version-heading rule: like the top title, the
// heading is formatting rather than content, so dropping it must not make a
// complete summary look truncated.
func TestSummarizeReleaseNotesDropsTheVersionHeadingWithoutAnEllipsis(t *testing.T) {
	body := "## 📦 v0.3.0 – Self-Update, Installers & CLI Contracts\n\n### Added\n\n- a short item\n"

	got := SummarizeReleaseNotes(body, 0)

	want := []string{"▸ Added", "• a short item"}
	if !slices.Equal(got, want) {
		t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", body, got, want)
	}
	if slices.Contains(got, "…") {
		t.Fatalf("summary = %#v, want no ellipsis: dropping the version heading is formatting, not truncation", got)
	}
}

// TestSummarizeReleaseNotesKeepsNilReservedForBlankBodies pins what an
// all-formatting body returns. Rendering removes the only line, but the body was
// not blank, so nil keeps meaning exactly "blank body" and callers can tell the
// two cases apart.
func TestSummarizeReleaseNotesKeepsNilReservedForBlankBodies(t *testing.T) {
	got := SummarizeReleaseNotes("# Changelog", 0)

	if got == nil {
		t.Fatal("SummarizeReleaseNotes(\"# Changelog\") = nil, want an empty, non-nil summary")
	}
	if len(got) != 0 {
		t.Fatalf("SummarizeReleaseNotes(\"# Changelog\") = %#v, want no lines", got)
	}
}

// TestSummarizeReleaseNotesDistinguishesBlankFromNothingRenderable makes the
// load-bearing nil rule legible in one place: nil means exactly "blank body",
// while a non-blank body whose every line was formatting returns an empty,
// non-nil slice. The two are different answers — the first says there are no
// release notes, the second says there was a body with nothing to show — so this
// test checks nil-ness itself rather than comparing lengths, which would let a
// nil slice and an empty slice pass as the same thing.
//
// The last case is the one the existing tests did not reach: a blank-looking
// body whose whitespace-only lines surround a real title. Those blank runs are
// skipped, but the title is non-blank content, so the body is not blank and the
// answer must stay non-nil.
func TestSummarizeReleaseNotesDistinguishesBlankFromNothingRenderable(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantNil bool
	}{
		{name: "genuinely empty body", body: "", wantNil: true},
		{name: "whitespace-only body", body: " \n\t\n   \n", wantNil: true},
		{name: "body that is only the title line", body: "# Changelog", wantNil: false},
		{
			name:    "title wrapped in blank runs is still a non-blank body",
			body:    "\n   \n# Changelog\n\t\n\n",
			wantNil: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body, 0)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want nil: a blank body has no release notes", tc.body, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("SummarizeReleaseNotes(%q) = nil, want an empty, non-nil summary: the body was not blank", tc.body)
			}
			if len(got) != 0 {
				t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want no renderable lines", tc.body, got)
			}
		})
	}
}

// TestSummarizeReleaseNotesWrapsBulletsAtWordBoundaries pins the unified text
// measure that replaced the item clamp: a bullet wider than the content width
// wraps at word boundaries instead of being cut, the first physical line keeps
// the "• " marker, and each continuation is indented by the marker's width so it
// hangs under the item's text.
//
// The overlong-token case is the one documented exception, shared with the icon
// wrapper: a single token wider than the width is placed on its own line and
// overflows rather than being split, because a URL or a hash is one value.
func TestSummarizeReleaseNotesWrapsBulletsAtWordBoundaries(t *testing.T) {
	cases := []struct {
		name         string
		body         string
		contentWidth int
		want         []string
	}{
		{
			name:         "a bullet wider than the width wraps at a word boundary",
			body:         "- alpha beta gamma delta epsilon",
			contentWidth: 20,
			want:         []string{"• alpha beta gamma", "  delta epsilon"},
		},
		{
			name:         "a nested bullet's continuations keep the nested indent",
			body:         "  - alpha beta gamma delta epsilon",
			contentWidth: 20,
			want:         []string{"  • alpha beta gamma", "    delta epsilon"},
		},
		{
			name:         "a bullet that fits stays on one line",
			body:         "- alpha beta",
			contentWidth: 20,
			want:         []string{"• alpha beta"},
		},
		{
			name:         "an overlong token overflows rather than splitting",
			body:         "- supercalifragilisticexpialidocious",
			contentWidth: 10,
			want:         []string{"• supercalifragilisticexpialidocious"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body, tc.contentWidth)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("SummarizeReleaseNotes(%q, %d) = %#v, want %#v", tc.body, tc.contentWidth, got, tc.want)
			}
		})
	}
}

// TestSummarizeReleaseNotesKeepsEveryWrappedBulletWhole pins the other half of
// the wrap contract: wrapping re-flows the item's text and never clips it. The
// rebuilt text is the original item word for word, and no ellipsis appears,
// because a complete digest must never look truncated.
func TestSummarizeReleaseNotesKeepsEveryWrappedBulletWhole(t *testing.T) {
	const item = "alpha beta gamma delta epsilon zeta eta theta"

	got := SummarizeReleaseNotes("- "+item, 20)

	if slices.Contains(got, releaseNotesEllipsis) {
		t.Fatalf("summary = %#v, want no ellipsis: wrapping loses no text", got)
	}
	if len(got) < 2 {
		t.Fatalf("summary = %#v, want the item wrapped across several physical lines", got)
	}
	if !strings.HasPrefix(got[0], releaseNotesBulletMarker) {
		t.Fatalf("first line = %q, want the bullet marker", got[0])
	}

	// Rebuild the item from its physical lines: strip the marker from the first
	// line and the two-space continuation indent from the rest, then join.
	words := make([]string, 0, len(got))
	for i, line := range got {
		if i == 0 {
			words = append(words, strings.TrimPrefix(line, releaseNotesBulletMarker))
			continue
		}
		words = append(words, strings.TrimPrefix(line, "  "))
	}
	if rebuilt := strings.Join(words, " "); rebuilt != item {
		t.Fatalf("rebuilt item = %q, want the original %q from %#v", rebuilt, item, got)
	}
}

// TestSummarizeReleaseNotesDoesNotWrapWithoutAMeasuredWidth pins the unmeasured
// contract callers rely on for piped and scripted output: a content width of
// zero or less means "no measurement", so every line is rendered whole, however
// long, with no continuation lines and no ellipsis.
func TestSummarizeReleaseNotesDoesNotWrapWithoutAMeasuredWidth(t *testing.T) {
	longItem := strings.TrimSpace(strings.Repeat("legibility ", 60))
	longPlain := strings.TrimSpace(strings.Repeat("prose ", 60))
	body := "- " + longItem + "\n" + longPlain

	for _, width := range []int{0, -1} {
		t.Run(fmt.Sprintf("width %d", width), func(t *testing.T) {
			got := SummarizeReleaseNotes(body, width)

			want := []string{releaseNotesBulletMarker + longItem, "", longPlain}
			if !slices.Equal(got, want) {
				t.Fatalf("SummarizeReleaseNotes(body, %d) = %#v, want the two whole lines %#v, spaced like any other pair of kept lines", width, got, want)
			}
			if slices.Contains(got, releaseNotesEllipsis) {
				t.Fatalf("summary = %#v, want no ellipsis: an unmeasured width is no reason to truncate", got)
			}
		})
	}
}

// TestSummarizeReleaseNotesLeavesHeadingsUnwrapped pins the deliberate boundary
// of the wrap rule: only bullets and plain lines wrap. A heading is a short
// title rather than prose, and a "▸ " marker has no second line to hang from, so
// an overlong heading is left intact and overflows its line exactly as an
// overlong token does in the icon wrapper.
func TestSummarizeReleaseNotesLeavesHeadingsUnwrapped(t *testing.T) {
	got := SummarizeReleaseNotes("## alpha beta gamma delta", 10)

	want := []string{releaseNotesHeadingMarker + "alpha beta gamma delta"}
	if !slices.Equal(got, want) {
		t.Fatalf("SummarizeReleaseNotes(heading, 10) = %#v, want the heading left unwrapped %#v", got, want)
	}
}

// TestSummarizeReleaseNotesDecidesHeadingClassFromTheRawLine pins the structure
// the spacing rule needs now that any kept line can be followed by content: the
// class is decided from the raw text, never by sniffing the rendered output for
// the heading marker. The rule that exposes the class is the heading exception —
// a line directly under a heading takes no separating blank — so a plain line
// that merely looks like a heading must still be followed by one, while a real
// heading must weld the line beneath it.
func TestSummarizeReleaseNotesDecidesHeadingClassFromTheRawLine(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "a real heading welds the line under it",
			body: "## Added\nintro",
			want: []string{"▸ Added", "intro"},
		},
		{
			name: "a plain line that looks like a marker is not a heading",
			body: "▸ literal text\nintro",
			want: []string{"▸ literal text", "", "intro"},
		},
		{
			name: "a bullet whose text starts with the marker is still a bullet",
			body: "- ▸ literal text\nintro",
			want: []string{"• ▸ literal text", "", "intro"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SummarizeReleaseNotes(tc.body, 0)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("SummarizeReleaseNotes(%q) = %#v, want %#v", tc.body, got, tc.want)
			}
		})
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
	if got := SummarizeReleaseNotes(fits, 0); len(got) != 1 || got[0] != wantFits {
		t.Fatalf("a heading rendering to exactly the cap = %#v, want it kept as %q", got, wantFits)
	}

	clips := "## " + strings.Repeat("x", maxReleaseNotesChars-1)
	if got := SummarizeReleaseNotes(clips, 0); len(got) != 1 || got[0] != "…" {
		t.Fatalf("a heading rendering past the cap = %#v, want [\"…\"]", got)
	}
}

// TestSummarizeReleaseNotesCapsTheCharacterBudget pins the character bound — the
// only flood guard left — together with the ellipsis line that announces it:
// rendered lines are added until the next one would exceed the budget, at which
// point it and the rest are dropped behind a single ellipsis line. The 200-rune
// lines keep the arithmetic exact, so the budget is what fires.
//
// The surviving content is read back out of the summary with the separators
// removed: where the blanks sit is the spacing rule's contract, pinned by
// TestSummarizeReleaseNotesSpacesContentWithBlankLines, while this test is about
// how much content survives.
func TestSummarizeReleaseNotesCapsTheCharacterBudget(t *testing.T) {
	const lineLen = 200
	fit := maxReleaseNotesChars / lineLen
	line := strings.Repeat("x", lineLen)

	lines := make([]string, 0, fit+1)
	for i := 0; i <= fit; i++ {
		lines = append(lines, line)
	}

	got := SummarizeReleaseNotes(strings.Join(lines, "\n"), 0)

	content := make([]string, 0, len(got))
	for _, rendered := range got {
		if rendered != releaseNotesSeparator {
			content = append(content, rendered)
		}
	}

	if len(content) != fit+1 {
		t.Fatalf("content = %#v, want %d lines (%d full lines plus the ellipsis)", content, fit+1, fit)
	}
	for i := 0; i < fit; i++ {
		if content[i] != line {
			t.Fatalf("line %d = %q, want the full %d-rune line", i, content[i], lineLen)
		}
	}
	if content[fit] != releaseNotesEllipsis {
		t.Fatalf("final line = %q, want the ellipsis %q", content[fit], releaseNotesEllipsis)
	}
	// The ellipsis continues the final block instead of opening one, so it is not
	// spaced from the line above it.
	if last := got[len(got)-2]; last == releaseNotesSeparator {
		t.Fatalf("summary = %#v, want the ellipsis appended directly to the last kept line, not spaced from it", got)
	}
}

// TestSummarizeReleaseNotesReportsAClippedOversizedLine records that an
// over-budget single line is not silently discarded: the body is non-blank, so
// the summary is the ellipsis line rather than nil.
func TestSummarizeReleaseNotesReportsAClippedOversizedLine(t *testing.T) {
	got := SummarizeReleaseNotes(strings.Repeat("x", maxReleaseNotesChars+1), 0)

	if len(got) != 1 || got[0] != "…" {
		t.Fatalf("SummarizeReleaseNotes(oversized single line) = %#v, want [\"…\"]", got)
	}
}

// TestSummarizeReleaseNotesCountsCharactersNotBytes locks the unit of the
// character budget: a line of 700 two-byte runes is 1400 bytes but only 700
// characters, so it fits and must be kept whole.
func TestSummarizeReleaseNotesCountsCharactersNotBytes(t *testing.T) {
	line := strings.Repeat("é", 700)

	got := SummarizeReleaseNotes(line, 0)

	if len(got) != 1 || got[0] != line {
		t.Fatalf("SummarizeReleaseNotes(700-rune line) = %#v, want it kept whole", got)
	}
}
