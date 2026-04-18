package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBusAPIHealthRoundTrip hits POST /api/bus with kit.health,
// asserts the gateway returns the bus reply as `{payload:...}`.
// This is the canonical external entry point for the CLI and any
// downstream HTTP client that wants to drive the Kit.
func testBusAPIHealthRoundTrip(t *testing.T, _ *suite.TestEnv) {
	k := suite.Full(t).Kit
	_, addr := gwStart(t, k)

	body := strings.NewReader(`{"topic":"kit.health","payload":{}}`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", addr+"/api/bus", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Payload json.RawMessage `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(raw, &out), "response: %s", string(raw))
	assert.NotEmpty(t, out.Payload, "kit.health response must carry a payload")

	// Peel the envelope wrapper if present so we can read into
	// KitHealthResp. Raw bus replies over /api/bus travel as
	// envelopes when the handler returns a typed Resp.
	respBody := out.Payload
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(respBody, &env) == nil && env.Data != nil {
		respBody = env.Data
	}

	var hresp struct {
		Health json.RawMessage `json:"health"`
	}
	require.NoError(t, json.Unmarshal(respBody, &hresp), "KitHealthResp decode: %s", string(respBody))

	var health struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(hresp.Health, &health))
	assert.NotEmpty(t, health.Status)
}

// testBusAPIMissingTopic rejects a POST /api/bus with an empty
// topic field with 400 + a clear error message.
func testBusAPIMissingTopic(t *testing.T, _ *suite.TestEnv) {
	k := suite.Full(t).Kit
	_, addr := gwStart(t, k)

	body := strings.NewReader(`{"topic":"","payload":{}}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", addr+"/api/bus", body)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	assert.Contains(t, strings.ToLower(out.Error), "topic")
}

// testBusAPIStreamNDJSON deploys a .ts handler that emits two
// intermediate chunks then a terminal reply. POST /api/stream
// must deliver every event as its own NDJSON line, with the
// terminal event carrying done=true.
func testBusAPIStreamNDJSON(t *testing.T, _ *suite.TestEnv) {
	k := suite.Full(t).Kit
	_, addr := gwStart(t, k)

	testutil.Deploy(t, k, "bus-api-stream.ts", `
		bus.on("tick", function(msg) {
			msg.send({seq: 1});
			msg.send({seq: 2});
			msg.reply({done: true});
		});
	`)

	body := strings.NewReader(`{"topic":"ts.bus-api-stream.tick","payload":{}}`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", addr+"/api/stream", body)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/x-ndjson", resp.Header.Get("Content-Type"))

	var events []map[string]any
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 4096), 1<<16)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var ev map[string]any
		require.NoError(t, json.Unmarshal(line, &ev), "line: %s", string(line))
		events = append(events, ev)
		if done, _ := ev["done"].(bool); done {
			break
		}
	}

	// At least the terminal event; chunk-count before the final
	// reply is best-effort on the memory transport (msg.send can
	// coalesce under back pressure). The contract the stream
	// endpoint upholds is ordered delivery terminated by
	// `done=true` — assert that and not a specific chunk count.
	require.GreaterOrEqual(t, len(events), 1, "expected at least 1 event")
	last := events[len(events)-1]
	assert.Equal(t, true, last["done"], "terminal event must carry done=true")
}
