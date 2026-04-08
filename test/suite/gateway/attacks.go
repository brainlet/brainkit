package gateway

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bkgw "github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAttackRequestBodyBomb — massive request body (10MB).
func testAttackRequestBodyBomb(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "gw-bomb.ts", `
		bus.on("echo", function(msg) {
			msg.reply({size: JSON.stringify(msg.payload).length});
		});
	`)
	gw.Handle("POST", "/bomb", "ts.gw-bomb.echo")

	// 10MB request body
	bigBody := `{"data":"` + strings.Repeat("A", 10*1024*1024) + `"}`
	resp, err := http.Post("http://"+gw.Addr()+"/bomb", "application/json", strings.NewReader(bigBody))
	if err != nil {
		return // connection refused/reset is fine
	}
	defer resp.Body.Close()
	// Either 200 (processed) or 4xx/5xx (rejected) — no crash
	assert.True(t, testutil.Alive(t, k), "kernel should survive 10MB HTTP body")
}

// testAttackMethodConfusion — HTTP method confusion on POST-only route.
func testAttackMethodConfusion(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "gw-method.ts", `
		bus.on("data", function(msg) { msg.reply({ok: true}); });
	`)
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

// testAttackConcurrentFlood — 100 concurrent requests hitting the same handler.
func testAttackConcurrentFlood(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "gw-flood.ts", `
		var count = 0;
		bus.on("req", function(msg) {
			count++;
			msg.reply({count: count});
		});
	`)
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
	assert.True(t, testutil.Alive(t, k))
}

// testAttackSSEClientDisconnect — SSE client disconnects mid-stream.
func testAttackSSEClientDisconnect(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "gw-sse.ts", `
		bus.on("stream", function(msg) {
			for (var i = 0; i < 100; i++) {
				msg.stream.text("chunk-" + i);
			}
			msg.stream.end({done: true});
		});
	`)
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
	assert.True(t, testutil.Alive(t, k), "kernel should survive SSE client disconnect")
}

// testAttackCORSBypass — CORS bypass with origin not in allowlist.
func testAttackCORSBypass(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := bkgw.New(k, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
		CORS: &bkgw.CORSConfig{
			AllowOrigins: []string{"https://legit.example.com"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type"},
		},
	})
	require.NoError(t, gw.Start())
	defer gw.Stop()

	testutil.Deploy(t, k, "gw-cors.ts", `bus.on("api", function(msg) { msg.reply({ok:true}); });`)
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

// testAttackErrorInfoLeak — error response leaks internal information.
func testAttackErrorInfoLeak(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "gw-err-leak.ts", `
		bus.on("crash", function(msg) {
			// Throw with internal details
			throw new Error("internal: connection to postgres://admin:password@db:5432/prod failed");
		});
	`)
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

// testAttackSlowloris — keep the connection open with slow writes.
func testAttackSlowloris(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "gw-slow.ts", `bus.on("api", function(msg) { msg.reply({ok:true}); });`)
	gw.Handle("POST", "/slow-api", "ts.gw-slow.api")

	// Start a slow request in a goroutine
	pr, pw := io.Pipe()
	go func() {
		for i := 0; i < 10; i++ {
			pw.Write([]byte("{"))
			time.Sleep(200 * time.Millisecond)
		}
		pw.Write([]byte(`"data":"test"}`))
		pw.Close()
	}()

	// Fire the slow POST in background — it blocks on the pipe writer
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post("http://"+gw.Addr()+"/slow-api", "application/json", pr)
		if err == nil {
			resp.Body.Close()
		}
	}()

	// Wait for the slow request to be in-flight, then check health
	time.Sleep(300 * time.Millisecond)
	status, _ := gwGet(t, gw, "/healthz")
	assert.Equal(t, 200, status, "gateway should serve health during slow request")
}

// testAttackRouteRemovalViaBus — can any .ts deployment remove routes?
func testAttackRouteRemovalViaBus(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	gw := gwSetup(t, k)

	testutil.Deploy(t, k, "gw-protected.ts", `bus.on("api", function(msg) { msg.reply({protected: true}); });`)
	gw.Handle("GET", "/protected", "ts.gw-protected.api")

	// Attacker tries to remove the route via bus
	testutil.Deploy(t, k, "gw-attacker.ts", `
		var r = bus.publish("gateway.http.route.remove", {method: "GET", path: "/protected"});
		output({replyTo: r.replyTo});
	`)

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
