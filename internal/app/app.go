package app

import (
	"fmt"
	"log/slog"
	"sync"
)

// App represents the main application layer that is generic and cross-platform
type App struct {
	logger     *slog.Logger
	mu         sync.RWMutex
	state      AppState
	hasStarted bool // true only if the application has ever been started (i.e. has been resumed from initial paused state)
}

// AppState represents the current state of the application
type AppState string

const (
	StateRunning AppState = "running"
	StatePaused  AppState = "paused"
)

// NewApp creates a new application instance
func NewApp(logger *slog.Logger) *App {
	return &App{
		logger:     logger,
		state:      StatePaused,
		hasStarted: false,
	}
}

// Pause disconnects from the server and ceases any background tasks
func (app *App) Pause() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch app.state {
	case StatePaused:
		return fmt.Errorf("application is already paused")
	case StateRunning:
		// valid state to pause from, do nothing
	default:
		return fmt.Errorf("unexpected state encountered while pausing application: %s", app.state)
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
	case StatePaused:
		// valid state to resume from, do nothing
	default:
		return fmt.Errorf("unexpected state encountered while resuming application: %s", app.state)
	}

	app.logger.Info("resuming application",
		"action", "resume",
		"previous_state", app.state,
		"new_state", StateRunning,
		"has_started", app.hasStarted,
	)

	// TODO: Implement actual resume logic
	// - Connect to Home Assistant WebSocket
	// - Start background tasks
	// - Resume sensor monitoring

	app.state = StateRunning
	app.hasStarted = true

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
	case StatePaused:
		return fmt.Errorf("cannot reload application when paused")
	case StateRunning:
		// valid state to reload from, do nothing
	default:
		return fmt.Errorf("unexpected state encountered while reloading application: %s", a.state)
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
