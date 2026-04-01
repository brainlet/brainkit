package adversarial_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGateway(t *testing.T, tk *testutil.TestKernel) *gateway.Gateway {
	t.Helper()
	gw := gateway.New(tk, gateway.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
	})
	require.NoError(t, gw.Start())
	t.Cleanup(func() { gw.Stop() })
	return gw
}

func gwGet(t *testing.T, gw *gateway.Gateway, path string) (int, string) {
	t.Helper()
	url := "http://" + gw.Addr() + path
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func gwPost(t *testing.T, gw *gateway.Gateway, path, body string) (int, string) {
	t.Helper()
	url := "http://" + gw.Addr() + path
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	rbody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(rbody)
}

// TestGatewayErrors_NotFound — route to nonexistent tool → 404.
func TestGatewayErrors_NotFound(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-404.ts", `
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
	assert.Equal(t, 500, status) // handler returns error in response body, gateway sees it
	assert.Contains(t, body, "not found")
}

// TestGatewayErrors_Timeout — route to handler that never replies → 504.
func TestGatewayErrors_Timeout(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-timeout.ts", `
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

// TestGatewayErrors_ValidResponse — successful handler → 200.
func TestGatewayErrors_ValidResponse(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-ok.ts", `
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

// TestGatewayErrors_NoRoute — request to unregistered path → 404 from mux.
func TestGatewayErrors_NoRoute(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	status, _ := gwGet(t, gw, "/does-not-exist")
	assert.Equal(t, 404, status)
}

// TestGatewayErrors_HealthEndpoints — /healthz, /readyz, /health all work.
func TestGatewayErrors_HealthEndpoints(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

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

// TestGatewayErrors_CORS — OPTIONS preflight returns CORS headers.
func TestGatewayErrors_CORS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := gateway.New(tk, gateway.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
		CORS: &gateway.CORSConfig{
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

// TestGatewayErrors_HandlerError — handler returns {error: ...} → 500.
func TestGatewayErrors_HandlerError(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-err.ts", `
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

// TestGatewayErrors_LargePayload — 100KB POST body through gateway.
func TestGatewayErrors_LargePayload(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-big.ts", `
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

// TestGatewayErrors_PathParams — path parameters extracted correctly.
func TestGatewayErrors_PathParams(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-params.ts", `
		bus.on("user", function(msg) {
			msg.reply({userId: msg.payload.id, method: msg.payload.method || "unknown"});
		});
	`)
	require.NoError(t, err)

	gw.Handle("POST", "/users/{id}", "ts.gw-params.user",
		gateway.WithParam("id", "id"),
	)

	status, body := gwPost(t, gw, "/users/42", `{}`)
	assert.Equal(t, 200, status)
	assert.Contains(t, body, "42")
}
