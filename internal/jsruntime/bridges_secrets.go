package jsruntime

import (
	"context"
	"encoding/json"
	"time"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	quickjs "github.com/buke/quickjs-go"
)

// registerSecretBridges adds __go_brainkit_secret_get bridge.
func (r *Runtime) registerSecretBridges(qctx *quickjs.Context) {
	// __go_brainkit_secret_get(name) → value or "" (not found)
	qctx.Globals().Set(js.JSBridgeSecretGet,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "name", Message: "is required"})
			}
			name := args[0].String()

			if r.core.SecretStore() == nil {
				return r.throwBrainkitError(qctx, &sdkerrors.NotConfiguredError{Feature: "secrets"})
			}
			val, err := r.core.SecretStore().Get(context.Background(), name)
			if err != nil {
				return r.throwBrainkitError(qctx, &sdkerrors.BridgeError{Function: "secret_get", Cause: err})
			}
			if val == "" {
				return qctx.NewString("") // legitimate "not found"
			}
			// Audit: emit secrets.accessed event
			source := r.CurrentSource()
			if source == "" {
				source = r.core.CallerID()
			}
			r.emitSecretEvent(context.Background(), secretmsg.SecretsAccessedEvent{
				Name:      name,
				Accessor:  source,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			return qctx.NewString(val)
		}))
}

func (r *Runtime) emitSecretEvent(ctx context.Context, event sdk.BrainkitMessage) {
	payload, _ := json.Marshal(event)
	_ = r.bus.PublishEvent(ctx, event.BusTopic(), payload)
}
