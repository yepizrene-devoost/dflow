package utils

import "fmt"

func Error(msg string) {
	fmt.Println("❌", msg)
}

func Info(msg string) {
	fmt.Println("ℹ️", msg)
}

func Success(msg string) {
	fmt.Println("✅", msg)
}

func Warn(msg string) {
	fmt.Println("⚠️", msg)
}
