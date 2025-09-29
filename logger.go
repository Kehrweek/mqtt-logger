package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Broker string `yaml:"broker"`
	Topic string `yaml:"topic"`
	ClientID string `yaml:"clientID"`
	KeepDays int `yaml:"keepdays"`
}

var cfg Config
var logDir = "logs"

func main () {
	// Check for config.yaml, create if missing
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		if err := createDefaultConfig(); err != nil {
			fmt.Println("Error creating default config.yaml:", err)
			return
		}
		fmt.Println("Created default config.yaml. Edit it before running again.")
		return
	}

	// Load configuration
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Failed to read config.yaml:", err)
		return
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Println("Failed to parse config.yaml:", err)
		return
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Println("Error creating log directory:", err)
		return
	}

	// MQTT setup
	opts := MQTT.NewClientOptions().AddBroker(cfg.Broker).SetClientID(cfg.ClientID)
	client := MQTT.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("MQTT connect error:", token.Error())
		return
	}
	fmt.Printf("Connected to %s, subscribed to %s\n", cfg.Broker, cfg.Topic)

	if token := client.Subscribe(cfg.Topic, 0, handleMessage); token.Wait() && token.Error() != nil {
		fmt.Println("MQTT subscribe error:", token.Error())
		return
	}

	// Keep running forever
	select {}
}

func handleMessage(_ MQTT.Client, msg MQTT.Message) {
	// Build log file name: YYYY-MM-DD_topic.log
	today := time.Now().Format("2006.01.02")
	safeTopic := strings.ReplaceAll(msg.Topic(), "/", "_")
	fileName := fmt.Sprintf("%s %s.log", today, safeTopic)
	logFile := filepath.Join(logDir, fileName)

	// Open log file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer f.Close()

	// Write message
	line := fmt.Sprintf("[%s] %s\n",
		time.Now().Format("2006.01.02 15:04:05"),
		string(msg.Payload()))
	if _, err := f.WriteString(line); err != nil {
		fmt.Println("Error writing log:", err)
	}

	// Cleanup old logs
	cleanupOldLogs()
}

func cleanupOldLogs() {
	cutoff := time.Now().AddDate(0, 0, -cfg.KeepDays)
	files, _ := os.ReadDir(logDir)

	for _, f := range files {
		if fi, err := f.Info(); err == nil {
			if fi.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(logDir, f.Name()))
			}
		}
	}
}

func createDefaultConfig() error {
	content := `# mqtt-logger configuration
broker: "tcp://localhost:1883"
topic: "your/topic/#"
clientid: "mqtt-logger"
keepdays: 14
`
	return os.WriteFile("config.yaml", []byte(content), 0644)
}
