package app

import (
	"fmt"
	"ha-tray/internal"
	"log/slog"
	"sync"
	"time"

	ga "github.com/Xevion/go-ha"
)

// App represents the main application layer that is generic and cross-platform
type App struct {
	logger      *slog.Logger
	mu          sync.RWMutex
	state       AppState
	config      *Config
	lastStarted *time.Time // time of last start, nil if never started
	tray        *Tray      // simple interface to systray
	ha          *ga.App
}

// AppState represents the current state of the application
type AppState int

const (
	StatePaused AppState = iota
	StateRunning
)

// String returns the string representation of the AppState
func (s AppState) String() string {
	switch s {
	case StatePaused:
		return "paused"
	case StateRunning:
		return "running"
	default:
		return "unknown"
	}
}

// NewApp creates a new application instance
func NewApp(logger *slog.Logger) *App {
	return &App{
		logger:      logger.With("type", "app"),
		state:       StatePaused,
		config:      nil,
		lastStarted: nil,
		tray:        NewTray(logger.With("type", "tray")),
		ha:          nil,
	}
}

// Pause disconnects from the server and ceases any background tasks
func (app *App) Pause() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch app.state {
	case StatePaused:
		app.logger.Warn("application is already paused")
		return nil
	case StateRunning:
		// valid state to pause from, do nothing
	default:
		return fmt.Errorf("unexpected state encountered while pausing application: %s", app.state)
	}

	app.logger.Info("pausing application",
		"action", "pause",
		"previous_state", app.state,
		"new_state", StatePaused)

	// - Disconnect from Home Assistant WebSocket
	err := app.ha.Close()
	if err != nil {
		app.logger.Error("failed to close home assistant connection", "error", err)
		return err
	}

	// - Stop tray icon event loop
	err = app.tray.Stop()
	if err != nil {
		app.logger.Error("failed to stop tray", "error", err)
		return err
	}

	app.state = StatePaused

	app.logger.Info("paused successfully",
		"action", "pause",
		"state", app.state)

	return nil
}

// Resume connects to the server and initiates background tasks
// This function does not block permanently, it will return very quickly with an error if anything goes wrong.
func (app *App) Resume() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	switch app.state {
	case StateRunning:
		app.logger.Warn("application is already running")
		return nil
	case StatePaused:
		// valid state to resume from, do nothing
	default:
		return fmt.Errorf("unexpected state encountered while resuming application: %s", app.state)
	}

	app.logger.Info("resuming application",
		"action", "resume",
		"previous_state", app.state,
		"new_state", StateRunning,
		"has_started", app.lastStarted,
	)

	// TODO: Implement actual resume logic
	// - Connect to Home Assistant WebSocket
	// - Start background tasks
	// - Resume sensor monitoring
	err := app.tray.Start(fmt.Sprintf("HATray v%s", "0.0.1"))
	if err != nil {
		app.logger.Error("failed to start tray", "error", err)
		return err
	}

	app.config = DefaultConfig()

	if err := app.config.Validate(); err != nil {
		app.logger.Error("invalid configuration", "error", err)
		return err
	}

	app.ha, err = ga.NewApp(ga.NewAppRequest{
		URL:         *app.config.Server,
		HAAuthToken: app.config.APIKey,
	})
	if err != nil {
		app.logger.Error("failed to create Home Assistant app", "error", err)
		return err
	}

	app.ha.RegisterEntityListeners(ga.NewEntityListener().EntityIds("binary_sensor.bedroom_door_opening").Call(app.onEntityStateChange).Build())

	go app.ha.Start()

	time.Sleep(2 * time.Second)
	state, err := app.ha.GetState().Get("binary_sensor.bedroom_door_opening")
	if err != nil {
		app.logger.Error("failed to get entity", "error", err)
		return err
	}

	app.logger.Info("state", "state", state.State)

	if state.State == "on" {
		app.tray.SetIcon(IconOpen)
	} else {
		app.tray.SetIcon(IconClosed)
	}

	app.state = StateRunning
	app.lastStarted = internal.Ptr(time.Now())

	app.logger.Info("resumed successfully",
		"action", "resume",
		"state", app.state)

	return nil
}

func (a *App) onEntityStateChange(se *ga.Service, st ga.State, e ga.EntityData) {
	entity, err := st.Get(e.TriggerEntityId)
	if err != nil {
		a.logger.Error("failed to get entity", "error", err)
		return
	}
	a.logger.Info("sensor.test state changed", "entity", e.TriggerEntityId, "state", entity.State)

	if entity.State == "on" {
		a.tray.SetIcon(IconOpen)
	} else {
		a.tray.SetIcon(IconClosed)
	}
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
		a.logger.Info("application is already paused during reload")
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
