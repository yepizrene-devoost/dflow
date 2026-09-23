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

// NonInteractiveError builds the error returned when a command needs to prompt
// but stdin or stdout is not a terminal.
//
// It keeps the message shape consistent across commands and forces every caller
// to name the concrete remedy (a flag or argument) that avoids the prompt.
func NonInteractiveError(action, remedy string) error {
	return fmt.Errorf("cannot %s without an interactive terminal; %s", action, remedy)
}
