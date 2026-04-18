package brainkit_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCallWrapperRoundTrip verifies that a generated synchronous
// Call wrapper delegates correctly to the underlying generic and
// returns a typed response. Uses kit.health because it's
// always-on, requires no config, and returns a structurally
// interesting response.
func TestCallWrapperRoundTrip(t *testing.T) {
	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "call-wrapper-test",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { kit.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := brainkit.CallKitHealth(kit, ctx, sdk.KitHealthMsg{})
	require.NoError(t, err, "generated CallKitHealth wrapper must round-trip")
	assert.NotEmpty(t, resp.Health, "health response should carry a Health payload")
}

// TestCallWrapperParity asserts that the wrapper returns the same
// shape as the underlying generic Call — proving the generated
// file is a pure delegation, not a drift risk. Compares the
// stable `status` field (kit.health also reports an uptime
// counter that changes between invocations, so the full payload
// blob isn't byte-equal).
func TestCallWrapperParity(t *testing.T) {
	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "call-wrapper-parity",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { kit.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	viaWrapper, err := brainkit.CallKitHealth(kit, ctx, sdk.KitHealthMsg{})
	require.NoError(t, err)

	viaGeneric, err := brainkit.Call[sdk.KitHealthMsg, sdk.KitHealthResp](
		kit, ctx, sdk.KitHealthMsg{},
	)
	require.NoError(t, err)

	decode := func(raw json.RawMessage) string {
		var shape struct {
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(raw, &shape))
		return shape.Status
	}
	assert.Equal(t, decode(viaGeneric.Health), decode(viaWrapper.Health))
}

// TestCallWrapperRespectsOptions verifies that CallOption values
// pass through the wrapper (WithCallTimeout here). A 1ns timeout
// must surface as a CallTimeoutError — confirming the option
// reached the generic.
func TestCallWrapperRespectsOptions(t *testing.T) {
	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "call-wrapper-opts",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { kit.Close() })

	// Parent ctx has plenty of headroom; the option must override.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = brainkit.CallKitHealth(kit, ctx, sdk.KitHealthMsg{},
		brainkit.WithCallTimeout(1))
	require.Error(t, err, "1ns timeout must surface an error")
}
