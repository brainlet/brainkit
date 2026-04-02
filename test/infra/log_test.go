package infra_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogHandler_TSCompartment(t *testing.T) {
	var mu sync.Mutex
	var logs []brainkit.LogEntry

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-log",
		FSRoot: t.TempDir(),
		LogHandler: func(e brainkit.LogEntry) {
			mu.Lock()
			logs = append(logs, e)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy .ts that logs at different levels
	_pr1, err := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "log-test.ts",
		Code:   `console.log("hello from ts"); console.warn("warning!"); console.error("error!");`,
	})
	require.NoError(t, err)
	_ch1 := make(chan messages.KitDeployResp, 1)
	_us1, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, _pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch1 <- r })
	defer _us1()
	select {
	case <-_ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Check captured logs
	mu.Lock()
	defer mu.Unlock()

	var tagged []string
	for _, l := range logs {
		if l.Source == "log-test.ts" {
			tagged = append(tagged, l.Level+":"+l.Message)
		}
	}
	assert.Contains(t, tagged, "log:hello from ts")
	assert.Contains(t, tagged, "warn:warning!")
	assert.Contains(t, tagged, "error:error!")
}

func TestLogHandler_TSCompartment_MultipleFiles(t *testing.T) {
	var mu sync.Mutex
	var logs []brainkit.LogEntry

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-log-multi",
		FSRoot: t.TempDir(),
		LogHandler: func(e brainkit.LogEntry) {
			mu.Lock()
			logs = append(logs, e)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy two different .ts files
	_pr2, err := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "file-a.ts",
		Code:   `console.log("from file A");`,
	})
	require.NoError(t, err)
	_ch2 := make(chan messages.KitDeployResp, 1)
	_us2, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, _pr2.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch2 <- r })
	defer _us2()
	select {
	case <-_ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	_pr3, err := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "file-b.ts",
		Code:   `console.log("from file B");`,
	})
	require.NoError(t, err)
	_ch3 := make(chan messages.KitDeployResp, 1)
	_us3, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, _pr3.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch3 <- r })
	defer _us3()
	select {
	case <-_ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	mu.Lock()
	defer mu.Unlock()

	var fromA, fromB []string
	for _, l := range logs {
		if l.Source == "file-a.ts" {
			fromA = append(fromA, l.Message)
		}
		if l.Source == "file-b.ts" {
			fromB = append(fromB, l.Message)
		}
	}
	assert.Contains(t, fromA, "from file A", "file-a.ts logs should be tagged")
	assert.Contains(t, fromB, "from file B", "file-b.ts logs should be tagged")
}
func TestLogHandler_NilDefault(t *testing.T) {
	// When LogHandler is nil, logs should go to default (stdout) without panicking
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-nil-log",
		FSRoot: t.TempDir(),
		// LogHandler: nil — default
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Should not panic
	_pr6, err := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "nil-test.ts",
		Code:   `console.log("should not panic");`,
	})
	require.NoError(t, err)
	_ch6 := make(chan messages.KitDeployResp, 1)
	_us6, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, _pr6.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch6 <- r })
	defer _us6()
	select {
	case <-_ch6:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
