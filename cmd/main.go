package main

import (
	"fmt"
	"log"
	"os"

	"ha-tray/internal"
)

func main() {
	// Create new application instance
	app := internal.NewApp()

	// Setup the application (logging, panic handling, service initialization)
	if err := app.Setup(); err != nil {
		log.Fatalf("Failed to setup application: %v", err)
	}

	// Run the application
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
