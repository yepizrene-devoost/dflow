package utils

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// IsInteractive reports whether both stdin and stdout are terminals.
//
// Commands use it to decide whether an interactive prompt can be answered and
// whether terminal chrome (spinner frames, the banner) is meaningful. Both
// streams must be terminals: a captured stdout or a redirected stdin means a
// prompt cannot be answered and carriage-return updates would be logged
// verbatim.
//
// It deliberately uses term.IsTerminal instead of a character-device stat
// check: /dev/null is a character device, so a stat check would wrongly report a
// terminal for `dflow ... > /dev/null`.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// stdoutWidth reports the width of the terminal the icon helpers render to, in
// columns, and whether that width could be measured at all.
//
// It is a variable rather than a direct call so tests can inject a fixed width
// or an "unknown" answer without a pseudo-terminal, mirroring the lookupEnv
// seam cmd/selfupdate already established. The end-to-end tests are unaffected:
// they run the built binary, where this variable still holds the real probe.
//
// It measures stdout, not stderr: the icon helpers write their lines to stdout,
// so the measured stream and the wrapped stream are the same stream. A stdout
// that is not a terminal — a pipe, a file, a command substitution — must report
// ok=false, and callers must read that as "do not wrap".
var stdoutWidth = terminalWidth

// terminalWidth reads the column count of stdout.
//
// It declines to answer (ok=false) when stdout is not a terminal or the reported
// size is unusable (a width of zero or less), which is exactly the signal to
// leave a line unwrapped rather than guess a width a scripted reader did not
// ask for.
func terminalWidth() (int, bool) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 0, false
	}
	return width, true
}

// TerminalWidth reports the width, in columns, of the terminal the human-facing
// helpers render to, and whether that width could be measured at all.
//
// It is the exported view of the stdoutWidth seam, for callers outside this
// package that must size text to the terminal themselves — the release-notes
// digest does, because it is built line by line rather than wrapped by one
// helper. ok=false means "do not size anything to a width": stdout is a pipe, a
// file or a command substitution, and the caller must render its unmeasured
// output exactly as it always has. A caller that does have a width is still
// responsible for its own floor and ceiling; this function reports the terminal,
// not a content width.
//
// It measures stdout, never stderr, for the same reason the icon helpers do:
// stdout is the stream the sized output is written to and the one a user reads
// in a terminal.
func TerminalWidth() (int, bool) {
	return stdoutWidth()
}

// NonInteractiveError builds the error returned when a command needs to prompt
// but stdin or stdout is not a terminal.
//
// It keeps the message shape consistent across commands and forces every caller
// to name the concrete remedy (a flag or argument) that avoids the prompt.
func NonInteractiveError(action, remedy string) error {
	return fmt.Errorf("cannot %s without an interactive terminal; %s", action, remedy)
}
