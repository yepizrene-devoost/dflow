package utils

import (
	"fmt"
	"strings"
	"testing"
)

// withStdoutWidth installs a fixed answer from the terminal-width seam for one
// test and restores the real probe afterwards.
//
// stdoutWidth is the package-level seam printWithIcon reads, exactly like the
// lookupEnv seam cmd/selfupdate established. Driving it here is what lets a
// test pin wrapped rendering without a pseudo-terminal: the width decision is
// the only thing that makes wrapping TTY-conditional, and it must be
// answerable from a plain pipe.
func withStdoutWidth(t *testing.T, width int, ok bool) {
	t.Helper()

	original := stdoutWidth
	stdoutWidth = func() (int, bool) { return width, ok }
	t.Cleanup(func() { stdoutWidth = original })
}

// withStdoutIsTTY installs a fixed answer from the terminal-detection seam
// PlainBold reads for one test and restores the real probe afterwards.
//
// stdoutIsTTY is the package-level seam that makes the bold heading conditional
// on a terminal together with the shared JSON suppression. Driving it here is
// what lets a test pin the styled rendering — and its byte-identical unstyled
// twin — without a pseudo-terminal.
func withStdoutIsTTY(t *testing.T, isTTY bool) {
	t.Helper()

	original := stdoutIsTTY
	stdoutIsTTY = func() bool { return isTTY }
	t.Cleanup(func() { stdoutIsTTY = original })
}

// longWarning is a single-line message wider than the 80-column floor's content
// width (76 runes). Its word lengths are chosen so the greedy fill breaks at a
// known point, and it is real prose rather than repeated filler so a failure
// reads as the sentence it is.
const longWarning = "the provenance warning must wrap at word boundaries and never split a word in the middle"

// TestWrapIconMessageBreaksAtWordBoundaries pins the greedy fill: a line never
// exceeds the content width unless it is a single overlong token, and every
// break replaces a space rather than landing inside a word.
func TestWrapIconMessageBreaksAtWordBoundaries(t *testing.T) {
	cases := []struct {
		name         string
		message      string
		contentWidth int
		want         []string
	}{
		{
			name:         "a message that fits stays on one line",
			message:      "one two",
			contentWidth: 7,
			want:         []string{"one two"},
		},
		{
			name:         "a break replaces the space at the boundary",
			message:      "alpha beta gamma",
			contentWidth: 10,
			want:         []string{"alpha beta", "gamma"},
		},
		{
			name:         "a word exactly at the width is not split",
			message:      "alphabet soup",
			contentWidth: 8,
			want:         []string{"alphabet", "soup"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wrapIconMessage(tc.message, tc.contentWidth)
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("wrapIconMessage(%q, %d) = %q, want %q", tc.message, tc.contentWidth, got, tc.want)
			}
			for _, line := range got {
				if line == "" {
					t.Fatalf("wrapIconMessage(%q, %d) produced an empty line in %q", tc.message, tc.contentWidth, got)
				}
			}
		})
	}
}

// TestWrapIconMessageLeavesAnOverlongTokenIntact pins the documented exception:
// a single word wider than the content width is not broken. A URL or a hash is
// one value, and cutting it in half would corrupt it; the terminal's own edge
// wrapping is the lesser evil.
func TestWrapIconMessageLeavesAnOverlongTokenIntact(t *testing.T) {
	const token = "https://example.com/releases/download/v9.9.9/dflow_9.9.9_darwin_arm64.tar.gz"

	got := wrapIconMessage("see "+token+" for details", 20)
	want := []string{"see", token, "for details"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("wrapIconMessage with an overlong token = %q, want %q", got, want)
	}
	for _, line := range got {
		if len(line) > 20 && line != token {
			t.Fatalf("only the overlong token may exceed the width, got %q", line)
		}
	}
}

// TestPrintWithIconHangsContinuationLinesUnderTheContentColumn is the rendering
// contract for a wrapped line: the first physical line keeps the
// "icon + space + message" shape, and every continuation line is indented by
// four spaces so it hangs under the message column instead of under the icon.
func TestPrintWithIconHangsContinuationLinesUnderTheContentColumn(t *testing.T) {
	withStdoutWidth(t, 80, true)

	output := captureStdout(t, func() { Warn("%s", longWarning) })

	want := "⚠️  the provenance warning must wrap at word boundaries and never split a word\n" +
		"    in the middle\n"
	if output != want {
		t.Fatalf("wrapped Warn output =\n%q\nwant\n%q", output, want)
	}
}

// TestPrintWithIconLeavesTheLineWholeWhenTheWidthIsUnknown pins the TTY gate: a
// pipe or a redirected file is never wrapped, so a machine that reads the
// output can copy a long line verbatim.
func TestPrintWithIconLeavesTheLineWholeWhenTheWidthIsUnknown(t *testing.T) {
	withStdoutWidth(t, 0, false)

	output := captureStdout(t, func() { Error("%s", longWarning) })

	want := fmt.Sprintf("%-3s %s\n", "❌", longWarning)
	if output != want {
		t.Fatalf("unmeasured width output =\n%q\nwant the single historical line\n%q", output, want)
	}
	if strings.Contains(output, "\n    ") {
		t.Fatalf("an unknown width must not add a hanging-indent continuation line:\n%q", output)
	}
}

