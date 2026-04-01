package adversarial_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGatewayAdvanced_SSEStreaming — SSE endpoint streams typed events.
func TestGatewayAdvanced_SSEStreaming(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "sse-handler.ts", `
		bus.on("stream", function(msg) {
			msg.stream.text("hello");
			msg.stream.text("world");
			msg.stream.progress(50, "half");
			msg.stream.end({done: true});
		});
	`)
	require.NoError(t, err)

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

// TestGatewayAdvanced_WebhookDelivery — webhook endpoint accepts POST.
func TestGatewayAdvanced_WebhookDelivery(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "webhook-handler.ts", `
		bus.on("webhook", function(msg) {
			// Webhook doesn't reply — fire and forget
		});
	`)
	require.NoError(t, err)

	gw.HandleWebhook("POST", "/webhook", "ts.webhook-handler.webhook")

	status, _ := gwPost(t, gw, "/webhook", `{"event":"push","repo":"test"}`)
	assert.Equal(t, 200, status)
}

// TestGatewayAdvanced_MultipleRoutes — multiple routes coexist.
func TestGatewayAdvanced_MultipleRoutes(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "multi-route-a.ts", `
		bus.on("hello", function(msg) { msg.reply({route: "a"}); });
	`)
	require.NoError(t, err)
	_, err = tk.Deploy(ctx, "multi-route-b.ts", `
		bus.on("world", function(msg) { msg.reply({route: "b"}); });
	`)
	require.NoError(t, err)

	gw.Handle("GET", "/route-a", "ts.multi-route-a.hello")
	gw.Handle("GET", "/route-b", "ts.multi-route-b.world")

	status1, body1 := gwGet(t, gw, "/route-a")
	assert.Equal(t, 200, status1)
	assert.Contains(t, body1, `"route":"a"`)

	status2, body2 := gwGet(t, gw, "/route-b")
	assert.Equal(t, 200, status2)
	assert.Contains(t, body2, `"route":"b"`)
}

// TestGatewayAdvanced_RouteReplacement — adding same route replaces handler.
func TestGatewayAdvanced_RouteReplacement(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, _ = tk.Deploy(ctx, "v1.ts", `bus.on("ver", function(msg) { msg.reply({v: 1}); });`)
	_, _ = tk.Deploy(ctx, "v2.ts", `bus.on("ver", function(msg) { msg.reply({v: 2}); });`)

	gw.Handle("GET", "/version", "ts.v1.ver")
	// Replace with v2
	gw.Handle("GET", "/version", "ts.v2.ver")

	_, body := gwGet(t, gw, "/version")
	assert.Contains(t, body, `"v":2`)
}

// TestGatewayAdvanced_ConcurrentRequests — 20 concurrent HTTP requests.
func TestGatewayAdvanced_ConcurrentRequests(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "concurrent-gw.ts", `
		bus.on("req", function(msg) { msg.reply({ok: true}); });
	`)
	require.NoError(t, err)
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

// TestGatewayAdvanced_LargeResponse — handler returns large JSON.
func TestGatewayAdvanced_LargeResponse(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "large-resp.ts", `
		bus.on("big", function(msg) {
			var data = "x".repeat(100000);
			msg.reply({data: data, size: data.length});
		});
	`)
	require.NoError(t, err)
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

// TestGatewayAdvanced_HealthDuringRequests — health stays ok during active HTTP traffic.
func TestGatewayAdvanced_HealthDuringRequests(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, _ = tk.Deploy(ctx, "health-gw.ts", `bus.on("ping", function(msg) { msg.reply({ok:true}); });`)
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
