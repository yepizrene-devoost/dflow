package utils

import "fmt"

// Error prints a message with a red cross (❌) prefix.
// Used to display fatal or important errors to the user.
func Error(msg string) {
	fmt.Println("❌", msg)
}

// Info prints a message with an information icon (ℹ️) prefix.
// Used to display general information about the process.
func Info(msg string) {
	fmt.Println("ℹ️", msg)
}

// Success prints a message with a checkmark icon (✅) prefix.
// Used to indicate successful completion of an operation.
func Success(msg string) {
	fmt.Println("✅", msg)
}

// Warn prints a message with a warning icon (⚠️) prefix.
// Used to display non-fatal issues or important alerts.
func Warn(msg string) {
	fmt.Println("⚠️", msg)
}
