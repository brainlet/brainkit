//go:build !production

package syncx

import "github.com/sasha-s/go-deadlock"

// Mutex uses go-deadlock for detection by default during development.
type Mutex = deadlock.Mutex

// RWMutex uses go-deadlock for detection by default during development.
type RWMutex = deadlock.RWMutex
