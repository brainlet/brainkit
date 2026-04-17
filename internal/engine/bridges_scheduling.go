package engine

import (
	"context"
	"encoding/json"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	quickjs "github.com/buke/quickjs-go"
)

// registerSchedulingBridges adds bus.schedule and bus.unschedule bridges.
// The bridges dispatch to Kernel.scheduleHandler, which the schedules module
// sets during Init. Without a handler the bridges throw NOT_CONFIGURED so
// .ts code gets a clean error instead of a silent no-op.
func (k *Kernel) registerSchedulingBridges(qctx *quickjs.Context) {
	// __go_brainkit_bus_schedule(expression, topic, payloadJSON, source) → scheduleID
	qctx.Globals().Set(js.JSBridgeBusSchedule,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.schedule: expected 4 args"})
			}
			handler := k.scheduleHandler
			if handler == nil {
				return k.throwBrainkitError(qctx, &sdkerrors.NotConfiguredError{Feature: "schedules"})
			}
			id, err := handler.Schedule(context.Background(), types.ScheduleConfig{
				Expression: args[0].String(),
				Topic:      args[1].String(),
				Payload:    json.RawMessage(args[2].String()),
				Source:     args[3].String(),
			})
			if err != nil {
				return k.throwBrainkitError(qctx, err)
			}
			return qctx.NewString(id)
		}))

	// __go_brainkit_bus_unschedule(scheduleID)
	qctx.Globals().Set(js.JSBridgeBusUnschedule,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "scheduleId", Message: "bus.unschedule: expected 1 arg"})
			}
			handler := k.scheduleHandler
			if handler == nil {
				return k.throwBrainkitError(qctx, &sdkerrors.NotConfiguredError{Feature: "schedules"})
			}
			_ = handler.Unschedule(context.Background(), args[0].String())
			return qctx.NewUndefined()
		}))
}
