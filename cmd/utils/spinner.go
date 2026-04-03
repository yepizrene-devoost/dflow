package utils

import (
	"fmt"
	"time"
)

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// Spinner renders a lightweight terminal spinner for long-running CLI tasks.
type Spinner struct {
	message string
	done    chan struct{}
}

// NewSpinner creates a spinner that displays the provided status message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan struct{}),
	}
}

// Start begins rendering the spinner asynchronously until Stop is called.
func (s *Spinner) Start() {
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
// If an icon is provided, it replaces the default success icon.
func (s *Spinner) Stop(message string, icon ...string) {
	close(s.done)

	finalIcon := "✅"
	if len(icon) > 0 && icon[0] != "" {
		finalIcon = icon[0]
	}

	clear := "\r\033[K"
	fmt.Printf("%s%-3s %s\n", clear, finalIcon, message)
}
