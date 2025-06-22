//go:build windows

package service

import (
	"fmt"
	"log/slog"
	"time"

	"ha-tray/internal/app"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

const serviceName = "HATray"

// WindowsService implements the Service interface for Windows
type WindowsService struct {
	logger  *slog.Logger
	elog    debug.Log
	isDebug bool
	app     *app.App
}

// newService creates a new Windows service instance
func newService(logger *slog.Logger, appLayer *app.App) Service {
	return &WindowsService{
		logger: logger,
		app:    appLayer,
	}
}

// Run implements the Service interface for Windows
func (w *WindowsService) Run() error {
	// Determine if we're running as a Windows service
	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to determine if running as Windows service: %v", err)
	}

	w.isDebug = !isService

	if isService {
		return w.runAsService()
	}

	// Interactive mode
	return w.runInteractive()
}

// runAsService runs the application as a Windows service
func (w *WindowsService) runAsService() error {
	var err error
	if w.isDebug {
		w.elog = debug.New(serviceName)
	} else {
		w.elog, err = eventlog.Open(serviceName)
		if err != nil {
			return fmt.Errorf("failed to open event log: %v", err)
		}
	}
	defer w.elog.Close()

	w.elog.Info(1, fmt.Sprintf("starting %s service", serviceName))

	run := svc.Run
	if w.isDebug {
		run = debug.Run
	}

	err = run(serviceName, &windowsServiceHandler{
		service: w,
	})

	if err != nil {
		w.elog.Error(1, fmt.Sprintf("%s service failed: %v", serviceName, err))
		return err
	}

	w.elog.Info(1, fmt.Sprintf("%s service stopped", serviceName))
	return nil
}

// runInteractive runs the application in interactive mode
func (w *WindowsService) runInteractive() error {
	w.logger.Info("Application starting in interactive mode")

	// Simple interactive loop
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		w.logger.Debug("Application heartbeat")
	}

	return nil
}

// windowsServiceHandler implements the Windows service handler interface
type windowsServiceHandler struct {
	service *WindowsService
}

func (h *windowsServiceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	h.service.logger.Info("Service starting")
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Main service loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				h.service.logger.Info("Service stopping")
				changes <- svc.Status{State: svc.StopPending}
				if err := h.service.app.Stop(); err != nil {
					h.service.logger.Error("Failed to stop app layer", "error", err)
				}
				return
			case svc.Pause:
				h.service.logger.Info("Service pausing")
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				if err := h.service.app.Pause(); err != nil {
					h.service.logger.Error("Failed to pause app layer", "error", err)
				}
			case svc.Continue:
				h.service.logger.Info("Service continuing")
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				if err := h.service.app.Resume(); err != nil {
					h.service.logger.Error("Failed to resume app layer", "error", err)
				}
			default:
				h.service.elog.Error(uint32(1), fmt.Sprintf("unexpected control request #%d", c))
			}
		case <-ticker.C:
			h.service.logger.Debug("Service heartbeat")
		}
	}
}
