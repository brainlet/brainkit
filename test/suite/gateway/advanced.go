package gateway

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAdvSSEStreaming — SSE endpoint streams typed events.
func testAdvSSEStreaming(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "sse-handler.ts", `
		bus.on("stream", function(msg) {
			msg.stream.text("hello");
			msg.stream.text("world");
			msg.stream.progress(50, "half");
			msg.stream.end({done: true});
		});
	`)

	gw.HandleStream("GET", "/sse-test", "ts.sse-handler.stream")

	resp, err := http.Get("http://" + gw.Addr() + "/sse-test")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	// Read SSE events
	scanner := bufio.NewScanner(resp.Body)
	var events []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:") {
			events = append(events, line)
		}
		if strings.Contains(line, "end") {
			break
		}
	}
	assert.Greater(t, len(events), 0, "should receive SSE events")
}

// testAdvWebhookDelivery — webhook endpoint accepts POST.
func testAdvWebhookDelivery(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "webhook-handler.ts", `
		bus.on("webhook", function(msg) {
			// Webhook doesn't reply — fire and forget
		});
	`)

	gw.HandleWebhook("POST", "/webhook", "ts.webhook-handler.webhook")

	status, _ := gwPost(t, gw, "/webhook", `{"event":"push","repo":"test"}`)
	assert.Equal(t, 200, status)
}

// testAdvMultipleRoutes — multiple routes coexist.
func testAdvMultipleRoutes(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "multi-route-a.ts", `
		bus.on("hello", function(msg) { msg.reply({route: "a"}); });
	`)
	testutil.Deploy(t, k, "multi-route-b.ts", `
		bus.on("world", function(msg) { msg.reply({route: "b"}); });
	`)

	gw.Handle("GET", "/route-a", "ts.multi-route-a.hello")
	gw.Handle("GET", "/route-b", "ts.multi-route-b.world")

	status1, body1 := gwGet(t, gw, "/route-a")
	assert.Equal(t, 200, status1)
	assert.Contains(t, body1, `"route":"a"`)

	status2, body2 := gwGet(t, gw, "/route-b")
	assert.Equal(t, 200, status2)
	assert.Contains(t, body2, `"route":"b"`)
}

// testAdvRouteReplacement — adding same route replaces handler.
func testAdvRouteReplacement(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "v1.ts", `bus.on("ver", function(msg) { msg.reply({v: 1}); });`)
	testutil.Deploy(t, k, "v2.ts", `bus.on("ver", function(msg) { msg.reply({v: 2}); });`)

	gw.Handle("GET", "/version", "ts.v1.ver")
	// Replace with v2
	gw.Handle("GET", "/version", "ts.v2.ver")

	_, body := gwGet(t, gw, "/version")
	assert.Contains(t, body, `"v":2`)
}

// testAdvConcurrentRequests — 20 concurrent HTTP requests.
func testAdvConcurrentRequests(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "concurrent-gw.ts", `
		bus.on("req", function(msg) { msg.reply({ok: true}); });
	`)
	gw.Handle("GET", "/concurrent", "ts.concurrent-gw.req")

	var wg sync.WaitGroup
	var succeeded atomic.Int64
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("http://" + gw.Addr() + "/concurrent")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					succeeded.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	t.Logf("20 concurrent requests: %d succeeded", succeeded.Load())
	assert.Greater(t, succeeded.Load(), int64(10))
}

// testAdvLargeResponse — handler returns large JSON.
func testAdvLargeResponse(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "large-resp.ts", `
		bus.on("big", function(msg) {
			var data = "x".repeat(100000);
			msg.reply({data: data, size: data.length});
		});
	`)
	gw.Handle("GET", "/large-response", "ts.large-resp.big")

	resp, err := http.Get("http://" + gw.Addr() + "/large-response")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 200, resp.StatusCode)

	var parsed struct{ Size int `json:"size"` }
	json.Unmarshal(body, &parsed)
	assert.Equal(t, 100000, parsed.Size)
}

// testAdvHealthDuringRequests — health stays ok during active HTTP traffic.
func testAdvHealthDuringRequests(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "health-gw.ts", `bus.on("ping", function(msg) { msg.reply({ok:true}); });`)
	gw.Handle("GET", "/health-ping", "ts.health-gw.ping")

	// Fire requests
	for i := 0; i < 10; i++ {
		gwGet(t, gw, "/health-ping")
	}

	// Health should still be good
	status, body := gwGet(t, gw, "/healthz")
	assert.Equal(t, 200, status)
	assert.Equal(t, "ok", body)
}
