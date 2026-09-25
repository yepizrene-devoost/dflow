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
