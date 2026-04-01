package adversarial_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// GATEWAY HTTP ATTACKS
// Forge HTTP requests to exploit the gateway's routing, CORS, and status mapping.
// ════════════════════════════════════════════════════════════════════════════

// Attack: inject headers to forge callerID
func TestGatewayAttack_HeaderInjection(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-header.ts", `
		bus.on("whoami", function(msg) {
			msg.reply({callerId: msg.callerId, topic: msg.topic});
		});
	`)
	require.NoError(t, err)
	gw.Handle("POST", "/whoami", "ts.gw-header.whoami")

	// Forge a request with a fake X-Caller-ID header
	req, _ := http.NewRequest("POST", "http://"+gw.Addr()+"/whoami", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "admin") // try to impersonate
	req.Header.Set("X-Correlation-ID", "forged-correlation")
	req.Header.Set("X-Reply-To", "evil.reply.topic")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// The callerId should be from the kernel's callerID, not the forged header
	assert.NotContains(t, string(body), `"callerId":"admin"`, "forged header should not set callerID")
}

// Attack: request body with __proto__ pollution
func TestGatewayAttack_ProtoPollutionViaHTTP(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-proto.ts", `
		bus.on("check", function(msg) {
			msg.reply({
				hasProto: msg.payload.__proto__ !== undefined,
				hasPollution: ({}).polluted === true,
				raw: msg.payload,
			});
		});
	`)
	require.NoError(t, err)
	gw.Handle("POST", "/proto-check", "ts.gw-proto.check")

	// Send proto pollution payload via HTTP
	evilPayload := `{"__proto__":{"polluted":true},"constructor":{"prototype":{"pwned":true}},"data":"test"}`
	status, body := gwPost(t, gw, "/proto-check", evilPayload)
	assert.Equal(t, 200, status)
	assert.NotContains(t, body, `"hasPollution":true`, "proto pollution via HTTP should not work")
}

// Attack: massive request body
func TestGatewayAttack_RequestBodyBomb(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-bomb.ts", `
		bus.on("echo", function(msg) {
			msg.reply({size: JSON.stringify(msg.payload).length});
		});
	`)
	require.NoError(t, err)
	gw.Handle("POST", "/bomb", "ts.gw-bomb.echo")

	// 10MB request body
	bigBody := `{"data":"` + strings.Repeat("A", 10*1024*1024) + `"}`
	resp, err := http.Post("http://"+gw.Addr()+"/bomb", "application/json", strings.NewReader(bigBody))
	if err != nil {
		return // connection refused/reset is fine
	}
	defer resp.Body.Close()
	// Either 200 (processed) or 4xx/5xx (rejected) — no crash
	assert.True(t, tk.Alive(ctx), "kernel should survive 10MB HTTP body")
}

// Attack: path traversal in URL parameters
func TestGatewayAttack_PathTraversalParams(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-params.ts", `
		bus.on("get", function(msg) {
			msg.reply({id: msg.payload.id});
		});
	`)
	require.NoError(t, err)
	gw.Handle("GET", "/items/{id}", "ts.gw-params.get", gateway.WithParam("id", "id"))

	// Path traversal in URL param
	evilPaths := []string{
		"/items/../../../etc/passwd",
		"/items/..%2F..%2F..%2Fetc%2Fpasswd",
		"/items/foo%00bar",
		"/items/<script>alert(1)</script>",
		"/items/' OR 1=1 --",
	}

	for _, path := range evilPaths {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get("http://" + gw.Addr() + path)
			if err != nil {
				return
			}
			resp.Body.Close()
			// 404 is expected (path doesn't match the route pattern)
			// Key: no crash, no panic
		})
	}
	assert.True(t, tk.Alive(ctx))
}

// Attack: HTTP method confusion
func TestGatewayAttack_MethodConfusion(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-method.ts", `
		bus.on("data", function(msg) { msg.reply({ok: true}); });
	`)
	require.NoError(t, err)
	gw.Handle("POST", "/api/data", "ts.gw-method.data")

	// Try GET on a POST-only route
	status, _ := gwGet(t, gw, "/api/data")
	assert.Equal(t, 404, status, "GET on POST route should 404")

	// Try DELETE, PUT, PATCH
	for _, method := range []string{"DELETE", "PUT", "PATCH"} {
		req, _ := http.NewRequest(method, "http://"+gw.Addr()+"/api/data", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		assert.Equal(t, 404, resp.StatusCode, method+" on POST route should 404")
	}
}

// Attack: concurrent gateway requests hitting the same handler
func TestGatewayAttack_ConcurrentFlood(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-flood.ts", `
		var count = 0;
		bus.on("req", function(msg) {
			count++;
			msg.reply({count: count});
		});
	`)
	require.NoError(t, err)
	gw.Handle("GET", "/flood", "ts.gw-flood.req")

	// 100 concurrent requests
	var wg sync.WaitGroup
	var succeeded atomic.Int64
	var failed atomic.Int64

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("http://" + gw.Addr() + "/flood")
			if err != nil {
				failed.Add(1)
				return
			}
			resp.Body.Close()
			if resp.StatusCode == 200 {
				succeeded.Add(1)
			} else {
				failed.Add(1)
			}
		}()
	}
	wg.Wait()

	t.Logf("100 concurrent: %d succeeded, %d failed", succeeded.Load(), failed.Load())
	assert.Greater(t, succeeded.Load(), int64(0), "some requests should succeed")
	assert.True(t, tk.Alive(ctx))
}

// Attack: SSE client disconnects mid-stream — does the handler hang?
func TestGatewayAttack_SSEClientDisconnect(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-sse.ts", `
		bus.on("stream", function(msg) {
			for (var i = 0; i < 100; i++) {
				msg.stream.text("chunk-" + i);
			}
			msg.stream.end({done: true});
		});
	`)
	require.NoError(t, err)
	gw.HandleStream("GET", "/sse-disconnect", "ts.gw-sse.stream")

	// Connect, read 2 chunks, disconnect
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + gw.Addr() + "/sse-disconnect")
	if err != nil {
		return // timeout is fine
	}
	// Read just one line and abort
	buf := make([]byte, 100)
	resp.Body.Read(buf)
	resp.Body.Close() // DISCONNECT

	time.Sleep(1 * time.Second)
	assert.True(t, tk.Alive(ctx), "kernel should survive SSE client disconnect")
}

// Attack: WebSocket message injection
func TestGatewayAttack_WebSocketInjection(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-ws.ts", `
		bus.on("ws", function(msg) {
			msg.reply({received: msg.payload});
		});
	`)
	require.NoError(t, err)
	gw.HandleWebSocket("/ws-attack", "ts.gw-ws.ws")

	// We can't easily test WebSocket without a WS client, but verify the endpoint exists
	// and doesn't crash on a non-upgrade request
	resp, err := http.Get("http://" + gw.Addr() + "/ws-attack")
	if err != nil {
		return
	}
	resp.Body.Close()
	// Should get 400 or similar (not a WS upgrade request)
	assert.True(t, tk.Alive(ctx))
}

// Attack: CORS bypass — origin not in allowlist
func TestGatewayAttack_CORSBypass(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := gateway.New(tk, gateway.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
		CORS: &gateway.CORSConfig{
			AllowOrigins: []string{"https://legit.example.com"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type"},
		},
	})
	require.NoError(t, gw.Start())
	defer gw.Stop()

	ctx := context.Background()
	_, _ = tk.Deploy(ctx, "gw-cors.ts", `bus.on("api", function(msg) { msg.reply({ok:true}); });`)
	gw.Handle("GET", "/cors-api", "ts.gw-cors.api")

	// Request with evil origin
	req, _ := http.NewRequest("OPTIONS", "http://"+gw.Addr()+"/cors-api", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should NOT have the evil origin in Access-Control-Allow-Origin
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "https://evil.example.com", allowOrigin, "CORS should not allow evil origin")
}

// Attack: error response leaks internal information
func TestGatewayAttack_ErrorInfoLeak(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-err-leak.ts", `
		bus.on("crash", function(msg) {
			// Throw with internal details
			throw new Error("internal: connection to postgres://admin:password@db:5432/prod failed");
		});
	`)
	require.NoError(t, err)
	gw.Handle("GET", "/crash", "ts.gw-err-leak.crash")

	status, body := gwGet(t, gw, "/crash")
	// The error message might contain internal details
	if status == 500 && strings.Contains(body, "postgres://") {
		t.Logf("FINDING: error response leaks connection string: %s", body[:min(200, len(body))])
	}
	if strings.Contains(body, "password") {
		t.Logf("FINDING: error response leaks password")
	}
}

// Attack: slowloris — keep the connection open with slow writes
func TestGatewayAttack_Slowloris(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, _ = tk.Deploy(ctx, "gw-slow.ts", `bus.on("api", function(msg) { msg.reply({ok:true}); });`)
	gw.Handle("POST", "/slow-api", "ts.gw-slow.api")

	// Start a request with a very slow body
	pr, pw := io.Pipe()
	go func() {
		// Write one byte per second
		for i := 0; i < 5; i++ {
			pw.Write([]byte("{"))
			time.Sleep(100 * time.Millisecond)
		}
		pw.Write([]byte(`"data":"test"}`))
		pw.Close()
	}()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post("http://"+gw.Addr()+"/slow-api", "application/json", pr)
	if err != nil {
		return // timeout is OK
	}
	resp.Body.Close()

	// Gateway should still serve other requests while this slow one is in progress
	status, _ := gwGet(t, gw, "/healthz")
	assert.Equal(t, 200, status, "gateway should serve health during slow request")
}

// Attack: gateway route removal via bus — can any .ts deployment remove routes?
func TestGatewayAttack_RouteRemovalViaBus(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, _ = tk.Deploy(ctx, "gw-protected.ts", `bus.on("api", function(msg) { msg.reply({protected: true}); });`)
	gw.Handle("GET", "/protected", "ts.gw-protected.api")

	// Attacker tries to remove the route via bus
	_, err := tk.Deploy(ctx, "gw-attacker.ts", `
		var r = bus.publish("gateway.http.route.remove", {method: "GET", path: "/protected"});
		output({replyTo: r.replyTo});
	`)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Check if route still works
	status, body := gwGet(t, gw, "/protected")
	if status == 404 {
		t.Logf("FINDING: attacker deployment removed a gateway route via bus")
	} else {
		assert.Equal(t, 200, status)
		assert.Contains(t, body, "protected")
	}
}
