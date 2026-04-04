package security

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	bkgw "github.com/brainlet/brainkit/gateway"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// secGateway creates a gateway with short timeout for security tests.
func secGateway(t *testing.T, env *suite.TestEnv) *bkgw.Gateway {
	t.Helper()
	gw := bkgw.New(env.Kernel, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
	})
	require.NoError(t, gw.Start())
	t.Cleanup(func() { gw.Stop() })
	return gw
}

// secGwPost performs an HTTP POST and returns status code + body string.
func secGwPost(t *testing.T, gw *bkgw.Gateway, path, body string) (int, string) {
	t.Helper()
	url := "http://" + gw.Addr() + path
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	rbody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(rbody)
}

// testGatewayHeaderInjection — inject headers to forge callerID.
func testGatewayHeaderInjection(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := bkgw.New(k, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
	})
	require.NoError(t, gw.Start())
	defer gw.Stop()

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-header-sec.ts", `
		bus.on("whoami", function(msg) {
			msg.reply({callerId: msg.callerId, topic: msg.topic});
		});
	`)
	require.NoError(t, err)
	gw.Handle("POST", "/whoami-sec", "ts.gw-header-sec.whoami")

	req, _ := http.NewRequest("POST", "http://"+gw.Addr()+"/whoami-sec", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-ID", "admin")
	req.Header.Set("X-Correlation-ID", "forged-correlation")
	req.Header.Set("X-Reply-To", "evil.reply.topic")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	assert.NotContains(t, string(body), `"callerId":"admin"`, "forged header should not set callerID")
}

// testGatewayProtoPollutionViaHTTP — request body with __proto__ pollution.
func testGatewayProtoPollutionViaHTTP(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := bkgw.New(k, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
	})
	require.NoError(t, gw.Start())
	defer gw.Stop()

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-proto-sec.ts", `
		bus.on("check", function(msg) {
			msg.reply({
				hasProto: msg.payload.__proto__ !== undefined,
				hasPollution: ({}).polluted === true,
				raw: msg.payload,
			});
		});
	`)
	require.NoError(t, err)
	gw.Handle("POST", "/proto-check-sec", "ts.gw-proto-sec.check")

	evilPayload := `{"__proto__":{"polluted":true},"constructor":{"prototype":{"pwned":true}},"data":"test"}`
	status, body := secGwPost(t, gw, "/proto-check-sec", evilPayload)
	assert.Equal(t, 200, status)
	assert.NotContains(t, body, `"hasPollution":true`, "proto pollution via HTTP should not work")
}

// testGatewayPathTraversalParams — path traversal in URL parameters.
func testGatewayPathTraversalParams(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := bkgw.New(k, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
	})
	require.NoError(t, gw.Start())
	defer gw.Stop()

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-params-sec.ts", `
		bus.on("get", function(msg) {
			msg.reply({id: msg.payload.id});
		});
	`)
	require.NoError(t, err)
	gw.Handle("GET", "/items-sec/{id}", "ts.gw-params-sec.get", bkgw.WithParam("id", "id"))

	evilPaths := []string{
		"/items-sec/../../../etc/passwd",
		"/items-sec/..%2F..%2F..%2Fetc%2Fpasswd",
		"/items-sec/foo%00bar",
		"/items-sec/<script>alert(1)</script>",
		"/items-sec/' OR 1=1 --",
	}

	for _, path := range evilPaths {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get("http://" + gw.Addr() + path)
			if err != nil {
				return
			}
			resp.Body.Close()
		})
	}
	assert.True(t, k.Alive(ctx))
}

// testGatewayWebSocketInjection — WebSocket message injection.
func testGatewayWebSocketInjection(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	gw := bkgw.New(k, bkgw.Config{
		Listen:  ":0",
		Timeout: 3 * time.Second,
	})
	require.NoError(t, gw.Start())
	defer gw.Stop()

	ctx := context.Background()
	_, err := k.Deploy(ctx, "gw-ws-sec.ts", `
		bus.on("ws", function(msg) {
			msg.reply({received: msg.payload});
		});
	`)
	require.NoError(t, err)
	gw.HandleWebSocket("/ws-attack-sec", "ts.gw-ws-sec.ws")

	resp, err := http.Get("http://" + gw.Addr() + "/ws-attack-sec")
	if err != nil {
		return
	}
	resp.Body.Close()
	assert.True(t, k.Alive(ctx))
}
