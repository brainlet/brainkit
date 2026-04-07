package engine

import (
	"context"
	"time"

	quickjs "github.com/buke/quickjs-go"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/sdk/messages"
)

// registerSecretBridges adds __go_brainkit_secret_get bridge.
func (k *Kernel) registerSecretBridges(qctx *quickjs.Context) {
	// __go_brainkit_secret_get(name) → value or "" (not found)
	qctx.Globals().Set(js.JSBridgeSecretGet,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "name", Message: "is required"})
			}
			name := args[0].String()

			// RBAC enforcement — must have secrets.get permission
			if err := k.checkCommandPermission(k.currentDeploymentSource(), "secrets.get"); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			if k.secretStore == nil {
				return k.throwBrainkitError(qctx, &sdkerrors.NotConfiguredError{Feature: "secrets"})
			}
			val, err := k.secretStore.Get(context.Background(), name)
			if err != nil {
				return k.throwBrainkitError(qctx, &sdkerrors.BridgeError{Function: "secret_get", Cause: err})
			}
			if val == "" {
				return qctx.NewString("") // legitimate "not found"
			}
			// Audit: emit secrets.accessed event
			source := k.currentDeploymentSource()
			if source == "" {
				source = k.callerID
			}
			k.emitSecretEvent(context.Background(), messages.SecretsAccessedEvent{
				Name:      name,
				Accessor:  source,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			return qctx.NewString(val)
		}))
}
