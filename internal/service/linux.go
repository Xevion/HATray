//go:build linux

package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ha-tray/internal/app"

	"github.com/coreos/go-systemd/daemon"
)

// linuxService implements the Service interface for Linux
// It integrates with systemd and controls the app layer
// according to systemd signals (start, stop, reload)
type linuxService struct {
	logger *slog.Logger
	app    *app.App
}

// NewService creates a new Linux service instance
func NewService(logger *slog.Logger) Service {
	return &linuxService{
		logger: logger,
		app:    app.NewApp(logger),
	}
}

// Run implements the Service interface for Linux
func (s *linuxService) Run() error {
	startTime := time.Now()
	s.logger.Info("starting service", "start_time", startTime.Format(time.RFC3339))

	// Notify systemd that we are starting
	daemon.SdNotify(false, "STATUS=starting\n")

	// Setup signal handling for systemd
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Setup watchdog to systemd
	var watchdog *time.Ticker
	if watchdogUSec, err := daemon.SdWatchdogEnabled(false); err == nil && watchdogUSec > 0 {
		watchdog = time.NewTicker(watchdogUSec / 2)
	}
	defer func() {
		if watchdog != nil {
			watchdog.Stop()
		}
	}()

	// Setup heartbeat to systemd
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Start the service (backgrounded so that the service can still respond to systemd signals, the app layer is still designed for concurrency)
	go func() {
		if err := s.app.Resume(); err != nil {
			s.logger.Error("failed to start (resume) app layer", "error", err)

			// TODO: This has no true error handling, retry mechanism, or timeout mechanism. If this fails, then the service will be stuck in the 'StartPending' state.
		}

		// Notify systemd that we are ready (and running)
		daemon.SdNotify(false, daemon.SdNotifyReady)
		daemon.SdNotify(false, fmt.Sprintf("STATUS=running for %s\n", time.Since(startTime).String()))
	}()

	for {
		select {
		// This is only called if the service is configured with watchdog
		case <-watchdog.C:
			daemon.SdNotify(false, daemon.SdNotifyWatchdog)
		case <-heartbeat.C:
			daemon.SdNotify(false, fmt.Sprintf("STATUS=running for %s\n", time.Since(startTime).String()))
		case sig := <-sigs:
			s.logger.Info("signal received", "signal", sig)

			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				daemon.SdNotify(false, daemon.SdNotifyStopping)
				s.logger.Info("stopping service")

				if err := s.app.Pause(); err != nil {
					s.logger.Error("failed to pause app layer", "error", err)
				}

				return nil // exit the service
			case syscall.SIGHUP:
				s.logger.Info("reloading service")
				daemon.SdNotify(false, daemon.SdNotifyReloading)

				if err := s.app.Reload(); err != nil {
					s.logger.Error("failed to reload app layer", "error", err)
				}

				daemon.SdNotify(false, daemon.SdNotifyReady)
			default:
				s.logger.Warn("unhandled signal", "signal", sig)
			}
		}
	}
}
