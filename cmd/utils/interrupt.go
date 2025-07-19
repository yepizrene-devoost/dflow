package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// HandleInterrupt sets up a listener for OS interrupt signals (e.g. Ctrl+C or SIGTERM)
// and gracefully exits the program with a message when triggered.
func HandleInterrupt() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		println("\n🚫 Execution cancelled by user.")
		os.Exit(1)
	}()
}
