package quickjs

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleHook(t *testing.T) {
	rt := NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	t.Run("HookCalledOnSchedule", func(t *testing.T) {
		var callCount atomic.Int32
		ctx.SetScheduleHook(func() {
			callCount.Add(1)
		})

		require.True(t, ctx.Schedule(func(*Context) {}))
		require.True(t, ctx.Schedule(func(*Context) {}))
		require.True(t, ctx.Schedule(func(*Context) {}))

		assert.Equal(t, int32(3), callCount.Load())

		ctx.ProcessJobs()
		ctx.SetScheduleHook(nil)
	})

	t.Run("NilHookDoesNotPanic", func(t *testing.T) {
		ctx.SetScheduleHook(nil)
		require.True(t, ctx.Schedule(func(*Context) {}))
		ctx.ProcessJobs()
	})

	t.Run("HookNotCalledOnFailedSchedule", func(t *testing.T) {
		var callCount atomic.Int32
		ctx.SetScheduleHook(func() {
			callCount.Add(1)
		})

		require.False(t, ctx.Schedule(nil))
		assert.Equal(t, int32(0), callCount.Load())

		ctx.SetScheduleHook(nil)
	})
}
