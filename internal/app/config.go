package app

import (
	"fmt"
	"ha-tray/internal"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server *string `toml:"server"`
	APIKey string  `toml:"api_key"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		// Try loading from .env
		err := godotenv.Load()
		if err != nil {
			apiKey = os.Getenv("API_KEY")
		} else {
			apiKey = os.Getenv("API_KEY")
		}
	}

	instanceUrl := internal.Ptr(strings.TrimSpace(os.Getenv("INSTANCE_URL")))
	if *instanceUrl == "" {
		instanceUrl = nil
	}

	return &Config{
		Server: instanceUrl,
		APIKey: apiKey,
	}
}

// LoadConfig loads configuration from a TOML file
func LoadConfig(filename string) (*Config, error) {
	config := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// Create default config file if it doesn't exist
		if err := SaveConfig(filename, config); err != nil {
			return nil, fmt.Errorf("failed to create default config file: %w", err)
		}
		return config, nil
	}

	// Load existing config file
	if _, err := toml.DecodeFile(filename, config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a TOML file
func SaveConfig(filename string, config *Config) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server == nil || *c.Server == "" {
		return fmt.Errorf("server address is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}
