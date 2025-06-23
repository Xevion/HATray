//go:build linux

package service

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ha-tray/internal/app"

	"github.com/coreos/go-systemd/daemon"
)

const serviceName = "HATray"

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
	s.logger.Info("starting service")

	// Notify systemd that we are starting
	daemon.SdNotify(false, "STATUS=Starting HATray...\n")

	// Setup signal handling for systemd
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Setup heartbeat to systemd
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Start the service (backgrounded so that the service can still respond to systemd signals, the app layer is still designed for concurrency)
	go func() {
		if err := s.app.Resume(); err != nil {
			s.logger.Error("failed to start (resume) app layer", "error", err)

			// TODO: This has no true error handling, retry mechanism, or timeout mechanism. If this fails, then the service will be stuck in the 'StartPending' state.
		}

		// Notify systemd that we are ready (and running)
		daemon.SdNotify(false, "READY=1")
		daemon.SdNotify(false, "STATUS=HATray running\n")
	}()

	for {
		select {
		case <-ticker.C:
			daemon.SdNotify(false, "WATCHDOG=1")
			s.logger.Debug("heartbeat") // TODO: add more detailed status information here
		case sig := <-sigs:
			s.logger.Info("signal received", "signal", sig)

			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				daemon.SdNotify(false, "STOPPING=1")
				s.logger.Info("stopping service")

				if err := s.app.Pause(); err != nil {
					s.logger.Error("failed to pause app layer", "error", err)
				}

				return nil // exit the service
			case syscall.SIGHUP:
				s.logger.Info("reloading service")
				daemon.SdNotify(false, "RELOADING=1")

				if err := s.app.Reload(); err != nil {
					s.logger.Error("failed to reload app layer", "error", err)
				}

				daemon.SdNotify(false, "READY=1")
			}
		}
	}
}
