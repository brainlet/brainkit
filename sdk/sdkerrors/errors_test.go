package sdkerrors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrainkitError_Interface(t *testing.T) {
	types := []sdkerrors.BrainkitError{
		&sdkerrors.NotFoundError{Resource: "tool", Name: "echo"},
		&sdkerrors.AlreadyExistsError{Resource: "deployment", Name: "agent.ts"},
		&sdkerrors.ValidationError{Field: "name", Message: "required"},
		&sdkerrors.TimeoutError{Operation: "plugin READY"},
		&sdkerrors.WorkspaceEscapeError{Path: "../etc/passwd"},
		&sdkerrors.NotConfiguredError{Feature: "rbac"},
		&sdkerrors.TransportError{Operation: "publish", Cause: fmt.Errorf("connection refused")},
		&sdkerrors.PersistenceError{Operation: "SaveDeployment", Source: "agent.ts", Cause: fmt.Errorf("disk full")},
		&sdkerrors.DeployError{Source: "agent.ts", Phase: "transpile", Cause: fmt.Errorf("syntax error")},
		&sdkerrors.BridgeError{Function: "__go_brainkit_request", Cause: fmt.Errorf("eval busy")},
		&sdkerrors.CompilerError{Cause: fmt.Errorf("out of memory")},
		&sdkerrors.CycleDetectedError{Depth: 16},
		&sdkerrors.DecodeError{Topic: "tools.call", Cause: fmt.Errorf("invalid json")},
	}

	for _, err := range types {
		t.Run(err.Code(), func(t *testing.T) {
			assert.NotEmpty(t, err.Error())
			assert.NotEmpty(t, err.Code())
			assert.NotNil(t, err.Details())
			assert.Regexp(t, `^[A-Z][A-Z0-9_]+$`, err.Code())
		})
	}
}

func TestBrainkitError_ErrorsAs(t *testing.T) {
	inner := &sdkerrors.NotFoundError{Resource: "tool", Name: "echo"}
	wrapped := fmt.Errorf("tools.call: %w", inner)

	var target *sdkerrors.NotFoundError
	require.True(t, errors.As(wrapped, &target))
	assert.Equal(t, "tool", target.Resource)
	assert.Equal(t, "echo", target.Name)

	var bk sdkerrors.BrainkitError
	require.True(t, errors.As(wrapped, &bk))
	assert.Equal(t, "NOT_FOUND", bk.Code())
}

func TestBrainkitError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := &sdkerrors.TransportError{Operation: "publish", Cause: cause}

	assert.True(t, errors.Is(err, cause))
	assert.Equal(t, "TRANSPORT_ERROR", err.Code())
	assert.Equal(t, "publish", err.Details()["operation"])
}

func TestDeployError_Details(t *testing.T) {
	cause := fmt.Errorf("unexpected token")
	err := &sdkerrors.DeployError{Source: "bot.ts", Phase: "transpile", Cause: cause}
	assert.Equal(t, "DEPLOY_ERROR", err.Code())
	assert.Equal(t, "bot.ts", err.Details()["source"])
	assert.Equal(t, "transpile", err.Details()["phase"])
	assert.True(t, errors.Is(err, cause))
}

func TestPersistenceError_Details(t *testing.T) {
	cause := fmt.Errorf("disk full")
	err := &sdkerrors.PersistenceError{Operation: "SaveDeployment", Source: "agent.ts", Cause: cause}
	assert.Equal(t, "PERSISTENCE_ERROR", err.Code())
	assert.Equal(t, "SaveDeployment", err.Details()["operation"])
	assert.Equal(t, "agent.ts", err.Details()["source"])
	assert.True(t, errors.Is(err, cause))
}

func TestNotConfiguredError_Variants(t *testing.T) {
	features := []string{"rbac", "mcp", "discovery", "tracing", "secrets", "workspace"}
	for _, f := range features {
		err := &sdkerrors.NotConfiguredError{Feature: f}
		assert.Equal(t, "NOT_CONFIGURED", err.Code())
		assert.Equal(t, f, err.Details()["feature"])
		assert.Contains(t, err.Error(), f)
	}
}

func TestCycleDetectedError(t *testing.T) {
	err := &sdkerrors.CycleDetectedError{Depth: 16}
	assert.Equal(t, "CYCLE_DETECTED", err.Code())
	assert.Equal(t, 16, err.Details()["depth"])
}

func TestDecodeError(t *testing.T) {
	cause := fmt.Errorf("invalid json")
	err := &sdkerrors.DecodeError{Topic: "tools.call", Cause: cause}
	assert.Equal(t, "DECODE_ERROR", err.Code())
	assert.Equal(t, "tools.call", err.Details()["topic"])
	assert.True(t, errors.Is(err, cause))
}

func TestBridgeError(t *testing.T) {
	cause := fmt.Errorf("eval busy")
	err := &sdkerrors.BridgeError{Function: "secret_get", Cause: cause}
	assert.Equal(t, "BRIDGE_ERROR", err.Code())
	assert.Equal(t, "secret_get", err.Details()["function"])
	assert.True(t, errors.Is(err, cause))
}

func TestCompilerError(t *testing.T) {
	cause := fmt.Errorf("out of memory")
	err := &sdkerrors.CompilerError{Cause: cause}
	assert.Equal(t, "COMPILER_ERROR", err.Code())
	assert.True(t, errors.Is(err, cause))
}
