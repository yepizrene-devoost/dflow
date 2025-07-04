package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// HandleInterrupt imprime un mensaje y cancela el proceso cuando se presiona Ctrl+C
func HandleInterrupt() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		println("\n🚫 Execution cancelled by user.")
		os.Exit(1)
	}()
}
