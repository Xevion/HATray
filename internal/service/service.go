package service

import (
	"ha-tray/internal/app"
	"log/slog"
)

// This is an intentionally very-simple interface as the main program entrypoint needs to know very little about the service layer.
// The service layer is completely responsible for the lifecycle of the application, implemented per-platform.
type Service interface {
	Run() error
}

// NewService creates a new service instance for the current platform
func NewService(logger *slog.Logger, appLayer *app.App) Service {
	return newService(logger, appLayer)
}
