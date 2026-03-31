package infra_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/kit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusRateLimit_ExceedsLimit(t *testing.T) {
	storePath := t.TempDir() + "/ratelimit.db"
	store, _ := kit.NewSQLiteStore(storePath)

	k, err := kit.NewKernel(kit.KernelConfig{
		Store: store,
		Roles: map[string]rbac.Role{
			"limited": {
				Name: "limited",
				Bus: rbac.BusPermissions{
					Publish:   rbac.TopicFilter{Allow: []string{"*"}},
					Subscribe: rbac.TopicFilter{Allow: []string{"*"}},
					Emit:      rbac.TopicFilter{Allow: []string{"*"}},
				},
				Commands:     rbac.CommandPermissions{Allow: []string{"*"}},
				Registration: rbac.RegistrationPermissions{Tools: true, Agents: true},
			},
		},
		DefaultRole:   "limited",
		BusRateLimits: map[string]float64{"limited": 2}, // 2 req/sec
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Deploy a service with the limited role
	_, err = k.Deploy(ctx, "rate-test.ts", `
		bus.on("test", async (msg) => {
			var results = [];
			for (var i = 0; i < 5; i++) {
				try {
					bus.publish("events.tick", { i: i });
					results.push({ i: i, ok: true });
				} catch(e) {
					results.push({ i: i, error: e.message });
				}
			}
			msg.reply({ results: results });
		});
	`, kit.WithRole("limited"))
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "rate-test.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		results, ok := resp["results"].([]any)
		require.True(t, ok, "expected results array, got: %v", resp)

		// First 2 should succeed (burst), rest should fail with rate limit
		successCount := 0
		rateLimitCount := 0
		for _, r := range results {
			item := r.(map[string]any)
			if item["ok"] == true {
				successCount++
			}
			if errMsg, ok := item["error"].(string); ok && errMsg != "" {
				rateLimitCount++
				assert.Contains(t, errMsg, "rate limit")
			}
		}
		assert.GreaterOrEqual(t, successCount, 1, "at least 1 publish should succeed")
		assert.GreaterOrEqual(t, rateLimitCount, 1, "at least 1 publish should be rate limited")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
