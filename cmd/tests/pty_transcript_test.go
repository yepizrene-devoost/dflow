package tests

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// This file pins the harness itself: that the interactive output capture keeps
// every byte the child wrote, including the last one, no matter how slowly the
// consumer works.
//
// The bug it pins is the one that made
// TestInitCompletesOnboardingWithoutOriginRemote fail on ubuntu-latest while
// passing on macOS: `dflow init` exited 0 and the transcript stopped right after
// the last prompt, because the harness drained cmd.StdoutPipe() by hand while
// cmd.Wait() ran concurrently, and Wait closes that pipe the moment the process
// exits. Anything the reader had not taken yet is discarded silently -- and the
// reader is deliberately not taking anything while it paces a reply, which is
// exactly the window in which a fast command finishes.

// pinFinalChunkDelay is how long the pin's child pauses between its first chunk
// and its final one. The child writes the final chunk and exits inside this
// window, so the harness is holding the first chunk when the process goes away.
const pinFinalChunkDelay = 50 * time.Millisecond

// pinSinkPacing is how long the harness stays busy with each chunk it receives.
// It is forced to be far longer than pinFinalChunkDelay: the child's whole
// remaining lifetime (pinFinalChunkDelay plus the write and the exit) always
// falls inside the first chunk's pacing window.
//
// This is what makes the pin deterministic instead of racy. It never asks "did
// the reader happen to be busy when the process exited"; it makes the reader busy
// for 400 ms while the child finishes in about 50 ms, so the loss window is open
// by construction and the only question left is whether the capture mechanism can
// lose bytes inside it. A hand-drained StdoutPipe answers "yes" -- Wait closes the
// pipe mid-window and the final chunk is gone with a "file already closed" read
// error -- and a Writer assigned to cmd.Stdout answers "no", because os/exec runs
// the copy itself and Wait waits for that copy to return before it returns.
const pinSinkPacing = 400 * time.Millisecond

// TestCapturedCLIKeepsOutputWrittenBeforeExit is the regression pin for the
// harness's output capture: a child that writes a final block and exits while the
// consumer is deliberately still busy with the previous chunk must have its
// complete output in the transcript, final line included.
//
// It exercises the same code path the interactive pins use (runCapturedCLI plus
// terminalDriver), but with a plain child instead of a pseudo-terminal, so it
// needs no script(1) and no prompt to answer. The child is written with printf
// and sleep only, so it behaves identically on macOS and Linux.
func TestCapturedCLIKeepsOutputWrittenBeforeExit(t *testing.T) {
	// The child writes a chunk, pauses for pinFinalChunkDelay, then writes its
	// final chunk and exits -- two separate writes, so the harness sees two chunks
	// and the second one is written while the harness is still busy with the first.
	child := fmt.Sprintf("printf 'first chunk\\n'; sleep %v; printf 'final chunk\\n'", pinFinalChunkDelay.Seconds())

	transcript, exitCode := runCapturedCLI(
		t,
		30*time.Second,
		exec.Command("sh", "-c", child),
		nil,
		pinSinkPacing,
	)

	if exitCode != 0 {
		t.Fatalf("the pin's child exited %d, want 0\ntranscript:\n%s", exitCode, transcript)
	}
	// Exact bytes, not a substring: the whole transcript must survive, in order,
	// with the final chunk last and nothing else invented on the way.
	if want := "first chunk\nfinal chunk\n"; transcript != want {
		t.Fatalf("the harness lost output the child wrote before it exited:\n--- want ---\n%q\n--- got ---\n%q", want, transcript)
	}
	if !strings.HasSuffix(transcript, "final chunk\n") {
		t.Fatalf("the transcript does not end on the child's final line:\n%s", transcript)
	}
}
