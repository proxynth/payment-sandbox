package main

import (
	"fmt"
	"os"

	"proxynth/payment-sandbox/internal/app"
	"proxynth/payment-sandbox/internal/version"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("payment-sandbox %s\n", version.Version)
		return
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
