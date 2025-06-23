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
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)
	done := make(chan struct{})

	// Notify systemd that we are ready
	daemon.SdNotify(false, "READY=1")
	daemon.SdNotify(false, "STATUS=HATray running\n")

	go func() {
		for {
			sig := <-sigs
			s.logger.Info("signal received", "signal", sig)
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				daemon.SdNotify(false, "STOPPING=1")
				s.logger.Info("stopping service")
				s.app.Stop()
				close(done)
				return
			case syscall.SIGHUP:
				s.logger.Info("reloading service")
				daemon.SdNotify(false, "RELOADING=1")
				s.app.Reload()
				daemon.SdNotify(false, "READY=1")
			case syscall.SIGUSR1:
				s.logger.Info("pausing service")
				s.app.Pause()
			case syscall.SIGUSR2:
				s.logger.Info("resuming service")
				s.app.Resume()
			}
		}
	}()

	// Main loop: heartbeat to systemd
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			s.logger.Info("service stopped")
			return nil
		case <-ticker.C:
			daemon.SdNotify(false, "WATCHDOG=1")
			s.logger.Debug("heartbeat")
		}
	}
}
