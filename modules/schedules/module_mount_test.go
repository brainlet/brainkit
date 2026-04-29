package schedules

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
	"github.com/stretchr/testify/require"
)

func TestSchedulesModuleHotMountsCommandsAndHook(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{Transport: brainkit.Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, NewModule(Config{})))

	create, err := brainkit.Call[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](
		k,
		ctx,
		schedulemsg.ScheduleCreateMsg{
			Expression: "in 1h",
			Topic:      "test.schedule.target",
			Payload:    json.RawMessage(`{"ok":true}`),
		},
		brainkit.WithCallTimeout(2*time.Second),
	)
	require.NoError(t, err)
	require.NotEmpty(t, create.ID)

	list, err := brainkit.Call[schedulemsg.ScheduleListMsg, schedulemsg.ScheduleListResp](
		k,
		ctx,
		schedulemsg.ScheduleListMsg{},
		brainkit.WithCallTimeout(2*time.Second),
	)
	require.NoError(t, err)
	require.Len(t, list.Schedules, 1)
	require.Equal(t, create.ID, list.Schedules[0].ID)

	require.True(t, k.HasCommand("schedules.list"))
	require.NoError(t, k.Unmount(ctx, "schedules"))
	require.False(t, k.HasCommand("schedules.list"))
}
