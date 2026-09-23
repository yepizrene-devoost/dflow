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
	finalIcon := "✅"
	if len(icon) > 0 && icon[0] != "" {
		finalIcon = icon[0]
	}

	s.terminate(func() {
		if !s.interactive {
			fmt.Printf("%-3s %s\n", finalIcon, message)
			return
		}

		clear := "\r\033[K"
		fmt.Printf("%s%-3s %s\n", clear, finalIcon, message)
	})
}

// Clear stops the spinner and erases its line without printing a status line.
//
// Use it on failure paths: the failure is reported by the returned error and
// rendered once by the shared output path, so a status line here would either
// duplicate the failure or announce it with the success icon.
func (s *Spinner) Clear() {
	s.terminate(func() {
		// Non-interactive Start printed the status as a plain log line, so there
		// is no spinner line to erase and nothing to add.
		if !s.interactive {
			return
		}

		fmt.Printf("\r\033[K")
	})
}

// terminate renders the spinner's final state exactly once, on the first Stop
// or Clear call, after closing done so the interactive goroutine exits. Sharing
// one guard makes both terminators idempotent and safe to combine, and safe to
// call whether or not Start ran.
func (s *Spinner) terminate(render func()) {
	s.stopOnce.Do(func() {
		close(s.done)
		render()
	})
}
