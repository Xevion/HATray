package app

import (
	"fmt"
	"log/slog"
	"sync"
)

// App represents the main application layer that is generic and cross-platform
type App struct {
	logger *slog.Logger
	mu     sync.RWMutex
	state  AppState
}

// AppState represents the current state of the application
type AppState string

const (
	StateRunning AppState = "running"
	StatePaused  AppState = "paused"
	StateStopped AppState = "stopped"
)

// NewApp creates a new application instance
func NewApp(logger *slog.Logger) *App {
	return &App{
		logger: logger,
		state:  StateStopped,
	}
}

// Start transitions the application from Stopped or Paused to Running
func (app *App) Start() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch app.state {
	case StateRunning:
		return fmt.Errorf("application is already running")
	case StateStopped, StatePaused:
		// valid states to start from, do nothing
	default:
		return fmt.Errorf("cannot start application from state: %s", app.state)
	}

	app.logger.Info("starting application",
		"action", "start",
		"previous_state", app.state,
		"new_state", StateRunning)

	// TODO: Implement actual start logic
	// - Connect to Home Assistant WebSocket
	// - Start background tasks
	// - Start sensor monitoring

	app.state = StateRunning

	app.logger.Info("started successfully",
		"action", "start",
		"state", app.state)

	return nil
}

// Pause disconnects from the server and ceases any background tasks
func (app *App) Pause() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch app.state {
	case StatePaused:
		return fmt.Errorf("application is already paused")
	case StateStopped:
		return fmt.Errorf("cannot pause application when stopped")
	}

	app.logger.Info("pausing application",
		"action", "pause",
		"previous_state", app.state,
		"new_state", StatePaused)

	// TODO: Implement actual pause logic
	// - Disconnect from Home Assistant WebSocket
	// - Stop background tasks
	// - Pause sensor monitoring

	app.state = StatePaused

	app.logger.Info("paused successfully",
		"action", "pause",
		"state", app.state)

	return nil
}

// Resume connects to the server and initiates background tasks
func (app *App) Resume() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch app.state {
	case StateRunning:
		return fmt.Errorf("application is already running")
	case StateStopped:
		return fmt.Errorf("cannot resume application when stopped, instead, start the application")
	case StatePaused:
		// valid state to resume from, do nothing
	default:
		return fmt.Errorf("cannot resume application from state: %s", app.state)
	}

	app.logger.Info("resuming application",
		"action", "resume",
		"previous_state", app.state,
		"new_state", StateRunning)

	// TODO: Implement actual resume logic
	// - Connect to Home Assistant WebSocket
	// - Start background tasks
	// - Resume sensor monitoring

	app.state = StateRunning

	app.logger.Info("resumed successfully",
		"action", "resume",
		"state", app.state)

	return nil
}

// Reload pauses the application, re-reads configuration files, then resumes
func (a *App) Reload() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch a.state {
	case StateStopped:
		return fmt.Errorf("cannot reload application when stopped")
	case StatePaused:
		return fmt.Errorf("cannot reload application when paused")
	case StateRunning:
		// valid state to reload from, do nothing
	default:
		return fmt.Errorf("cannot reload application from state: %s", a.state)
	}

	a.logger.Info("starting application reload",
		"action", "reload",
		"current_state", a.state)

	// Pause if not already paused
	switch a.state {
	case StatePaused:
		// already paused, do nothing
	case StateRunning:
		if err := a.Pause(); err != nil {
			a.logger.Error("failed to pause during reload",
				"action", "reload",
				"error", err)
			return err
		}
	default:
		return fmt.Errorf("unexpected state encountered while pausing for reload: %s", a.state)
	}

	// TODO: Implement configuration reload logic
	// - Re-read TOML configuration files
	// - Validate configuration
	// - Update internal state with new configuration

	a.logger.Info("configuration reloaded successfully",
		"action", "reload")

	// Resume the application
	if err := a.Resume(); err != nil {
		a.logger.Error("failed to resume during reload",
			"action", "reload",
			"error", err)
		return err
	}

	a.logger.Info("application reload completed successfully",
		"action", "reload",
		"final_state", a.state)

	return nil
}

// GetState returns the current state of the application
func (a *App) GetState() AppState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// Stop stops the application completely
func (app *App) Stop() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch app.state {
	case StateStopped:
		return fmt.Errorf("application is already stopped")
	case StatePaused, StateRunning:
		// valid state to stop from, do nothing
	default:
		return fmt.Errorf("unexpected state encountered while stopping application: %s", app.state)
	}

	app.logger.Info("stopping application",
		"action", "stop",
		"previous_state", app.state,
		"new_state", StateStopped)

	// TODO: Implement actual stop logic
	// - Disconnect from all services
	// - Clean up resources
	// - Stop all background tasks

	app.state = StateStopped

	app.logger.Info("application stopped successfully",
		"action", "stop",
		"state", app.state)

	return nil
}
