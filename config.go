package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Broker   string `yaml:"broker"`
	Topic    string `yaml:"topic"`
	ClientID string `yaml:"clientID"`
	KeepDays int    `yaml:"keepdays"`
}

// Global config instance
var cfg Config

func loadConfig(path string) error {
	// Create default config if missing
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefaultConfig(path); err != nil {
			return fmt.Errorf("creating default config: %w", err)
		}
		return fmt.Errorf("default config created at %s, please edit it", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	return nil
}

func createDefaultConfig(path string) error {
	content := `# mqtt-logger configuration
broker: "tcp://localhost:1883"
topic: "your/topic/#"
clientID: "mqtt-logger"
keepdays: 14
`
	return os.WriteFile(path, []byte(content), 0644)
}
