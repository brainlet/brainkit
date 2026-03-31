package infra_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startGateway(t *testing.T, k *testutil.TestKernel, opts ...func(*gateway.Config)) (*gateway.Gateway, string) {
	t.Helper()
	cfg := gateway.Config{Listen: "127.0.0.1:0", Timeout: 5 * time.Second}
	for _, opt := range opts {
		opt(&cfg)
	}
	gw := gateway.New(k, cfg)
	require.NoError(t, gw.Start())
	t.Cleanup(func() { gw.Stop() })
	time.Sleep(50 * time.Millisecond)
	return gw, "http://" + gw.Addr()
}

func TestGateway_RequestResponseE2E(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	_, err := k.Deploy(context.Background(), "gw-chat.ts", `
		bus.on("ask", (msg) => {
			msg.reply({ answer: "hello " + (msg.payload.name || "world") });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.Handle("POST", "/api/chat", "ts.gw-chat.ask")

	resp, err := http.Post(addr+"/api/chat", "application/json", strings.NewReader(`{"name":"david"}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "hello david", result["answer"])
}

func TestGateway_Timeout504(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-slow.ts", `
		bus.on("slow", (msg) => { /* no reply */ });
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k, func(cfg *gateway.Config) {
		cfg.Timeout = 1 * time.Second
	})
	gw.Handle("POST", "/api/slow", "ts.gw-slow.slow")

	resp, err := http.Post(addr+"/api/slow", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 504, resp.StatusCode)
}

