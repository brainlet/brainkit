package gateway

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	bkgw "github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testStreamHeartbeatTimeout(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	testutil.Deploy(t, k.Kit, "hb-timeout.ts", `
		bus.on("stall", async (msg) => {
			msg.stream.text("start");
			// Never sends end — should trigger heartbeat timeout
			await new Promise(r => setTimeout(r, 60000));
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := gwStartWithStream(t, k.Kit, &bkgw.StreamConfig{
		HeartbeatInterval: 1 * time.Second,
		HeartbeatTimeout:  3 * time.Second,
		MaxDuration:       30 * time.Second,
		MaxEvents:         100,
		GracePeriod:       5 * time.Second,
	})
	gw.HandleStream("GET", "/api/stall", "ts.hb-timeout.stall")

	resp, err := http.Get(addr + "/api/stall")
	require.NoError(t, err)
	defer resp.Body.Close()

	content := readSSEContent(t, resp, 10*time.Second)
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, `"reason":"heartbeat_timeout"`)
}

func testStreamMaxDuration(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	testutil.Deploy(t, k.Kit, "slow-stream.ts", `
		bus.on("slow", async (msg) => {
			for (var i = 0; i < 100; i++) {
				msg.stream.text("tick " + i);
				await new Promise(r => setTimeout(r, 500));
			}
			msg.stream.end({});
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := gwStartWithStream(t, k.Kit, &bkgw.StreamConfig{
		HeartbeatInterval: 1 * time.Second,
		HeartbeatTimeout:  10 * time.Second,
		MaxDuration:       3 * time.Second,
		MaxEvents:         10000,
		GracePeriod:       5 * time.Second,
	})
	gw.HandleStream("GET", "/api/slow", "ts.slow-stream.slow")

	resp, err := http.Get(addr + "/api/slow")
	require.NoError(t, err)
	defer resp.Body.Close()

	content := readSSEContent(t, resp, 10*time.Second)
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, `"reason":"max_duration"`)
}

func testStreamMaxEvents(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	testutil.Deploy(t, k.Kit, "flood.ts", `
		bus.on("flood", async (msg) => {
			for (var i = 0; i < 20; i++) {
				msg.stream.text("msg " + i);
			}
			msg.stream.end({});
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := gwStartWithStream(t, k.Kit, &bkgw.StreamConfig{
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTimeout:  25 * time.Second,
		MaxDuration:       30 * time.Second,
		MaxEvents:         5,
		GracePeriod:       5 * time.Second,
	})
	gw.HandleStream("GET", "/api/flood", "ts.flood.flood")

	resp, err := http.Get(addr + "/api/flood")
	require.NoError(t, err)
	defer resp.Body.Close()

	content := readSSEContent(t, resp, 10*time.Second)
	assert.Contains(t, content, `"reason":"max_events"`)
}

func testStreamKeepaliveComments(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	// Handler has a 12s delay between events — heartbeat fires at 10s interval
	testutil.Deploy(t, k.Kit, "delayed.ts", `
		bus.on("delayed", async (msg) => {
			msg.stream.text("first");
			await new Promise(r => setTimeout(r, 12000));
			msg.stream.text("second");
			msg.stream.end({});
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := gwStartWithStream(t, k.Kit, &bkgw.StreamConfig{
		HeartbeatTimeout: 25 * time.Second,
		MaxDuration:      30 * time.Second,
		MaxEvents:        10000,
		GracePeriod:      5 * time.Second,
	})
	gw.HandleStream("GET", "/api/delayed", "ts.delayed.delayed")

	resp, err := http.Get(addr + "/api/delayed")
	require.NoError(t, err)
	defer resp.Body.Close()

	content := readSSEContent(t, resp, 20*time.Second)
	assert.Contains(t, content, ":keepalive")
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, "event: end")
}

func testStreamReconnection(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	testutil.Deploy(t, k.Kit, "recon.ts", `
		bus.on("stream", async (msg) => {
			for (var i = 0; i < 8; i++) {
				msg.stream.text("chunk " + i);
				await new Promise(r => setTimeout(r, 300));
			}
			msg.stream.end({ final: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := gwStartWithStream(t, k.Kit, &bkgw.StreamConfig{
		HeartbeatInterval: 1 * time.Second,
		HeartbeatTimeout:  10 * time.Second,
		MaxDuration:       30 * time.Second,
		MaxEvents:         10000,
		GracePeriod:       10 * time.Second,
	})
	gw.HandleStream("GET", "/api/recon", "ts.recon.stream")

	// First connection — read 3 events then disconnect
	resp1, err := http.Get(addr + "/api/recon")
	require.NoError(t, err)

	scanner := bufio.NewScanner(resp1.Body)
	var lastID string
	eventsRead := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "id: ") {
			lastID = strings.TrimPrefix(line, "id: ")
		}
		if strings.HasPrefix(line, "event: text") {
			eventsRead++
			if eventsRead >= 3 {
				break
			}
		}
	}
	resp1.Body.Close()
	require.NotEmpty(t, lastID, "should have received at least one id")
	require.GreaterOrEqual(t, eventsRead, 3, "should have read 3 text events")

	// Wait briefly for more events to buffer
	time.Sleep(1 * time.Second)

	// Reconnect with Last-Event-Id
	req, _ := http.NewRequest("GET", addr+"/api/recon", nil)
	req.Header.Set("Last-Event-Id", lastID)
	resp2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, 200, resp2.StatusCode)
	content := readSSEContent(t, resp2, 10*time.Second)
	// Should receive replayed events + live events + end
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, "event: end")
}

func testStreamSessionExpired(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	testutil.Deploy(t, k.Kit, "expire.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.text("hello");
			msg.stream.end({});
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := gwStartWithStream(t, k.Kit, &bkgw.StreamConfig{
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTimeout:  25 * time.Second,
		MaxDuration:       30 * time.Second,
		MaxEvents:         10000,
		GracePeriod:       1 * time.Second, // very short grace period
	})
	gw.HandleStream("GET", "/api/expire", "ts.expire.stream")

	// First connection — completes normally
	resp1, err := http.Get(addr + "/api/expire")
	require.NoError(t, err)
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	require.Contains(t, string(body1), "event: end")

	// Extract stream token from first event
	lines := strings.Split(string(body1), "\n")
	var lastID string
	for _, line := range lines {
		if strings.HasPrefix(line, "id: ") {
			lastID = strings.TrimPrefix(line, "id: ")
		}
	}
	require.NotEmpty(t, lastID)

	// Wait past grace period
	time.Sleep(2 * time.Second)

	// Reconnect — should get 410 Gone
	req, _ := http.NewRequest("GET", addr+"/api/expire", nil)
	req.Header.Set("Last-Event-Id", lastID)
	resp2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusGone, resp2.StatusCode)
}

func testStreamConcurrent(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	for i := 0; i < 3; i++ {
		name := []string{"svc-a", "svc-b", "svc-c"}[i]
		testutil.Deploy(t, k.Kit, name+".ts", `
			bus.on("stream", async (msg) => {
				msg.stream.text("`+name+`");
				msg.stream.end({});
			});
		`)
	}
	time.Sleep(200 * time.Millisecond)

	gw, addr := gwStartWithStream(t, k.Kit, nil)
	gw.HandleStream("GET", "/api/a", "ts.svc-a.stream")
	gw.HandleStream("GET", "/api/b", "ts.svc-b.stream")
	gw.HandleStream("GET", "/api/c", "ts.svc-c.stream")

	results := make([]string, 3)
	done := make(chan int, 3)
	for i, path := range []string{"/api/a", "/api/b", "/api/c"} {
		go func(idx int, p string) {
			resp, err := http.Get(addr + p)
			if err != nil {
				done <- idx
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			results[idx] = string(body)
			done <- idx
		}(i, path)
	}
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("stream %d timed out", i)
		}
	}

	assert.Contains(t, results[0], "svc-a")
	assert.Contains(t, results[1], "svc-b")
	assert.Contains(t, results[2], "svc-c")
}

func testStreamGatewayShutdown(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t)
	testutil.Deploy(t, k.Kit, "shutdown-test.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.text("start");
			await new Promise(r => setTimeout(r, 30000));
			msg.stream.end({});
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw := bkgw.New(k.Kit, bkgw.Config{
		Listen:  "127.0.0.1:0",
		Timeout: 5 * time.Second,
		Stream: &bkgw.StreamConfig{
			HeartbeatInterval: 1 * time.Second,
			HeartbeatTimeout:  25 * time.Second,
			MaxDuration:       30 * time.Second,
			MaxEvents:         10000,
			GracePeriod:       5 * time.Second,
		},
	})
	require.NoError(t, gw.Start())
	addr := "http://" + gw.Addr()
	gw.HandleStream("GET", "/api/shutdown", "ts.shutdown-test.stream")

	// Start a stream
	resp, err := http.Get(addr + "/api/shutdown")
	require.NoError(t, err)

	// Read first event
	scanner := bufio.NewScanner(resp.Body)
	gotText := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "event: text") {
			gotText = true
			break
		}
	}
	require.True(t, gotText, "should receive text event before shutdown")

	// Stop gateway while stream is active
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- gw.Stop() }()

	select {
	case <-shutdownDone:
		// Stop completed — may return context deadline exceeded from http.Server.Shutdown
		// if active SSE connections didn't close within the 10s timeout.
	case <-time.After(15 * time.Second):
		t.Fatal("gateway.Stop() blocked for > 15s")
	}

	resp.Body.Close()
}
