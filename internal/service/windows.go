//go:build windows

package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ha-tray/internal/app"

	"golang.org/x/sys/windows/registry"
)

// WindowsTrayService implements the Service interface for Windows as a user application
type WindowsTrayService struct {
	app          *app.App
	logger       *slog.Logger
	restartCount int
	maxRestarts  int
	restartDelay time.Duration
	quitChan     chan struct{}
	restartChan  chan struct{}
}

// NewService creates a new Windows tray service instance
func NewService(logger *slog.Logger) Service {
	return &WindowsTrayService{
		logger:       logger.With("type", "service", "variant", "windows"),
		app:          app.NewApp(logger),
		maxRestarts:  3,
		restartDelay: 5 * time.Second,
		quitChan:     make(chan struct{}),
		restartChan:  make(chan struct{}),
	}
}

// Run implements the Service interface for Windows
func (svc *WindowsTrayService) Run() error {
	svc.logger.Info("starting Windows tray service")

	// Setup auto-start if not already configured
	if err := svc.setupAutoStart(); err != nil {
		svc.logger.Warn("failed to setup auto-start", "error", err)
	}

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Setup power management (sleep/wake)
	svc.setupPowerManagement()

	// Main service loop with restart capability
	for {
		select {
		case <-svc.quitChan:
			svc.logger.Info("service shutdown requested")
			return nil
		default:
			if err := svc.runServiceLoop(sigs); err != nil {
				svc.logger.Error("service loop failed", "error", err)

				if svc.restartCount < svc.maxRestarts {
					svc.restartCount++
					svc.logger.Info("restarting service", "attempt", svc.restartCount, "max", svc.maxRestarts)
					time.Sleep(svc.restartDelay)
					continue
				} else {
					svc.logger.Error("max restarts exceeded, shutting down")
					return err
				}
			}
		}
	}
}

// runServiceLoop runs the main service loop
func (svc *WindowsTrayService) runServiceLoop(sigs chan os.Signal) error {
	// Start the application in background
	go func() {
		if err := svc.app.Resume(); err != nil {
			svc.logger.Error("failed to start app layer", "error", err)
		}
	}()

	// Service heartbeat
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Watchdog for app health
	watchdog := time.NewTicker(60 * time.Second)
	defer watchdog.Stop()

	for {
		select {
		case <-svc.quitChan:
			svc.logger.Info("shutting down service")
			if err := svc.app.Pause(); err != nil {
				svc.logger.Error("failed to pause app layer", "error", err)
			}
			return nil

		case <-svc.restartChan:
			svc.logger.Info("restarting service")
			if err := svc.app.Reload(); err != nil {
				svc.logger.Error("failed to reload app layer", "error", err)
			}

		case <-heartbeat.C:
			svc.logger.Debug("service heartbeat", "uptime", time.Since(time.Now()))

		case <-watchdog.C:
			// Check if app is healthy
			if !svc.isAppHealthy() {
				svc.logger.Warn("app health check failed, triggering restart")
				svc.restartChan <- struct{}{}
			}

		case sig := <-sigs:
			svc.logger.Info("signal received", "signal", sig)
			close(svc.quitChan)
		}
	}
}

// setupAutoStart configures the application to start automatically on login
func (svc *WindowsTrayService) setupAutoStart() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %v", err)
	}
	defer key.Close()

	// Use quotes around path to handle spaces
	exePath = fmt.Sprintf(`"%s"`, exePath)

	if err := key.SetStringValue("HATray", exePath); err != nil {
		return fmt.Errorf("failed to set registry value: %v", err)
	}

	svc.logger.Info("auto-start configured", "path", exePath)
	return nil
}

// removeAutoStart removes the auto-start configuration
func (svc *WindowsTrayService) removeAutoStart() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %v", err)
	}
	defer key.Close()

	if err := key.DeleteValue("HATray"); err != nil {
		return fmt.Errorf("failed to delete registry value: %v", err)
	}

	svc.logger.Info("auto-start removed")
	return nil
}

// setupPowerManagement handles sleep/wake events
func (svc *WindowsTrayService) setupPowerManagement() {
	// TODO: Implement Windows power management
	// - Listen for WM_POWERBROADCAST messages
	// - Handle system sleep/wake events
	// - Pause/resume app accordingly
	svc.logger.Debug("power management setup (not implemented)")
}

// isAppHealthy checks if the application is running properly
func (svc *WindowsTrayService) isAppHealthy() bool {
	// TODO: Implement health checks
	// - Check if Home Assistant connection is alive
	// - Check if systray is responsive
	// - Check memory usage
	// - Check for any error conditions
	return true
}
