package module

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopeCloseRetainsFailedCleanupsForRetry(t *testing.T) {
	scope := NewScope("retry")
	want := errors.New("close failed")
	fail := true
	var calls []string

	scope.Defer(func(context.Context) error {
		calls = append(calls, "first")
		return nil
	})
	scope.Defer(func(context.Context) error {
		calls = append(calls, "second")
		if fail {
			fail = false
			return want
		}
		return nil
	})

	err := scope.Close(context.Background())
	require.ErrorIs(t, err, want)
	require.Equal(t, []string{"second", "first"}, calls)

	require.NoError(t, scope.Close(context.Background()))
	require.Equal(t, []string{"second", "first", "second"}, calls)

	require.NoError(t, scope.Close(context.Background()))
	require.Equal(t, []string{"second", "first", "second"}, calls)
}
