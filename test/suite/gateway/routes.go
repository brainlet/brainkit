package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	bkgw "github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequestResponseE2E(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-chat.ts", `
		bus.on("ask", (msg) => {
			msg.reply({ answer: "hello " + (msg.payload.name || "world") });
		});
	`)

	gw, addr := gwStart(t, env.Kit)
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

func testTimeout504(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-slow.ts", `
		bus.on("slow", (msg) => { /* no reply */ });
	`)

	gw, addr := gwStart(t, env.Kit, func(cfg *bkgw.Config) {
		cfg.Timeout = 1 * time.Second
	})
	gw.Handle("POST", "/api/slow", "ts.gw-slow.slow")

	resp, err := http.Post(addr+"/api/slow", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 504, resp.StatusCode)
}

func testWebhook(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw, addr := gwStart(t, env.Kit)
	gw.HandleWebhook("POST", "/webhook/test", "gateway.webhook.test")

	resp, err := http.Post(addr+"/webhook/test", "application/json", strings.NewReader(`{"event":"test"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "ok")
}

func testDrainReturns503(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw, addr := gwStart(t, env.Kit)
	gw.Handle("POST", "/api/test", "gateway.test")

	testutil.SetDraining(t, env.Kit, true)
	defer testutil.SetDraining(t, env.Kit, false)

	resp, err := http.Post(addr+"/api/test", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 503, resp.StatusCode)
}

