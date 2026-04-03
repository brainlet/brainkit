package config

import (
	"fmt"
	"os"

	"github.com/brainlet/brainkit"
)

func Connect(cfg *CLIConfig) (*brainkit.BusClient, error) {
	nc, err := BuildNodeConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := brainkit.NewClient(nc)
	if err != nil {
		transport := cfg.Transport
		if transport == "" {
			transport = "memory"
		}
		return nil, fmt.Errorf("cannot connect to brainkit instance (%s)\nHint: start an instance with `brainkit start`", transport)
	}
	return client, nil
}

func MustConnect(cfg *CLIConfig) *brainkit.BusClient {
	client, err := Connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	return client
}
