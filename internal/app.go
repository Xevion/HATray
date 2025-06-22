package internal

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"ha-tray/internal/app"
	"ha-tray/internal/service"
)

// App represents the main application
type App struct {
	appLayer *app.App
	service  service.Service
	logger   *slog.Logger
	logFile  *os.File
}

// NewApp creates a new application instance
func NewApp() *App {
	return &App{}
}

// Setup initializes the application with logging and panic handling
func (a *App) Setup() error {
	// Setup panic recovery
	defer a.recoverPanic()

	// Setup logging
	if err := a.setupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %v", err)
	}

	// Setup app layer
	a.appLayer = app.NewApp(a.logger)

	// Setup service
	if err := a.setupService(); err != nil {
		return fmt.Errorf("failed to setup service: %v", err)
	}

	return nil
}

// Run starts the application
func (a *App) Run() error {
	defer a.cleanup()

	a.logger.Info("Starting HATray application")

	// Run the service
	return a.service.Run()
}

// setupLogging initializes structured logging
func (a *App) setupLogging() error {
	// Get the directory where the executable is located
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	// Open log file in the same directory as the executable
	logFile, err := os.OpenFile(filepath.Join(exeDir, "current.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	a.logFile = logFile

	// Create multi-writer to log to both file and stdout
	multiWriter := io.MultiWriter(logFile, os.Stdout)

	// Create JSON handler for structured logging
	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	a.logger = slog.New(handler)

	return nil
}

// setupService initializes the platform-specific service
func (a *App) setupService() error {
	// Platform-specific service initialization using build flags
	a.service = service.NewService(a.logger, a.appLayer)
	return nil
}

// recoverPanic handles panic recovery and logs the error
func (a *App) recoverPanic() {
	if r := recover(); r != nil {
		if a.logger != nil {
			a.logger.Error("Panic recovered", "panic", r)
		} else {
			fmt.Printf("Panic recovered: %v\n", r)
		}
	}
}

// cleanup performs cleanup operations
func (a *App) cleanup() {
	if a.logFile != nil {
		a.logFile.Close()
	}
}