func TestGateway_Webhook(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw, addr := startGateway(t, k)
	gw.HandleWebhook("POST", "/webhook/test", "gateway.webhook.test")

	resp, err := http.Post(addr+"/webhook/test", "application/json", strings.NewReader(`{"event":"test"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "ok")
}

func TestGateway_DrainReturns503(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw, addr := startGateway(t, k)
	gw.Handle("POST", "/api/test", "gateway.test")

	k.Kernel.SetDraining(true)
	defer k.Kernel.SetDraining(false)

	resp, err := http.Post(addr+"/api/test", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 503, resp.StatusCode)
}

func TestGateway_DrainAllowsWebhooks(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw, addr := startGateway(t, k)
	gw.HandleWebhook("POST", "/webhook/drain", "gateway.drain.webhook")

	k.Kernel.SetDraining(true)
	defer k.Kernel.SetDraining(false)

	resp, err := http.Post(addr+"/webhook/drain", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGateway_NotFound(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	_, addr := startGateway(t, k)

	resp, err := http.Get(addr + "/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode)
}

func TestGateway_CORSPreflight(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw, addr := startGateway(t, k, func(cfg *gateway.Config) {
		cfg.CORS = &gateway.CORSConfig{
			AllowOrigins: []string{"http://localhost:3000"},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Content-Type"},
		}
	})
	gw.Handle("POST", "/api/test", "gateway.test")

	req, _ := http.NewRequest("OPTIONS", addr+"/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestGateway_WithHTTPContext(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-ctx.ts", `
		bus.on("ctx", (msg) => {
			msg.reply({
				method: msg.payload.method,
				path: msg.payload.path,
				hasBody: msg.payload.body !== null && msg.payload.body !== undefined,
			});
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.Handle("POST", "/api/context", "ts.gw-ctx.ctx", gateway.WithHTTPContext())

	resp, err := http.Post(addr+"/api/context", "application/json", strings.NewReader(`{"test":true}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(body, &result)
	assert.Equal(t, "POST", result["method"])
	assert.Equal(t, "/api/context", result["path"])
	assert.Equal(t, true, result["hasBody"])
}

func TestGateway_PathParams(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-params.ts", `
		bus.on("call", (msg) => {
			msg.reply({ tool: msg.payload.name, input: msg.payload.input || "none" });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.Handle("POST", "/api/tools/{name}", "ts.gw-params.call", gateway.WithParam("name", "name"))

	resp, err := http.Post(addr+"/api/tools/echo", "application/json", strings.NewReader(`{"input":"hello"}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(body, &result)
	assert.Equal(t, "echo", result["tool"])
	assert.Equal(t, "hello", result["input"])
}

func TestGateway_RouteTable(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw := gateway.New(k, gateway.Config{Listen: "127.0.0.1:0"})

	gw.Handle("POST", "/api/chat", "gateway.chat", gateway.OwnedBy("chat.ts"))
	gw.HandleStream("GET", "/api/stream", "gateway.stream", gateway.OwnedBy("chat.ts"))
	gw.HandleWebhook("POST", "/webhook/tg", "gateway.tg", gateway.OwnedBy("tg.ts"))

	routes := gw.ListRoutes()
	assert.Len(t, routes, 3)

	removed := gw.RemoveByOwner("chat.ts")
	assert.Equal(t, 2, removed)
	assert.Len(t, gw.ListRoutes(), 1)

	gw.Remove("POST", "/webhook/tg")
	assert.Len(t, gw.ListRoutes(), 0)
}

func TestGateway_BusRouteAdd(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-bus.ts", `
		bus.on("dynamic", (msg) => {
			msg.reply({ dynamic: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)

	// Add route via bus command
	pr, err := sdk.Publish(k, context.Background(), messages.GatewayRouteAddMsg{
		Method: "POST", Path: "/api/dynamic", Topic: "ts.gw-bus.dynamic",
		Type: "handle", Owner: "gw-bus.ts",
	})
	require.NoError(t, err)

	done := make(chan messages.GatewayRouteAddResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.GatewayRouteAddResp](k, context.Background(), pr.ReplyTo, func(resp messages.GatewayRouteAddResp, msg messages.Message) {
		done <- resp
	})
	defer unsub()

	select {
	case resp := <-done:
		assert.True(t, resp.Added)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for route.add reply")
	}

	// Verify route exists
	routes := gw.ListRoutes()
	require.Len(t, routes, 1)
	assert.Equal(t, "/api/dynamic", routes[0].Path)

	// Hit the dynamic route
	resp, err := http.Post(addr+"/api/dynamic", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]bool
	json.Unmarshal(body, &result)
	assert.True(t, result["dynamic"])
}

func TestGateway_BusRouteRemoveByOwner(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw, _ := startGateway(t, k)

	gw.Handle("POST", "/a", "topic.a", gateway.OwnedBy("svc.ts"))
	gw.Handle("POST", "/b", "topic.b", gateway.OwnedBy("svc.ts"))
	gw.Handle("POST", "/c", "topic.c", gateway.OwnedBy("other.ts"))
	assert.Len(t, gw.ListRoutes(), 3)

	pr, _ := sdk.Publish(k, context.Background(), messages.GatewayRouteRemoveMsg{Owner: "svc.ts"})
	done := make(chan messages.GatewayRouteRemoveResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.GatewayRouteRemoveResp](k, context.Background(), pr.ReplyTo, func(resp messages.GatewayRouteRemoveResp, msg messages.Message) {
		done <- resp
	})
	defer unsub()

	select {
	case resp := <-done:
		assert.Equal(t, 2, resp.Removed)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	assert.Len(t, gw.ListRoutes(), 1)
	assert.Equal(t, "topic.c", gw.ListRoutes()[0].Topic)
}

func TestGateway_HealthEndpoints(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	_, addr := startGateway(t, k)

	resp, err := http.Get(addr + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))
}

func TestGateway_ReadyzDuringDrain(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	_, addr := startGateway(t, k)

	resp, _ := http.Get(addr + "/readyz")
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	k.Kernel.SetDraining(true)
	resp, _ = http.Get(addr + "/readyz")
	assert.Equal(t, 503, resp.StatusCode)
	resp.Body.Close()
	k.Kernel.SetDraining(false)
}

func TestGateway_SSEStreaming(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-sse.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.text("hello ");
			msg.stream.text("world");
			msg.stream.end({ done: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.HandleStream("GET", "/api/stream", "ts.gw-sse.stream")

	resp, err := http.Get(addr + "/api/stream")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	content := string(body)
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, "event: end")
}

func TestGateway_SSEProgressAndEvents(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-sse2.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.progress(0.5, "halfway");
			msg.stream.event("tool_start", { name: "search" });
			msg.stream.text("result");
			msg.stream.end({});
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.HandleStream("GET", "/api/stream2", "ts.gw-sse2.stream")

	resp, err := http.Get(addr + "/api/stream2")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)
	assert.Contains(t, content, "event: progress")
	assert.Contains(t, content, "event: tool_start")
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, "event: end")
}

func TestGateway_SSEErrorTerminates(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-sse-err.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.text("start");
			msg.stream.error("something broke");
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.HandleStream("GET", "/api/stream-err", "ts.gw-sse-err.stream")

	resp, err := http.Get(addr + "/api/stream-err")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, "event: error")
	assert.Contains(t, content, "something broke")
}

func TestGateway_ErrorResponse500(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-err.ts", `
		bus.on("fail", (msg) => {
			msg.reply({ error: "something went wrong" });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.Handle("POST", "/api/fail", "ts.gw-err.fail")

	resp, err := http.Post(addr+"/api/fail", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 500, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(body, &result)
	assert.Equal(t, "something went wrong", result["error"])
}

func TestGateway_HealthJSON(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	_, addr := startGateway(t, k)

	resp, err := http.Get(addr + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, true, result["healthy"])
	assert.Equal(t, "running", result["status"])
	checks, ok := result["checks"].([]any)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(checks), 2)
}

func TestGateway_RouteReplacement(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw := gateway.New(k, gateway.Config{Listen: "127.0.0.1:0"})

	gw.Handle("POST", "/api/chat", "topic.v1", gateway.OwnedBy("v1.ts"))
	routes := gw.ListRoutes()
	require.Len(t, routes, 1)
	assert.Equal(t, "topic.v1", routes[0].Topic)
	assert.Equal(t, "v1.ts", routes[0].Owner)

	// Same method+path → replace
	gw.Handle("POST", "/api/chat", "topic.v2", gateway.OwnedBy("v2.ts"))
	routes = gw.ListRoutes()
	require.Len(t, routes, 1)
	assert.Equal(t, "topic.v2", routes[0].Topic)
	assert.Equal(t, "v2.ts", routes[0].Owner)
}

func TestGateway_BusRouteList(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw, _ := startGateway(t, k)

	gw.Handle("POST", "/a", "topic.a")
	gw.HandleWebhook("POST", "/b", "topic.b")

	pr, err := sdk.Publish(k, context.Background(), messages.GatewayRouteListMsg{})
	require.NoError(t, err)

	done := make(chan messages.GatewayRouteListResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.GatewayRouteListResp](k, context.Background(), pr.ReplyTo, func(resp messages.GatewayRouteListResp, msg messages.Message) {
		done <- resp
	})
	defer unsub()

	select {
	case resp := <-done:
		assert.Len(t, resp.Routes, 2)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestGateway_BusStatus(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw, _ := startGateway(t, k)

	gw.Handle("POST", "/a", "topic.a")
	gw.Handle("POST", "/b", "topic.b")
	gw.Handle("POST", "/c", "topic.c")

	pr, err := sdk.Publish(k, context.Background(), messages.GatewayStatusMsg{})
	require.NoError(t, err)

	done := make(chan messages.GatewayStatusResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.GatewayStatusResp](k, context.Background(), pr.ReplyTo, func(resp messages.GatewayStatusResp, msg messages.Message) {
		done <- resp
	})
	defer unsub()

	select {
	case resp := <-done:
		assert.True(t, resp.Listening)
		assert.Equal(t, 3, resp.RouteCount)
		assert.NotEmpty(t, resp.Address)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestGateway_WebSocket(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	k.Deploy(context.Background(), "gw-ws.ts", `
		bus.on("ws", (msg) => {
			var data = msg.payload.data;
			if (typeof data === "string") {
				try { data = JSON.parse(data); } catch(e) {}
			}
			msg.reply({ echo: data, sessionId: msg.payload.sessionId });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	gw, addr := startGateway(t, k)
	gw.HandleWebSocket("/ws", "ts.gw-ws.ws")

	// Use the coder/websocket client
	wsURL := strings.Replace(addr, "http://", "ws://", 1) + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer conn.CloseNow()

	// Send a message
	err = conn.Write(ctx, websocket.MessageText, []byte(`{"hello":"world"}`))
	require.NoError(t, err)

	// Read reply
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	assert.NotEmpty(t, result["sessionId"])
	// The echo field contains the parsed data
	echo, ok := result["echo"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "world", echo["hello"])

	conn.Close(websocket.StatusNormalClosure, "done")
}
