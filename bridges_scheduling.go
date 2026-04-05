package brainkit

import (
	"context"
	"encoding/json"

	quickjs "github.com/buke/quickjs-go"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/sdkerrors"
)

// registerSchedulingBridges adds bus.schedule and bus.unschedule bridges.
func (k *Kernel) registerSchedulingBridges(qctx *quickjs.Context) {
	// __go_brainkit_bus_schedule(expression, topic, payloadJSON, source) → scheduleID
	qctx.Globals().Set(js.JSBridgeBusSchedule,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.schedule: expected 4 args"})
			}
			expression := args[0].String()
			topic := args[1].String()
			payload := json.RawMessage(args[2].String())
			source := args[3].String()

			id, err := k.Schedule(context.Background(), ScheduleConfig{
				Expression: expression,
				Topic:      topic,
				Payload:    payload,
				Source:     source,
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
			k.Unschedule(context.Background(), args[0].String())
			return qctx.NewUndefined()
		}))
}
