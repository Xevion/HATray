package app

import (
	"fmt"
	"ha-tray/internal"
	"log/slog"
	"time"

	"github.com/getlantern/systray"
)

type IconReference string

const (
	IconOpen    IconReference = "open"
	IconClosed  IconReference = "closed"
	IconUnknown IconReference = "unknown"
)

// Path returns the path to the icon file
func (i IconReference) Path() string {
	switch i {
	case IconOpen:
		return "resources/open.ico"
	case IconClosed:
		return "resources/closed.ico"
	default:
		return "resources/unknown.ico"
	}
}

type Tray struct {
	active      bool
	currentIcon *IconReference
	logger      *slog.Logger
}

func NewTray(logger *slog.Logger) *Tray {
	return &Tray{
		logger:      logger,
		currentIcon: nil,
		active:      false,
	}
}

func (t *Tray) SetIcon(icon IconReference) error {
	if !t.active {
		return fmt.Errorf("tray is not active")
	}

	iconBytes, err := internal.Icons.ReadFile(icon.Path())
	if err != nil {
		return fmt.Errorf("failed to read icon: %w", err)
	}
	systray.SetIcon(iconBytes)
	t.currentIcon = &icon

	return nil
}

func (t *Tray) Start(title string) error {
	if t.active {
		t.logger.Warn("tray is already active")
		return nil
	}

	t.logger.Info("attempting to start systray", "title", title)
	readyTimeout := make(chan struct{}, 1)
	go systray.Run(func() {
		systray.SetTitle(title)
		systray.SetTooltip(title)

		t.logger.Info("systray started")
		readyTimeout <- struct{}{}
		close(readyTimeout)
	}, func() {
		t.active = false
	})

	select {
	case <-readyTimeout:
		t.logger.Info("systray start confirmed")
		t.active = true
		return nil
	case <-time.After(5 * time.Second):
		close(readyTimeout)
		t.logger.Error("systray start timed out")
		return fmt.Errorf("tray did not start in time")
	}
}

func (t *Tray) Stop() error {
	if !t.active {
		return fmt.Errorf("tray is not active")
	}

	systray.Quit()
	t.active = false

	return nil
}
