package schedules

import (
	"context"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

func (m *Module) handleCreate(ctx context.Context, req sdk.ScheduleCreateMsg) (*sdk.ScheduleCreateResp, error) {
	id, err := m.scheduler.Schedule(ctx, types.ScheduleConfig{
		Expression: req.Expression,
		Topic:      req.Topic,
		Payload:    req.Payload,
	})
	if err != nil {
		return nil, err
	}
	return &sdk.ScheduleCreateResp{ID: id}, nil
}

func (m *Module) handleCancel(ctx context.Context, req sdk.ScheduleCancelMsg) (*sdk.ScheduleCancelResp, error) {
	if req.ID == "" {
		return nil, &sdkerrors.ValidationError{Field: "id", Message: "is required"}
	}
	if err := m.scheduler.Unschedule(ctx, req.ID); err != nil {
		return nil, err
	}
	return &sdk.ScheduleCancelResp{Cancelled: true}, nil
}

func (m *Module) handleList(ctx context.Context, req sdk.ScheduleListMsg) (*sdk.ScheduleListResp, error) {
	schedules := m.scheduler.List()
	infos := make([]sdk.ScheduleInfo, 0, len(schedules))
	for _, s := range schedules {
		infos = append(infos, sdk.ScheduleInfo{
			ID:         s.ID,
			Expression: s.Expression,
			Topic:      s.Topic,
			NextFire:   s.NextFire.Format("2006-01-02T15:04:05Z07:00"),
			OneTime:    s.OneTime,
			Source:     s.Source,
		})
	}
	return &sdk.ScheduleListResp{Schedules: infos}, nil
}
