package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// HandleInterrupt installs a signal handler for OS interrupts (e.g. Ctrl+C or SIGTERM).
//
// When triggered, it prints a cancellation message and terminates the program
// with exit code 1. This is intended to provide graceful shutdown behavior
// during interactive command-line execution.
func HandleInterrupt() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		println("\n🚫 Execution cancelled by user.")
		os.Exit(1)
	}()
}
