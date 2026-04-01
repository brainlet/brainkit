package adversarial_test

import (
	"context"
	"sync"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogging_CustomHandler — LogHandler receives .ts console output.
func TestLogging_CustomHandler(t *testing.T) {
	var mu sync.Mutex
	var logs []brainkit.LogEntry

	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		LogHandler: func(entry brainkit.LogEntry) {
			mu.Lock()
			logs = append(logs, entry)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	_, err = k.Deploy(ctx, "log-test.ts", `
		console.log("hello from ts");
		console.warn("warning from ts");
		console.error("error from ts");
		output("logged");
	`)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// Should have captured the console output
	assert.Greater(t, len(logs), 0, "LogHandler should receive console output")

	hasLog := false
	hasWarn := false
	hasError := false
	for _, l := range logs {
		if l.Level == "log" && l.Source == "log-test.ts" {
			hasLog = true
		}
		if l.Level == "warn" && l.Source == "log-test.ts" {
			hasWarn = true
		}
		if l.Level == "error" && l.Source == "log-test.ts" {
			hasError = true
		}
	}
	assert.True(t, hasLog, "should have log entry")
	assert.True(t, hasWarn, "should have warn entry")
	assert.True(t, hasError, "should have error entry")
}

// TestLogging_NilHandler — nil LogHandler doesn't panic.
func TestLogging_NilHandler(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:  "test",
		CallerID:   "test",
		FSRoot:     tmpDir,
		LogHandler: nil, // default
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	_, err = k.Deploy(ctx, "log-nil.ts", `console.log("no panic"); output("ok");`)
	require.NoError(t, err) // no panic
}

// TestLogging_ConcurrentOutput — multiple deployments logging concurrently.
func TestLogging_ConcurrentOutput(t *testing.T) {
	var mu sync.Mutex
	var count int

	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		LogHandler: func(entry brainkit.LogEntry) {
			mu.Lock()
			count++
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Deploy multiple services that all log
	for i := 0; i < 5; i++ {
		src := "concurrent-log.ts"
		k.Deploy(ctx, src, `console.log("concurrent"); output("ok");`)
		k.Teardown(ctx, src)
	}

	mu.Lock()
	assert.Greater(t, count, 0, "should have received log entries from concurrent deploys")
	mu.Unlock()
}
