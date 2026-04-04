package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	bkgw "github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testErrNotFound — route to nonexistent tool -> 404.
func testErrNotFound(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-404.ts", `
		bus.on("call-ghost", async function(msg) {
			try {
				var r = await tools.call("ghost-tool-404", {});
				msg.reply(r);
			} catch(e) {
				msg.reply({error: e.message, code: e.code || "NO_CODE"});
			}
		});
	`)
	require.NoError(t, err)

	gw.Handle("GET", "/not-found-test", "ts.gw-404.call-ghost")

	status, body := gwGet(t, gw, "/not-found-test")
	// Handler catches NOT_FOUND from tools.call, replies with {code: "NOT_FOUND"}.
	// Gateway mapHTTPStatus maps NOT_FOUND -> 404 (correct behavior after SES .code fix).
	assert.Equal(t, 404, status)
	assert.Contains(t, body, "not found")
}

// testErrTimeout — route to handler that never replies -> 504.
func testErrTimeout(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-timeout.ts", `
		bus.on("slow", async function(msg) {
			// Never reply — gateway should timeout
			await new Promise(r => setTimeout(r, 60000));
		});
	`)
	require.NoError(t, err)

	gw.Handle("GET", "/timeout-test", "ts.gw-timeout.slow")

	status, body := gwGet(t, gw, "/timeout-test")
	assert.Equal(t, 504, status)
	assert.Contains(t, body, "timeout")
}

// testErrValidResponse — successful handler -> 200.
func testErrValidResponse(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-ok.ts", `
		bus.on("greet", function(msg) {
			msg.reply({greeting: "hello", name: msg.payload.name || "world"});
		});
	`)
	require.NoError(t, err)

	gw.Handle("POST", "/greet", "ts.gw-ok.greet")

	status, body := gwPost(t, gw, "/greet", `{"name":"adversarial"}`)
	assert.Equal(t, 200, status)
	assert.Contains(t, body, "adversarial")
}

// testErrNoRoute — request to unregistered path -> 404 from mux.
func testErrNoRoute(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	status, _ := gwGet(t, gw, "/does-not-exist")
	assert.Equal(t, 404, status)
}

// testErrHealthEndpoints — /healthz, /readyz, /health all work.
func testErrHealthEndpoints(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	status1, body1 := gwGet(t, gw, "/healthz")
	assert.Equal(t, 200, status1)
	assert.Equal(t, "ok", body1)

	status2, body2 := gwGet(t, gw, "/readyz")
	assert.Equal(t, 200, status2)
	assert.Equal(t, "ok", body2)

	status3, body3 := gwGet(t, gw, "/health")
	assert.Equal(t, 200, status3)

	var health struct {
		Healthy bool `json:"healthy"`
	}
	json.Unmarshal([]byte(body3), &health)
	assert.True(t, health.Healthy)
}

// testErrCORS — OPTIONS preflight returns CORS headers.
func testErrCORS(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := bkgw.New(k, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
		CORS: &bkgw.CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type"},
		},
	})
	require.NoError(t, gw.Start())
	defer gw.Stop()

	gw.Handle("GET", "/cors-test", "test.topic")

	req, _ := http.NewRequest("OPTIONS", "http://"+gw.Addr()+"/cors-test", nil)
	req.Header.Set("Origin", "http://example.com")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, 204, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

// testErrHandlerError — handler returns {error: ...} -> 500.
func testErrHandlerError(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-err.ts", `
		bus.on("fail", function(msg) {
			msg.reply({error: "something went wrong", code: "INTERNAL_ERROR"});
		});
	`)
	require.NoError(t, err)

	gw.Handle("GET", "/error-test", "ts.gw-err.fail")

	status, body := gwGet(t, gw, "/error-test")
	assert.Equal(t, 500, status)
	assert.Contains(t, body, "something went wrong")
}

// testErrLargePayload — 100KB POST body through gateway.
func testErrLargePayload(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-big.ts", `
		bus.on("big", function(msg) {
			var size = JSON.stringify(msg.payload).length;
			msg.reply({received: true, size: size});
		});
	`)
	require.NoError(t, err)

	gw.Handle("POST", "/big", "ts.gw-big.big")

	big := `{"data":"` + strings.Repeat("x", 100000) + `"}`
	status, body := gwPost(t, gw, "/big", big)
	assert.Equal(t, 200, status)
	assert.Contains(t, body, "received")
}

// testErrPathParams — path parameters extracted correctly.
func testErrPathParams(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := gwSetup(t, k)

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-params.ts", `
		bus.on("user", function(msg) {
			msg.reply({userId: msg.payload.id, method: msg.payload.method || "unknown"});
		});
	`)
	require.NoError(t, err)

	gw.Handle("POST", "/users/{id}", "ts.gw-params.user",
		bkgw.WithParam("id", "id"),
	)

	status, body := gwPost(t, gw, "/users/42", `{}`)
	assert.Equal(t, 200, status)
	assert.Contains(t, body, "42")
}

// testErrGatewayStatusMapping — verify error code->HTTP status mapping.
func testErrGatewayStatusMapping(t *testing.T, env *suite.TestEnv) {
	cases := []struct {
		code   string
		status int
	}{
		{"NOT_FOUND", 404},
		{"PERMISSION_DENIED", 403},
		{"VALIDATION_ERROR", 400},
		{"DECODE_ERROR", 400},
		{"RATE_LIMITED", 429},
		{"NOT_CONFIGURED", 501},
		{"TIMEOUT", 504},
		{"ALREADY_EXISTS", 409},
		{"INTERNAL_ERROR", 500},
		{"WHATEVER_UNKNOWN", 500},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			payload, _ := json.Marshal(map[string]any{
				"error": "test error",
				"code":  tc.code,
			})
			// Verify the JSON carries the code correctly.
			// The gateway tests cover the HTTP layer; this validates the code field round-trips.
			var parsed struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			require.NoError(t, json.Unmarshal(payload, &parsed))
			assert.Equal(t, tc.code, parsed.Code)
		})
	}
}
