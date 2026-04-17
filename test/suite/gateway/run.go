// Package gateway provides the gateway domain test suite.
// All test functions take *suite.TestEnv and are registered via Run().
// The standalone gateway_test.go creates a Full env for the memory fast path.
package gateway

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	bkgw "github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// Run executes all gateway domain tests against the given environment.
func Run(t *testing.T, env *suite.TestEnv) {
	t.Run("gateway", func(t *testing.T) {
		// routes.go — core gateway routes (from infra/gateway_test.go)
		t.Run("request_response_e2e", func(t *testing.T) { testRequestResponseE2E(t, env) })
		t.Run("timeout_504", func(t *testing.T) { testTimeout504(t, env) })
		t.Run("webhook", func(t *testing.T) { testWebhook(t, env) })
		t.Run("drain_returns_503", func(t *testing.T) { testDrainReturns503(t, env) })
		t.Run("drain_allows_webhooks", func(t *testing.T) { testDrainAllowsWebhooks(t, env) })
		t.Run("not_found", func(t *testing.T) { testNotFound(t, env) })
		t.Run("cors_preflight", func(t *testing.T) { testCORSPreflight(t, env) })
		t.Run("with_http_context", func(t *testing.T) { testWithHTTPContext(t, env) })
		t.Run("path_params", func(t *testing.T) { testPathParams(t, env) })
		t.Run("route_table", func(t *testing.T) { testRouteTable(t, env) })
		t.Run("bus_route_add", func(t *testing.T) { testBusRouteAdd(t, env) })
		t.Run("bus_route_remove_by_owner", func(t *testing.T) { testBusRouteRemoveByOwner(t, env) })
		t.Run("health_endpoints", func(t *testing.T) { testHealthEndpoints(t, env) })
		t.Run("readyz_during_drain", func(t *testing.T) { testReadyzDuringDrain(t, env) })
		t.Run("sse_streaming", func(t *testing.T) { testSSEStreaming(t, env) })
		t.Run("sse_progress_and_events", func(t *testing.T) { testSSEProgressAndEvents(t, env) })
		t.Run("sse_error_terminates", func(t *testing.T) { testSSEErrorTerminates(t, env) })
		t.Run("error_response_500", func(t *testing.T) { testErrorResponse500(t, env) })
		t.Run("route_replacement", func(t *testing.T) { testRouteReplacement(t, env) })
		t.Run("bus_route_list", func(t *testing.T) { testBusRouteList(t, env) })
		t.Run("bus_status", func(t *testing.T) { testBusStatus(t, env) })
		t.Run("websocket", func(t *testing.T) { testWebSocket(t, env) })
		t.Run("rate_limiting", func(t *testing.T) { testRateLimiting(t, env) })

		// stream.go — streaming tests (from infra/stream_test.go)
		t.Run("stream_heartbeat_timeout", func(t *testing.T) { testStreamHeartbeatTimeout(t, env) })
		t.Run("stream_max_duration", func(t *testing.T) { testStreamMaxDuration(t, env) })
		t.Run("stream_max_events", func(t *testing.T) { testStreamMaxEvents(t, env) })
		t.Run("stream_keepalive_comments", func(t *testing.T) { testStreamKeepaliveComments(t, env) })
		t.Run("stream_reconnection", func(t *testing.T) { testStreamReconnection(t, env) })
		t.Run("stream_session_expired", func(t *testing.T) { testStreamSessionExpired(t, env) })
		t.Run("stream_concurrent", func(t *testing.T) { testStreamConcurrent(t, env) })
		t.Run("stream_gateway_shutdown", func(t *testing.T) { testStreamGatewayShutdown(t, env) })

		// advanced.go — adversarial advanced tests (from adversarial/gateway_advanced_test.go)
		t.Run("adv_sse_streaming", func(t *testing.T) { testAdvSSEStreaming(t, env) })
		t.Run("adv_webhook_delivery", func(t *testing.T) { testAdvWebhookDelivery(t, env) })
		t.Run("adv_multiple_routes", func(t *testing.T) { testAdvMultipleRoutes(t, env) })
		t.Run("adv_route_replacement", func(t *testing.T) { testAdvRouteReplacement(t, env) })
		t.Run("adv_concurrent_requests", func(t *testing.T) { testAdvConcurrentRequests(t, env) })
		t.Run("adv_large_response", func(t *testing.T) { testAdvLargeResponse(t, env) })
		t.Run("adv_health_during_requests", func(t *testing.T) { testAdvHealthDuringRequests(t, env) })

		// errors.go — adversarial error tests (from adversarial/gateway_errors_test.go)
		t.Run("err_not_found", func(t *testing.T) { testErrNotFound(t, env) })
		t.Run("err_timeout", func(t *testing.T) { testErrTimeout(t, env) })
		t.Run("err_valid_response", func(t *testing.T) { testErrValidResponse(t, env) })
		t.Run("err_no_route", func(t *testing.T) { testErrNoRoute(t, env) })
		t.Run("err_health_endpoints", func(t *testing.T) { testErrHealthEndpoints(t, env) })
		t.Run("err_cors", func(t *testing.T) { testErrCORS(t, env) })
		t.Run("err_handler_error", func(t *testing.T) { testErrHandlerError(t, env) })
		t.Run("err_large_payload", func(t *testing.T) { testErrLargePayload(t, env) })
		t.Run("err_path_params", func(t *testing.T) { testErrPathParams(t, env) })
		t.Run("err_gateway_status_mapping", func(t *testing.T) { testErrGatewayStatusMapping(t, env) })

		// attacks.go — gateway-specific attack tests (from adversarial/gateway_attack_test.go)
		t.Run("attack_request_body_bomb", func(t *testing.T) { testAttackRequestBodyBomb(t, env) })
		t.Run("attack_method_confusion", func(t *testing.T) { testAttackMethodConfusion(t, env) })
		t.Run("attack_concurrent_flood", func(t *testing.T) { testAttackConcurrentFlood(t, env) })
		t.Run("attack_sse_client_disconnect", func(t *testing.T) { testAttackSSEClientDisconnect(t, env) })
		t.Run("attack_cors_bypass", func(t *testing.T) { testAttackCORSBypass(t, env) })
		t.Run("attack_error_info_leak", func(t *testing.T) { testAttackErrorInfoLeak(t, env) })
		t.Run("attack_slowloris", func(t *testing.T) { testAttackSlowloris(t, env) })
		t.Run("attack_route_removal_via_bus", func(t *testing.T) { testAttackRouteRemovalViaBus(t, env) })
	})
}

