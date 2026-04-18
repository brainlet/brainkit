package main

import (
	"log"
	"os"
)

func tempDir() (string, error) {
	return os.MkdirTemp("", "brainkit-secrets-")
}

func cleanupTemp(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("cleanup %s: %v", dir, err)
	}
}