func testDrainAllowsWebhooks(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw, addr := gwStart(t, env.Kit)
	gw.HandleWebhook("POST", "/webhook/drain", "gateway.drain.webhook")

	testutil.SetDraining(t, env.Kit, true)
	defer testutil.SetDraining(t, env.Kit, false)

	resp, err := http.Post(addr+"/webhook/drain", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func testNotFound(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	_, addr := gwStart(t, env.Kit)

	resp, err := http.Get(addr + "/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode)
}

func testCORSPreflight(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw, addr := gwStart(t, env.Kit, func(cfg *bkgw.Config) {
		cfg.CORS = &bkgw.CORSConfig{
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

func testWithHTTPContext(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-ctx.ts", `
		bus.on("ctx", (msg) => {
			msg.reply({
				method: msg.payload.method,
				path: msg.payload.path,
				hasBody: msg.payload.body !== null && msg.payload.body !== undefined,
			});
		});
	`)

	gw, addr := gwStart(t, env.Kit)
	gw.Handle("POST", "/api/context", "ts.gw-ctx.ctx", bkgw.WithHTTPContext())

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

func testPathParams(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-params.ts", `
		bus.on("call", (msg) => {
			msg.reply({ tool: msg.payload.name, input: msg.payload.input || "none" });
		});
	`)

	gw, addr := gwStart(t, env.Kit)
	gw.Handle("POST", "/api/tools/{name}", "ts.gw-params.call", bkgw.WithParam("name", "name"))

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

func testRouteTable(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw := bkgw.New(bkgw.Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(env.Kit)

	gw.Handle("POST", "/api/chat", "gateway.chat", bkgw.OwnedBy("chat.ts"))
	gw.HandleStream("GET", "/api/stream", "gateway.stream", bkgw.OwnedBy("chat.ts"))
	gw.HandleWebhook("POST", "/webhook/tg", "gateway.tg", bkgw.OwnedBy("tg.ts"))

	routes := gw.ListRoutes()
	assert.Len(t, routes, 3)

	removed := gw.RemoveByOwner("chat.ts")
	assert.Equal(t, 2, removed)
	assert.Len(t, gw.ListRoutes(), 1)

	gw.Remove("POST", "/webhook/tg")
	assert.Len(t, gw.ListRoutes(), 0)
}

func testBusRouteAdd(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-bus.ts", `
		bus.on("dynamic", (msg) => {
			msg.reply({ dynamic: true });
		});
	`)

	gw, addr := gwStart(t, env.Kit)

	// Add route via bus command
	pr, err := sdk.Publish(env.Kit, context.Background(), sdk.GatewayRouteAddMsg{
		Method: "POST", Path: "/api/dynamic", Topic: "ts.gw-bus.dynamic",
		Type: "handle", Owner: "gw-bus.ts",
	})
	require.NoError(t, err)

	done := make(chan sdk.GatewayRouteAddResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.GatewayRouteAddResp](env.Kit, context.Background(), pr.ReplyTo, func(resp sdk.GatewayRouteAddResp, msg sdk.Message) {
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

func testBusRouteRemoveByOwner(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw, _ := gwStart(t, env.Kit)

	gw.Handle("POST", "/a", "topic.a", bkgw.OwnedBy("svc.ts"))
	gw.Handle("POST", "/b", "topic.b", bkgw.OwnedBy("svc.ts"))
	gw.Handle("POST", "/c", "topic.c", bkgw.OwnedBy("other.ts"))
	assert.Len(t, gw.ListRoutes(), 3)

	pr, _ := sdk.Publish(env.Kit, context.Background(), sdk.GatewayRouteRemoveMsg{Owner: "svc.ts"})
	done := make(chan sdk.GatewayRouteRemoveResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.GatewayRouteRemoveResp](env.Kit, context.Background(), pr.ReplyTo, func(resp sdk.GatewayRouteRemoveResp, msg sdk.Message) {
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

func testHealthEndpoints(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	_, addr := gwStart(t, env.Kit)

	resp, err := http.Get(addr + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))
}

func testReadyzDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	_, addr := gwStart(t, env.Kit)

	resp, _ := http.Get(addr + "/readyz")
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	testutil.SetDraining(t, env.Kit, true)
	resp, _ = http.Get(addr + "/readyz")
	assert.Equal(t, 503, resp.StatusCode)
	resp.Body.Close()
	testutil.SetDraining(t, env.Kit, false)
}

func testSSEStreaming(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-sse.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.text("hello ");
			msg.stream.text("world");
			msg.stream.end({ done: true });
		});
	`)

	gw, addr := gwStart(t, env.Kit)
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
	assert.Regexp(t, `id: [a-f0-9-]+:0`, content, "first event should have id with seq 0")
	assert.Regexp(t, `id: [a-f0-9-]+:\d+`, content, "events should have id: token:seq format")
}

func testSSEProgressAndEvents(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-sse2.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.progress(0.5, "halfway");
			msg.stream.event("tool_start", { name: "search" });
			msg.stream.text("result");
			msg.stream.end({});
		});
	`)

	gw, addr := gwStart(t, env.Kit)
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
	assert.Regexp(t, `id: [a-f0-9-]+:\d+`, content, "events should have id: token:seq format")
}

func testSSEErrorTerminates(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-sse-err.ts", `
		bus.on("stream", async (msg) => {
			msg.stream.text("start");
			msg.stream.error("something broke");
		});
	`)

	gw, addr := gwStart(t, env.Kit)
	gw.HandleStream("GET", "/api/stream-err", "ts.gw-sse-err.stream")

	resp, err := http.Get(addr + "/api/stream-err")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)
	assert.Contains(t, content, "event: text")
	assert.Contains(t, content, "event: error")
	assert.Contains(t, content, "something broke")
	assert.Regexp(t, `id: [a-f0-9-]+:\d+`, content, "error event should have id")
}

func testErrorResponse500(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-err.ts", `
		bus.on("fail", (msg) => {
			msg.reply({ error: "something went wrong" });
		});
	`)

	gw, addr := gwStart(t, env.Kit)
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

func testRouteReplacement(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw := bkgw.New(bkgw.Config{Listen: "127.0.0.1:0"})
	gw.SetRuntime(env.Kit)

	gw.Handle("POST", "/api/chat", "topic.v1", bkgw.OwnedBy("v1.ts"))
	routes := gw.ListRoutes()
	require.Len(t, routes, 1)
	assert.Equal(t, "topic.v1", routes[0].Topic)
	assert.Equal(t, "v1.ts", routes[0].Owner)

	// Same method+path -> replace
	gw.Handle("POST", "/api/chat", "topic.v2", bkgw.OwnedBy("v2.ts"))
	routes = gw.ListRoutes()
	require.Len(t, routes, 1)
	assert.Equal(t, "topic.v2", routes[0].Topic)
	assert.Equal(t, "v2.ts", routes[0].Owner)
}

func testBusRouteList(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw, _ := gwStart(t, env.Kit)

	gw.Handle("POST", "/a", "topic.a")
	gw.HandleWebhook("POST", "/b", "topic.b")

	pr, err := sdk.Publish(env.Kit, context.Background(), sdk.GatewayRouteListMsg{})
	require.NoError(t, err)

	done := make(chan sdk.GatewayRouteListResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.GatewayRouteListResp](env.Kit, context.Background(), pr.ReplyTo, func(resp sdk.GatewayRouteListResp, msg sdk.Message) {
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

func testBusStatus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw, _ := gwStart(t, env.Kit)

	gw.Handle("POST", "/a", "topic.a")
	gw.Handle("POST", "/b", "topic.b")
	gw.Handle("POST", "/c", "topic.c")

	pr, err := sdk.Publish(env.Kit, context.Background(), sdk.GatewayStatusMsg{})
	require.NoError(t, err)

	done := make(chan sdk.GatewayStatusResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.GatewayStatusResp](env.Kit, context.Background(), pr.ReplyTo, func(resp sdk.GatewayStatusResp, msg sdk.Message) {
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

func testWebSocket(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "gw-ws.ts", `
		bus.on("ws", (msg) => {
			var data = msg.payload.data;
			if (typeof data === "string") {
				try { data = JSON.parse(data); } catch(e) {}
			}
			msg.reply({ echo: data, sessionId: msg.payload.sessionId });
		});
	`)

	gw, addr := gwStart(t, env.Kit)
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

func testRateLimiting(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	gw := bkgw.New(bkgw.Config{
		Listen:   ":0",
		NoHealth: true,
		RateLimit: &bkgw.RateLimitConfig{
			RequestsPerSecond: 2,
			Burst:             2,
		},
	})
	gw.HandleWebhook("POST", "/test", "gateway.ratelimit.test")
	require.NoError(t, gw.Init(env.Kit))
	defer gw.Stop()

	url := "http://" + gw.Addr() + "/test"

	// First 2 requests should succeed (burst capacity)
	for i := 0; i < 2; i++ {
		resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode, "request %d should succeed", i)
		resp.Body.Close()
	}

	// Third request should be rate limited (burst exhausted, no time to refill)
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode, "third request should be rate limited")
	resp.Body.Close()
}
