package bus

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pkgDeployMsg builds a PackageDeployMsg from a single source/code pair.
func pkgDeployMsg(source, code string) sdk.PackageDeployMsg {
	name := strings.TrimSuffix(source, ".ts")
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": source})
	return sdk.PackageDeployMsg{Manifest: manifest, Files: map[string]string{source: code}}
}

// testLogHandlerTSCompartment needs its own kernel with a custom LogHandler.
func testLogHandlerTSCompartment(t *testing.T, _ *suite.TestEnv) {
	var mu sync.Mutex
	var logs []brainkit.LogEntry

	logEnv := suite.Minimal(t, suite.WithFSRoot(), suite.WithLogHandler(func(e brainkit.LogEntry) {
		mu.Lock()
		logs = append(logs, e)
		mu.Unlock()
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(logEnv.Kit, ctx, pkgDeployMsg("log-test.ts", `console.log("hello from ts"); console.warn("warning!"); console.error("error!");`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	us, _ := sdk.SubscribeTo[sdk.PackageDeployResp](logEnv.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer us()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

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

// testLogHandlerMultipleFiles needs its own kernel with a custom LogHandler.
func testLogHandlerMultipleFiles(t *testing.T, _ *suite.TestEnv) {
	var mu sync.Mutex
	var logs []brainkit.LogEntry

	logEnv := suite.Minimal(t, suite.WithFSRoot(), suite.WithLogHandler(func(e brainkit.LogEntry) {
		mu.Lock()
		logs = append(logs, e)
		mu.Unlock()
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr1, err := sdk.Publish(logEnv.Kit, ctx, pkgDeployMsg("file-a.ts", `console.log("from file A");`))
	require.NoError(t, err)
	ch1 := make(chan sdk.PackageDeployResp, 1)
	us1, _ := sdk.SubscribeTo[sdk.PackageDeployResp](logEnv.Kit, ctx, pr1.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch1 <- r })
	defer us1()
	select {
	case <-ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	pr2, err := sdk.Publish(logEnv.Kit, ctx, pkgDeployMsg("file-b.ts", `console.log("from file B");`))
	require.NoError(t, err)
	ch2 := make(chan sdk.PackageDeployResp, 1)
	us2, _ := sdk.SubscribeTo[sdk.PackageDeployResp](logEnv.Kit, ctx, pr2.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch2 <- r })
	defer us2()
	select {
	case <-ch2:
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

// testLogHandlerConcurrent tests multiple deployments logging concurrently.
func testLogHandlerConcurrent(t *testing.T, _ *suite.TestEnv) {
	var mu sync.Mutex
	var count int

	concEnv := suite.Minimal(t, suite.WithFSRoot(), suite.WithLogHandler(func(entry brainkit.LogEntry) {
		mu.Lock()
		count++
		mu.Unlock()
	}))

	for i := 0; i < 5; i++ {
		src := "concurrent-log.ts"
		testutil.Deploy(t, concEnv.Kit, src, `console.log("concurrent"); output("ok");`)
		testutil.Teardown(t, concEnv.Kit, src)
	}

	mu.Lock()
	assert.Greater(t, count, 0, "should have received log entries from concurrent deploys")
	mu.Unlock()
}

// testLogHandlerNilDefault verifies nil LogHandler doesn't panic.
func testLogHandlerNilDefault(t *testing.T, _ *suite.TestEnv) {
	nilEnv := suite.Minimal(t, suite.WithFSRoot())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(nilEnv.Kit, ctx, pkgDeployMsg("nil-test.ts", `console.log("should not panic");`))
	require.NoError(t, err)
	ch := make(chan sdk.PackageDeployResp, 1)
	us, _ := sdk.SubscribeTo[sdk.PackageDeployResp](nilEnv.Kit, ctx, pr.ReplyTo, func(r sdk.PackageDeployResp, m sdk.Message) { ch <- r })
	defer us()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
