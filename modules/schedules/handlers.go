package schedules

import (
	"context"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

func (m *Module) handleCreate(ctx context.Context, req schedulemsg.ScheduleCreateMsg) (*schedulemsg.ScheduleCreateResp, error) {
	id, err := m.scheduler.Schedule(ctx, types.ScheduleConfig{
		Expression: req.Expression,
		Topic:      req.Topic,
		Payload:    req.Payload,
	})
	if err != nil {
		return nil, err
	}
	return &schedulemsg.ScheduleCreateResp{ID: id}, nil
}

func (m *Module) handleCancel(ctx context.Context, req schedulemsg.ScheduleCancelMsg) (*schedulemsg.ScheduleCancelResp, error) {
	if req.ID == "" {
		return nil, &sdkerrors.ValidationError{Field: "id", Message: "is required"}
	}
	if err := m.scheduler.Unschedule(ctx, req.ID); err != nil {
		return nil, err
	}
	return &schedulemsg.ScheduleCancelResp{Cancelled: true}, nil
}

func (m *Module) handleList(ctx context.Context, req schedulemsg.ScheduleListMsg) (*schedulemsg.ScheduleListResp, error) {
	schedules := m.scheduler.List()
	infos := make([]schedulemsg.ScheduleInfo, 0, len(schedules))
	for _, s := range schedules {
		infos = append(infos, schedulemsg.ScheduleInfo{
			ID:         s.ID,
			Expression: s.Expression,
			Topic:      s.Topic,
			NextFire:   s.NextFire.Format("2006-01-02T15:04:05Z07:00"),
			OneTime:    s.OneTime,
			Source:     s.Source,
		})
	}
	return &schedulemsg.ScheduleListResp{Schedules: infos}, nil
}