// gwStart starts a gateway on a random port with optional config modifications.
// Waits for the gateway to be ready (healthz returns 200) before returning.
func gwStart(t *testing.T, k *brainkit.Kit, opts ...func(*bkgw.Config)) (*bkgw.Gateway, string) {
	t.Helper()
	cfg := bkgw.Config{Listen: "127.0.0.1:0", Timeout: 5 * time.Second}
	for _, opt := range opts {
		opt(&cfg)
	}
	gw := bkgw.New(k, cfg)
	require.NoError(t, gw.Start())
	t.Cleanup(func() { gw.Stop() })
	addr := "http://" + gw.Addr()
	gwWaitReady(t, addr)
	return gw, addr
}

// gwStartWithStream starts a gateway with streaming configuration.
func gwStartWithStream(t *testing.T, k *brainkit.Kit, streamCfg *bkgw.StreamConfig) (*bkgw.Gateway, string) {
	t.Helper()
	gw := bkgw.New(k, bkgw.Config{
		Listen:  "127.0.0.1:0",
		Timeout: 5 * time.Second,
		Stream:  streamCfg,
	})
	require.NoError(t, gw.Start())
	t.Cleanup(func() { gw.Stop() })
	addr := "http://" + gw.Addr()
	gwWaitReady(t, addr)
	return gw, addr
}

// gwSetup creates a simple gateway with short timeout for adversarial tests.
func gwSetup(t *testing.T, k *brainkit.Kit) *bkgw.Gateway {
	t.Helper()
	gw := bkgw.New(k, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
	})
	require.NoError(t, gw.Start())
	t.Cleanup(func() { gw.Stop() })
	gwWaitReady(t, "http://"+gw.Addr())
	return gw
}

// ── Reliability helpers ──────────────────────────────────────────────────────

// gwWaitReady waits for the gateway HTTP server to accept connections.
// Polls any path — a response (even 404) means the server is up.
func gwWaitReady(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(addr + "/__ready_probe")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("gateway not accepting connections after 5s")
}

// gwWaitForStatus polls a path until the expected status code is returned.
// Used after deploy (wait for non-404) and after SetDraining (wait for 503).
func gwWaitForStatus(t *testing.T, method, url string, expected int) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var resp *http.Response
		var err error
		if method == "POST" {
			resp, err = client.Post(url, "application/json", strings.NewReader("{}"))
		} else {
			resp, err = client.Get(url)
		}
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == expected {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected status %d at %s after 5s", expected, url)
}

// gwWaitForDeploy deploys .ts code and waits for the gateway route to return non-404.
func gwWaitForDeploy(t *testing.T, kit *brainkit.Kit, addr, source, code, method, path string) {
	t.Helper()
	testutil.Deploy(t, kit, source, code)
	gwWaitForStatus(t, method, addr+path, 200)
}

// gwGet performs an HTTP GET and returns status code + body string.
// Replicates gwGet from adversarial/gateway_errors_test.go.
func gwGet(t *testing.T, gw *bkgw.Gateway, path string) (int, string) {
	t.Helper()
	url := "http://" + gw.Addr() + path
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

// gwPost performs an HTTP POST and returns status code + body string.
// Replicates gwPost from adversarial/gateway_errors_test.go.
func gwPost(t *testing.T, gw *bkgw.Gateway, path, body string) (int, string) {
	t.Helper()
	url := "http://" + gw.Addr() + path
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	rbody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(rbody)
}

// readSSEContent reads SSE body until connection closes or timeout.
func readSSEContent(t *testing.T, resp *http.Response, timeout time.Duration) string {
	t.Helper()
	done := make(chan string, 1)
	go func() {
		body, _ := io.ReadAll(resp.Body)
		done <- string(body)
	}()
	select {
	case content := <-done:
		return content
	case <-time.After(timeout):
		resp.Body.Close()
		return <-done
	}
}
