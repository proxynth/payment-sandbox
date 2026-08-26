package main

import (
	"flag"
	"fmt"
	"os"

	"proxynth/payment-sandbox/internal/app"
	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/version"
)

func main() {
	flags := flag.NewFlagSet("payment-sandbox", flag.ExitOnError)
	envFile := flags.String("env-file", "", "load environment variables from a dotenv file")
	showVersion := flags.Bool("version", false, "print the application version")
	flags.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: payment-sandbox [--env-file path] [--version]")
		flags.PrintDefaults()
	}
	_ = flags.Parse(os.Args[1:])

	if *envFile != "" {
		if err := config.LoadEnvFile(*envFile); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "load environment file: %v\n", err)
			os.Exit(1)
		}
	}

	if *showVersion {
		fmt.Printf("payment-sandbox %s\n", version.Version)
		return
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
