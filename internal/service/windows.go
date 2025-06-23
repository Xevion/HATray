//go:build windows

package service

import (
	"fmt"
	"log/slog"
	"time"

	"ha-tray/internal/app"

	winsvc "golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

const serviceName = "HATray"

// WindowsService implements the Service interface for Windows
type WindowsService struct {
	app    *app.App
	logger *slog.Logger // logger instance, logs to file (and console in debug mode)
	elog   debug.Log    // event log instance; connects to the Windows Event Log
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

	var run func(string, winsvc.Handler) error

	// Acquire the appropriate run function & eventlog instance depending on service type
	if isService {
		svc.logger.Debug("running as Windows service", "serviceName", serviceName)

		run = winsvc.Run
		svc.elog, err = eventlog.Open(serviceName)
		if err != nil {
			return fmt.Errorf("failed to open event log: %v", err)
		}
	} else {
		svc.logger.Debug("running as debug service", "serviceName", serviceName)

		run = debug.Run
		svc.elog = debug.New(serviceName)
	}

	defer svc.elog.Close()

	svc.elog.Info(1, fmt.Sprintf("starting %s service", serviceName))
	// Run the service with our handler
	err = run(serviceName, &serviceHandler{
		service: svc,
	})
	if err != nil {
		svc.elog.Error(1, fmt.Sprintf("%s service failed: %v", serviceName, err))
		return err
	}

	return nil
}

type serviceHandler struct {
	service *WindowsService
}

func (handler *serviceHandler) Execute(args []string, r <-chan winsvc.ChangeRequest, changes chan<- winsvc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = winsvc.AcceptStop | winsvc.AcceptShutdown | winsvc.AcceptPauseAndContinue
	changes <- winsvc.Status{State: winsvc.StartPending}

	handler.service.logger.Info("service starting")
	changes <- winsvc.Status{State: winsvc.Running, Accepts: cmdsAccepted}

	// Service heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		// handle heartbeat
		case <-ticker.C:
			// TODO: in debug mode, I'd like heartbeats to have more interactive, changing information, such as state details, connection status, runtime etc.
			handler.service.logger.Debug("heartbeat")
		// handle service control requests
		case c := <-r:
			handler.service.logger.Debug("service control request", "request", c)

			switch c.Cmd {
			case winsvc.Interrogate:
				changes <- c.CurrentStatus
				handler.service.logger.Debug("service interrogate", "status", c.CurrentStatus)
			case winsvc.Stop, winsvc.Shutdown:
				changes <- winsvc.Status{State: winsvc.StopPending}

				handler.service.logger.Info("service stopping", "shutdown", c.Cmd == winsvc.Shutdown)

				if err := handler.service.app.Stop(); err != nil {
					handler.service.logger.Error("Failed to stop app layer", "error", err)
				}
				return
			case winsvc.Pause:
				changes <- winsvc.Status{State: winsvc.Paused, Accepts: cmdsAccepted}

				handler.service.logger.Info("service pausing")
				if err := handler.service.app.Pause(); err != nil {
					handler.service.logger.Error("Failed to pause app layer", "error", err)
				}
			case winsvc.Continue:
				changes <- winsvc.Status{State: winsvc.Running, Accepts: cmdsAccepted}

				handler.service.logger.Info("service continuing")
				if err := handler.service.app.Resume(); err != nil {
					handler.service.logger.Error("Failed to resume app layer", "error", err)
				}
			default:
				// Log the error to the event log & service logger
				handler.service.logger.Error("unexpected control request", "request", c)
				handler.service.elog.Error(uint32(1), fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
}
