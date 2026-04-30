package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopeCloseRunsCleanupsInReverseOrder(t *testing.T) {
	scope := NewScope("test")
	var order []int
	scope.Defer(func(context.Context) error {
		order = append(order, 1)
		return nil
	})
	scope.Defer(func(context.Context) error {
		order = append(order, 2)
		return nil
	})

	require.NoError(t, scope.Close(context.Background()))
	require.Equal(t, []int{2, 1}, order)
	require.NoError(t, scope.Close(context.Background()))
	require.Equal(t, []int{2, 1}, order)
}

func TestScopeChildClosesWithParent(t *testing.T) {
	parent := NewScope("parent")
	child := parent.Child("child")
	closed := false
	child.Defer(func(context.Context) error {
		closed = true
		return nil
	})

	require.NoError(t, parent.Close(context.Background()))
	require.True(t, closed)
}

func TestScopeResourcesIncludeChildrenAndNormalize(t *testing.T) {
	parent := NewScope("parent")
	parent.Resource(Resource(ResourceKindProcess, "worker"))
	parent.Resource(Resource(ResourceKindProcess, "worker"))
	child := parent.Child("child")
	child.Resource(Resource(ResourceKindHook, "handler"))

	require.Equal(t, []ResourceDescriptor{
		Resource(ResourceKindHook, "handler"),
		Resource(ResourceKindProcess, "worker"),
	}, parent.Resources())
}

func TestCapabilityRegistryProvideAndClose(t *testing.T) {
	registry := NewCapabilityRegistry()
	handle, err := registry.Provide(context.Background(), "trace-store", "value")
	require.NoError(t, err)

	value, ok := registry.Get("trace-store")
	require.True(t, ok)
	require.Equal(t, "value", value)

	_, err = registry.Provide(context.Background(), "trace-store", "other")
	require.Error(t, err)

	require.NoError(t, handle.Close(context.Background()))
	_, ok = registry.Get("trace-store")
	require.False(t, ok)
}
