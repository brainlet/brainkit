//go:build production

// Package syncx provides sync.Mutex and sync.RWMutex wrappers.
// By default, these use go-deadlock for detection during development.
// With -tags production, they switch to plain Go mutexes — zero overhead.
//
// Usage: replace sync.Mutex with syncx.Mutex across the codebase.
// Build with: go build -tags production ./... for native mutexes.
package syncx

import "sync"

// Mutex is sync.Mutex in production builds.
type Mutex = sync.Mutex

// RWMutex is sync.RWMutex in production builds.
type RWMutex = sync.RWMutex