// TestPrintWithIconCapsTheWrapWidthAtOneHundredColumns pins the ceiling that
// keeps icon lines and the release-notes digest on one common measure: a
// terminal far wider than the cap still wraps at maxWrapWidth content columns,
// so a 200-column terminal does not produce 196-column lines while the digest
// wraps at 100. Without the cap the message below fits on one 179-rune line.
func TestPrintWithIconCapsTheWrapWidthAtOneHundredColumns(t *testing.T) {
	// Thirty "alpha" words: 179 runes, wider than the 100-column cap but well
	// under the 196 content columns an uncapped 200-column terminal would offer.
	message := strings.TrimSpace(strings.Repeat("alpha ", 30))

	withStdoutWidth(t, 200, true)
	output := captureStdout(t, func() { Warn("%s", message) })

	// The greedy fill takes sixteen words at 95 runes and cannot add a
	// seventeenth (101), so the remaining fourteen take the continuation line.
	firstLine := strings.TrimSpace(strings.Repeat("alpha ", 16))
	secondLine := strings.TrimSpace(strings.Repeat("alpha ", 14))
	want := fmt.Sprintf("%-3s %s\n%s%s\n", "⚠️", firstLine, continuationIndent, secondLine)
	if output != want {
		t.Fatalf("200-column output =\n%q\nwant the wrap at the %d-column ceiling\n%q", output, maxWrapWidth, want)
	}
}

// TestPrintWithIconTreatsANarrowTerminalAsEightyColumnsWide pins the floor: a
// terminal narrower than the floor wraps exactly as an 80-column one does, so
// the floor is a wrapping width and not an opt-out from wrapping.
func TestPrintWithIconTreatsANarrowTerminalAsEightyColumnsWide(t *testing.T) {
	withStdoutWidth(t, 80, true)
	want := captureStdout(t, func() { Warn("%s", longWarning) })

	withStdoutWidth(t, 40, true)
	got := captureStdout(t, func() { Warn("%s", longWarning) })

	if got != want {
		t.Fatalf("a 40-column terminal wrapped differently from an 80-column one:\ngot:\n%q\nwant:\n%q", got, want)
	}
	if !strings.Contains(got, "\n    ") {
		t.Fatalf("the 40-column case must still wrap, got:\n%q", got)
	}
}

// TestPrintWithIconKeepsAShortMessageByteIdentical protects every existing icon
// line: a message that fits on one line is rendered exactly as the historical
// single Printf did, so the wrap changes nothing outside the wrap.
func TestPrintWithIconKeepsAShortMessageByteIdentical(t *testing.T) {
	withStdoutWidth(t, 80, true)

	output := captureStdout(t, func() { Success("%s", "dflow is already up to date") })

	want := fmt.Sprintf("%-3s %s\n", "✅", "dflow is already up to date")
	if output != want {
		t.Fatalf("short-message output = %q, want the unchanged single line %q", output, want)
	}
}

// TestPlainBoldBoldsTheHeadingOnATerminal pins the styling this work unit adds:
// on a terminal a section heading of the update digest is wrapped in the ANSI
// bold sequence, so its title anchors the airy block beneath it.
func TestPlainBoldBoldsTheHeadingOnATerminal(t *testing.T) {
	withStdoutIsTTY(t, true)

	output := captureStdout(t, func() { PlainBold("  %s", "▸ Added") })

	want := "\033[1m  ▸ Added\033[0m\n"
	if output != want {
		t.Fatalf("PlainBold on a terminal = %q, want %q", output, want)
	}
}

// TestPlainBoldIsByteIdenticalToPlainOffATerminal pins the other half of the
// gate: when stdout is not a terminal — a pipe, a file or a command substitution
// — the line is emitted exactly as Plain emits it, so copied and scripted output
// carries no escape codes and stays easy to paste verbatim.
func TestPlainBoldIsByteIdenticalToPlainOffATerminal(t *testing.T) {
	withStdoutIsTTY(t, false)

	styled := captureStdout(t, func() { PlainBold("  %s", "▸ Added") })
	plain := captureStdout(t, func() { Plain("  %s", "▸ Added") })

	if styled != plain {
		t.Fatalf("PlainBold off a terminal = %q, want Plain's byte-identical %q", styled, plain)
	}
	if strings.Contains(styled, "\033") {
		t.Fatalf("off a terminal PlainBold emitted an escape code: %q", styled)
	}
}

// TestPlainBoldYieldsToJSONDocuments pins the shared suppression every output
// helper honours: when stdout carries a machine-readable document, a styled
// heading is nothing but noise, so PlainBold writes nothing at all — exactly
// like Plain, and regardless of the terminal gate.
func TestPlainBoldYieldsToJSONDocuments(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatHuman) })
	withStdoutIsTTY(t, true)
	SetFormat(FormatJSON)

	output := captureStdout(t, func() { PlainBold("  %s", "▸ Added") })

	if output != "" {
		t.Fatalf("PlainBold in JSON mode = %q, want nothing", output)
	}
}
