package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brainlet/brainkit"
)

// Connect reads the control port from the pidfile and creates an HTTP BusClient.
func Connect(cfg *CLIConfig) (*brainkit.BusClient, error) {
	pidFile := filepath.Join("data", "brainkit.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil, fmt.Errorf("cannot find running brainkit instance (no %s)\nHint: start an instance with `brainkit start`", pidFile)
	}
	port := strings.TrimSpace(string(data))
	if port == "" {
		return nil, fmt.Errorf("invalid pidfile %s", pidFile)
	}
	return brainkit.NewClient("http://127.0.0.1:" + port), nil
}

func MustConnect(cfg *CLIConfig) *brainkit.BusClient {
	client, err := Connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	return client
}
