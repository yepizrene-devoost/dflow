package utils

import (
	"os"
	"testing"

	"golang.org/x/term"
)

// TestStdoutWidthDeclinesToMeasureANonTerminalStdout pins the gate that keeps
// piped and scripted output unwrapped. `go test` runs this process with stdout
// connected to the test harness, never a terminal, so the real probe must
// report ok=false; that is the answer printWithIcon turns into the historical
// single-line rendering.
func TestStdoutWidthDeclinesToMeasureANonTerminalStdout(t *testing.T) {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		t.Skip("stdout is a terminal in this environment; the non-terminal branch cannot be observed here")
	}

	if width, ok := stdoutWidth(); ok {
		t.Fatalf("stdoutWidth() = (%d, true) for a non-terminal stdout, want ok=false", width)
	}
}

// TestTerminalWidthExposesTheStdoutSeam pins the exported view callers outside
// this package measure through (the release-notes digest does, because it is
// built line by line rather than wrapped by one helper): TerminalWidth answers
// exactly what the stdoutWidth seam answers, so a caller sizes its text to the
// stream it writes and gets the same unmeasured signal a pipe produces.
func TestTerminalWidthExposesTheStdoutSeam(t *testing.T) {
	withStdoutWidth(t, 120, true)
	if width, ok := TerminalWidth(); !ok || width != 120 {
		t.Fatalf("TerminalWidth() = (%d, %v), want (120, true)", width, ok)
	}

	withStdoutWidth(t, 0, false)
	if width, ok := TerminalWidth(); ok || width != 0 {
		t.Fatalf("TerminalWidth() = (%d, %v), want (0, false) for an unmeasured stdout", width, ok)
	}
}
