//go:build windows

package service

import (
	"fmt"
	"log/slog"
	"time"

	"ha-tray/internal/app"

	winsvc "golang.org/x/sys/windows/svc"
	winsvcDebug "golang.org/x/sys/windows/svc/debug"
	winsvcEventlog "golang.org/x/sys/windows/svc/eventlog"
)

const serviceName = "HATray"

// WindowsService implements the Service interface for Windows
type WindowsService struct {
	logger  *slog.Logger
	elog    winsvcDebug.Log
	isDebug bool
	app     *app.App
}

// newService creates a new Windows service instance
func NewService(logger *slog.Logger) Service {
	return &WindowsService{
		logger: logger,
		app:    app.NewApp(logger),
	}
}

// Run implements the Service interface for Windows
func (svc *WindowsService) Run() error {
	// Determine if we're running as a Windows service
	isService, err := winsvc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to determine if running as Windows service: %v", err)
	}

	svc.isDebug = !isService

	if isService {
		return svc.runAsService()
	}

	// Interactive mode
	return svc.runInteractive()
}

// runAsService runs the application as a Windows service
func (svc *WindowsService) runAsService() error {
	var err error
	if svc.isDebug {
		svc.elog = winsvcDebug.New(serviceName)
	} else {
		svc.elog, err = winsvcEventlog.Open(serviceName)
		if err != nil {
			return fmt.Errorf("failed to open event log: %v", err)
		}
	}
	defer svc.elog.Close()

	svc.elog.Info(1, fmt.Sprintf("starting %s service", serviceName))

	run := winsvc.Run
	if svc.isDebug {
		run = winsvcDebug.Run
	}

	err = run(serviceName, &serviceHandler{
		service: svc,
	})

	if err != nil {
		svc.elog.Error(1, fmt.Sprintf("%s service failed: %v", serviceName, err))
		return err
	}

	svc.elog.Info(1, fmt.Sprintf("%s service stopped", serviceName))
	return nil
}

// runInteractive runs the application in interactive mode
func (svc *WindowsService) runInteractive() error {
	svc.logger.Info("Application starting in interactive mode")

	// Simple interactive loop
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		svc.logger.Debug("Application heartbeat")
	}

	return nil
}

// serviceHandler implements the Windows service handler interface
type serviceHandler struct {
	service *WindowsService
}

func (handler *serviceHandler) Execute(args []string, r <-chan winsvc.ChangeRequest, changes chan<- winsvc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = winsvc.AcceptStop | winsvc.AcceptShutdown | winsvc.AcceptPauseAndContinue
	changes <- winsvc.Status{State: winsvc.StartPending}

	handler.service.logger.Info("Service starting")
	changes <- winsvc.Status{State: winsvc.Running, Accepts: cmdsAccepted}

	// Service heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case winsvc.Interrogate:
				changes <- c.CurrentStatus
			case winsvc.Stop, winsvc.Shutdown:
				changes <- winsvc.Status{State: winsvc.StopPending}

				handler.service.logger.Info("Service stopping")
				if err := handler.service.app.Stop(); err != nil {
					handler.service.logger.Error("Failed to stop app layer", "error", err)
				}
				return
			case winsvc.Pause:
				changes <- winsvc.Status{State: winsvc.Paused, Accepts: cmdsAccepted}

				handler.service.logger.Info("Service pausing")
				if err := handler.service.app.Pause(); err != nil {
					handler.service.logger.Error("Failed to pause app layer", "error", err)
				}
			case winsvc.Continue:
				changes <- winsvc.Status{State: winsvc.Running, Accepts: cmdsAccepted}

				handler.service.logger.Info("Service continuing")
				if err := handler.service.app.Resume(); err != nil {
					handler.service.logger.Error("Failed to resume app layer", "error", err)
				}
			default:
				// Log the error to the event log & service logger
				handler.service.logger.Error("unexpected control request", "request", c)
				handler.service.elog.Error(uint32(1), fmt.Sprintf("unexpected control request #%d", c))
			}
		case <-ticker.C:
			handler.service.logger.Debug("heartbeat")
		}
	}
}
