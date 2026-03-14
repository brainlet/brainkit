package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	quickjs "github.com/buke/quickjs-go"
)

// registerBridges adds Go bridge functions to the sandbox's QuickJS context.
// Called by brainlet-runtime.js for PLATFORM operations.
func (s *Sandbox) registerBridges() {
	ctx := s.agents.Bridge().Context()

	// __go_brainkit_request(topic, payloadJSON) → resultJSON
	ctx.Globals().Set("__go_brainkit_request",
		ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("brainkit_request: expected 2 args (topic, payload)"))
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			goCtx := context.Background()
			resp, err := s.kit.Bus.Request(goCtx, topic, s.callerID, payload)
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("brainkit_request %s: %w", topic, err))
			}

			return qctx.NewString(string(resp.Payload))
		}))

	// Set sandbox context globals
	ctx.Globals().Set("__brainkit_sandbox_id", ctx.NewString(s.id))
	ctx.Globals().Set("__brainkit_sandbox_namespace", ctx.NewString(s.namespace))
	ctx.Globals().Set("__brainkit_sandbox_callerID", ctx.NewString(s.callerID))
}
