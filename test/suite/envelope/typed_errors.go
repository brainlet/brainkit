package envelope

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNotFoundRoundTrip — tools.call of a missing tool produces a
// wire envelope with code=NOT_FOUND that decodes back to *NotFoundError.
func testNotFoundRoundTrip(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{Name: "ghost-tool-env-rt"})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		ch <- m.Payload
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		wire, derr := sdk.DecodeEnvelope(payload)
		require.NoError(t, derr)
		require.False(t, wire.Ok, "wire envelope must be ok=false for NOT_FOUND")
		require.NotNil(t, wire.Error)
		assert.Equal(t, "NOT_FOUND", wire.Error.Code)

		// FromEnvelope should reconstruct *NotFoundError
		decoded := sdk.FromEnvelope(wire)
		var nf *sdkerrors.NotFoundError
		require.True(t, errors.As(decoded, &nf), "want *NotFoundError, got %T: %v", decoded, decoded)
		assert.Equal(t, "tool", nf.Resource)
		assert.Equal(t, "ghost-tool-env-rt", nf.Name)
	case <-ctx.Done():
		t.Fatal("timeout waiting for envelope reply")
	}
}

// testValidationErrorRoundTrip — secrets.set with empty name surfaces
// VALIDATION_ERROR envelope that decodes back to *ValidationError.
func testValidationErrorRoundTrip(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "", Value: "val"})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		wire, derr := sdk.DecodeEnvelope(payload)
		require.NoError(t, derr)
		require.False(t, wire.Ok)
		require.NotNil(t, wire.Error)
		assert.Equal(t, "VALIDATION_ERROR", wire.Error.Code)

		decoded := sdk.FromEnvelope(wire)
		var ve *sdkerrors.ValidationError
		assert.True(t, errors.As(decoded, &ve), "want *ValidationError, got %T", decoded)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testUnknownCodeBecomesBusError — synthetic envelope with unknown code
// decodes to *BusError (generic carrier) not something more specific.
func testUnknownCodeBecomesBusError(t *testing.T, _ *suite.TestEnv) {
	raw, err := sdk.EncodeEnvelope(sdk.EnvelopeErr("SOME_FUTURE_CODE", "future failure", map[string]any{"x": 1}))
	require.NoError(t, err)

	wire, derr := sdk.DecodeEnvelope(raw)
	require.NoError(t, derr)
	require.False(t, wire.Ok)

	decoded := sdk.FromEnvelope(wire)
	var be *sdkerrors.BusError
	require.True(t, errors.As(decoded, &be), "want *BusError, got %T", decoded)
	assert.Equal(t, "SOME_FUTURE_CODE", be.Code())
	assert.Equal(t, "future failure", be.Message)
	assert.Equal(t, float64(1), be.Details()["x"])

	// Interface contract: BusError implements BrainkitError.
	var bk sdkerrors.BrainkitError
	require.True(t, errors.As(decoded, &bk))
	assert.Equal(t, "SOME_FUTURE_CODE", bk.Code())

	// Cheap smoke: json round-trip
	_ = json.RawMessage(raw)
}
