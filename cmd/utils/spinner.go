package utils

import (
	"fmt"
	"sync"
	"time"
)

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// Spinner renders a lightweight terminal spinner for long-running CLI tasks.
//
// When stdout is not a terminal the spinner degrades to plain lines: it cannot
// rely on the carriage return to overwrite a line, so it prints the status
// message once on Start and the final icon and message on Stop without frames or
// ANSI escapes.
type Spinner struct {
	message     string
	done        chan struct{}
	stopOnce    sync.Once
	interactive bool
}

// NewSpinner creates a spinner that displays the provided status message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:     message,
		done:        make(chan struct{}),
		interactive: IsInteractive(),
	}
}

// Start begins rendering the spinner asynchronously until Stop is called.
//
// On a non-interactive stream it prints the status message once and spawns no
// goroutine, so the output stays a single line.
func (s *Spinner) Start() {
	if !s.interactive {
		fmt.Printf("%s\n", s.message)
		return
	}

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				fmt.Printf("\r%s %s", string(spinnerFrames[i]), s.message)
				time.Sleep(100 * time.Millisecond)
				i = (i + 1) % len(spinnerFrames)
			}
		}
	}()
}

// Stop clears the spinner and prints a final status line.
//
// If an icon is provided, it replaces the default success icon. Stop is safe to
// call whether or not Start ran, and it never double-closes the done channel.
func (s *Spinner) Stop(message string, icon ...string) {
	s.stopOnce.Do(func() {
		close(s.done)
	})

	finalIcon := "✅"
	if len(icon) > 0 && icon[0] != "" {
		finalIcon = icon[0]
	}

	if !s.interactive {
		fmt.Printf("%-3s %s\n", finalIcon, message)
		return
	}

	clear := "\r\033[K"
	fmt.Printf("%s%-3s %s\n", clear, finalIcon, message)
}
