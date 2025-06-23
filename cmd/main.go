package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"ha-tray/internal/service"
)

var (
	Version   = "dev"
	Commit    = ""
	BuildDate = ""
)

func main() {
	logger, logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("failed to setup logging: %v", err)
	}
	defer logFile.Sync()
	defer logFile.Close()

	logger.Info("HATray started", "version", Version, "commit", Commit, "built", BuildDate)

	defer func() {
		if r := recover(); r != nil {
			logger.Error("uncaught panic recovered", "panic", r)
		}
	}()

	// Create service layer
	svc := service.NewService(logger)

	logger.Info("service initialized")

	// Main loop
	if err := svc.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}

func setupLogging() (*slog.Logger, *os.File, error) {
	// Get the directory where the executable is located
	exePath, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	// Open log file in the same directory as the executable
	logFile, err := os.OpenFile(filepath.Join(exeDir, "current.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Create multi-writer to log to both file and stdout
	multiWriter := io.MultiWriter(logFile, os.Stdout)

	// Create JSON handler for structured logging
	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	return logger, logFile, nil
}
