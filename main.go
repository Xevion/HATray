package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

type myservice struct{}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// Get the directory where the executable is located
	exePath, err := os.Executable()
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to get executable path: %v", err))
		return
	}
	exeDir := filepath.Dir(exePath)

	// Open log file in the same directory as the executable
	logFile, err := os.OpenFile(filepath.Join(exeDir, "current.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		elog.Error(1, fmt.Sprintf("Failed to open log file: %v", err))
		return
	}
	defer logFile.Close()

	// Create JSON logger
	logger := log.New(logFile, "", 0)

	// Log startup
	startupLog := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     "debug",
		"message":   "Service starting",
		"service":   "hass-tray",
	}
	startupJSON, _ := json.Marshal(startupLog)
	logger.Println(string(startupJSON))

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
				// Log shutdown
				shutdownLog := map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"level":     "debug",
					"message":   "Service stopping",
					"service":   "hass-tray",
				}
				shutdownJSON, _ := json.Marshal(shutdownLog)
				logger.Println(string(shutdownJSON))

				changes <- svc.Status{State: svc.StopPending}
				return
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				elog.Error(uint32(1), fmt.Sprintf("unexpected control request #%d", c))
			}
		case <-ticker.C:
			// Log heartbeat
			heartbeatLog := map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"level":     "debug",
				"message":   "Service heartbeat",
				"service":   "hass-tray",
			}
			heartbeatJSON, _ := json.Marshal(heartbeatLog)
			logger.Println(string(heartbeatJSON))
		}
	}
}

func runService(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &myservice{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}

func main() {
	isDebug, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	if !isDebug {
		runService("hass-tray", false)
		return
	}

	// Interactive mode - just run the service logic directly
	fmt.Println("Running in interactive mode...")

	// Get the current directory for log file
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Open log file
	logFile, err := os.OpenFile(filepath.Join(currentDir, "current.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Create JSON logger
	logger := log.New(logFile, "", 0)

	// Log startup
	startupLog := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     "debug",
		"message":   "Application starting in interactive mode",
		"service":   "hass-tray",
	}
	startupJSON, _ := json.Marshal(startupLog)
	logger.Println(string(startupJSON))

	fmt.Println("Press Ctrl+C to stop...")

	// Simple interactive loop
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Log heartbeat
			heartbeatLog := map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"level":     "debug",
				"message":   "Application heartbeat",
				"service":   "hass-tray",
			}
			heartbeatJSON, _ := json.Marshal(heartbeatLog)
			logger.Println(string(heartbeatJSON))
		}
	}
}
