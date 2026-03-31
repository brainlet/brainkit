package brainkit

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandler_DefaultDoesNotPanic(t *testing.T) {
	InvokeErrorHandler(nil, &sdkerrors.PersistenceError{
		Operation: "SaveDeployment", Source: "test.ts", Cause: fmt.Errorf("disk full"),
	}, ErrorContext{
		Operation: "SaveDeployment", Component: "kernel", Source: "test.ts",
	})
}

func TestErrorHandler_CustomReceivesTypedErrors(t *testing.T) {
	var mu sync.Mutex
	var received []error
	var contexts []ErrorContext

	handler := func(err error, ctx ErrorContext) {
		mu.Lock()
		received = append(received, err)
		contexts = append(contexts, ctx)
		mu.Unlock()
	}

	InvokeErrorHandler(handler, &sdkerrors.PersistenceError{
		Operation: "LoadDeployments", Source: "", Cause: fmt.Errorf("corrupt db"),
	}, ErrorContext{
		Operation: "LoadDeployments", Component: "kernel",
	})

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1)
	require.Len(t, contexts, 1)

	var pe *sdkerrors.PersistenceError
	assert.True(t, errors.As(received[0], &pe))
	assert.Equal(t, "LoadDeployments", pe.Operation)
	assert.Equal(t, "kernel", contexts[0].Component)
}

func TestErrorHandler_ContextFields(t *testing.T) {
	var ctx ErrorContext
	handler := func(err error, c ErrorContext) { ctx = c }

	InvokeErrorHandler(handler, fmt.Errorf("test"), ErrorContext{
		Operation: "RestorePlugin", Component: "node", Source: "telegram",
	})

	assert.Equal(t, "RestorePlugin", ctx.Operation)
	assert.Equal(t, "node", ctx.Component)
	assert.Equal(t, "telegram", ctx.Source)
}
