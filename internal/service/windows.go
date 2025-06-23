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
		svc.logger.Debug("running as Windows service")

		run = winsvc.Run
		svc.elog, err = eventlog.Open("HATray")
		if err != nil {
			return fmt.Errorf("failed to open event log: %v", err)
		}
	} else {
		svc.logger.Debug("running as debug service")

		run = debug.Run
		svc.elog = debug.New("HATray")
	}

	defer svc.elog.Close()

	svc.elog.Info(1, "starting service")
	// Run the service with our handler
	err = run("HATray", &serviceHandler{
		service: svc,
	})
	if err != nil {
		svc.elog.Error(1, fmt.Sprintf("service failed: %v", err))
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

	handler.service.logger.Info("starting service")
	changes <- winsvc.Status{State: winsvc.Running, Accepts: cmdsAccepted}

	// Start the application; backgrounded so that the service can still respond to Windows control requests (the app layer can handle concurrent requests)
	go func() {
		// TODO: This has no true error handling, retry mechanism, or timeout mechanism. If this fails, then the service will be stuck in the 'StartPending' state.
		if err := handler.service.app.Resume(); err != nil {
			handler.service.logger.Error("failed to start (resume) app layer", "error", err)
		}
	}()

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

				if err := handler.service.app.Pause(); err != nil {
					handler.service.logger.Error("failed to pause app layer", "error", err)
				}
				return
			case winsvc.Pause:
				changes <- winsvc.Status{State: winsvc.Paused, Accepts: cmdsAccepted}

				handler.service.logger.Info("service pausing")
				if err := handler.service.app.Pause(); err != nil {
					handler.service.logger.Error("failed to pause app layer", "error", err)
				}
			case winsvc.Continue:
				changes <- winsvc.Status{State: winsvc.Running, Accepts: cmdsAccepted}

				handler.service.logger.Info("service continuing")
				if err := handler.service.app.Resume(); err != nil {
					handler.service.logger.Error("failed to resume app layer", "error", err)
				}
			default:
				// Log the error to the event log & service logger
				handler.service.logger.Error("unexpected control request", "request", c)
				handler.service.elog.Error(uint32(1), fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
}
