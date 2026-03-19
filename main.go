package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/getlantern/systray"
)

//go:embed icons/icon.ico
var iconData []byte

//go:embed icons/connected.ico
var connectedIconData []byte

//go:embed icons/disconnected.ico
var disconnectedIconData []byte

var logDir = "logs"

// systray vars
var mStatus *systray.MenuItem
var mExit *systray.MenuItem

func main() {
	// Systray setup
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("MQTT Logger")
	systray.SetTooltip("MQTT Logger")

	// Status
	mStatus = systray.AddMenuItem("", "")
	mStatus.Hide()
	// Config
	mConfig := systray.AddMenuItem("Config", "Show the Configuration")
	go func() {
		<-mConfig.ClickedCh
		exec.Command("notepad.exe", "config.yaml").Start()
	}()
	mConfig.Hide()
	systray.AddSeparator()
	// Exit
	mExit = systray.AddMenuItem("Exit", "Exit the application")
	go func() {
		<-mExit.ClickedCh
		systray.Quit()
	}()

	// Load configuration
	if err := loadConfig("config.yaml"); err != nil {
		fmt.Println(err)
		return
	}
	mConfig.Show()

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Println("Error creating log directory:", err)
		return
	}

	// Start MQTT reconnect loop
	go startMQTTLoop()
}

func startMQTTLoop() {
	mStatus.Show()
	for {
		opts := MQTT.NewClientOptions().
			AddBroker(cfg.Broker).
			SetClientID(cfg.ClientID).
			SetAutoReconnect(false) // we handle reconnect manually
		client := MQTT.NewClient(opts)

		fmt.Println("Attempting to connect to broker...")
		mStatus.SetTitle("Connecting...")
		mStatus.SetIcon(disconnectedIconData)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			fmt.Println("MQTT connect error:", token.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Printf("Connected to %s, subscribing to %s\n", cfg.Broker, cfg.Topic)
		if token := client.Subscribe(cfg.Topic, 0, handleMessage); token.Wait() && token.Error() != nil {
			fmt.Println("MQTT subscribe error:", token.Error())
			client.Disconnect(250)
			time.Sleep(1 * time.Second)
			continue
		}

		mStatus.SetTitle("Connected")
		mStatus.SetIcon(connectedIconData)

		// Wait until connection is lost
		for client.IsConnected() {
			time.Sleep(500 * time.Millisecond)
		}

		fmt.Println("Connection lost, retrying in 1 second...")
		mStatus.SetTitle("Disconnected")
		mStatus.Hide()
		time.Sleep(1 * time.Second)
	}
}

func onExit() {
	os.Exit(0)
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
