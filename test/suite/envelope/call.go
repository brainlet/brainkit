package envelope

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCallReturnsTypedNotFound — brainkit.Call of a missing tool
// returns a *NotFoundError directly (not a fake success reply with an
// error field buried inside).
func testCallReturnsTypedNotFound(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := brainkit.Call[sdk.ToolCallMsg, json.RawMessage](env.Kit, ctx, sdk.ToolCallMsg{
		Name: "ghost-tool-env-call",
	})
	require.Error(t, err)

	var nf *sdkerrors.NotFoundError
	require.True(t, errors.As(err, &nf), "want *NotFoundError, got %T: %v", err, err)
	assert.Equal(t, "tool", nf.Resource)
	assert.Equal(t, "ghost-tool-env-call", nf.Name)
}

// testCallReturnsTypedValidation — brainkit.Call with invalid input
// returns a *ValidationError directly.
func testCallReturnsTypedValidation(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := brainkit.Call[sdk.SecretsSetMsg, json.RawMessage](env.Kit, ctx, sdk.SecretsSetMsg{
		Name:  "",
		Value: "val",
	})
	require.Error(t, err)

	var ve *sdkerrors.ValidationError
	assert.True(t, errors.As(err, &ve), "want *ValidationError, got %T: %v", err, err)
}

// testCallReturnsBusErrorOnUnknownCode — a synthetic envelope with a
// code not in FromEnvelope's switch resolves to *BusError (carrier)
// after a JSON round trip (the path real wire messages take).
func testCallReturnsBusErrorOnUnknownCode(t *testing.T, _ *suite.TestEnv) {
	raw, err := sdk.EncodeEnvelope(sdk.EnvelopeErr("FORWARD_COMPAT_CODE", "new thing", map[string]any{"v": 2}))
	require.NoError(t, err)
	wire, err := sdk.DecodeEnvelope(raw)
	require.NoError(t, err)

	decoded := sdk.FromEnvelope(wire)
	var be *sdkerrors.BusError
	require.True(t, errors.As(decoded, &be))
	assert.Equal(t, "FORWARD_COMPAT_CODE", be.Code())
	assert.Equal(t, "new thing", be.Message)
	// After JSON round trip, numbers are float64.
	assert.Equal(t, float64(2), be.Details()["v"])
}
