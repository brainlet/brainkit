package engine

import (
	"encoding/json"
	"errors"
	"fmt"

	quickjs "github.com/buke/quickjs-go"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

var (
	sdkEnvelopeErr  = sdk.EnvelopeErr
	sdkFromEnvelope = sdk.FromEnvelope
)

// registerLoggingBridge adds __go_console_log_tagged for per-Compartment tagged logging.
func (k *Kernel) registerLoggingBridge(qctx *quickjs.Context) {
	qctx.Globals().Set(js.JSBridgeConsoleLogTagged,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 3 {
				return qctx.NewUndefined()
			}
			source := args[0].String()
			level := args[1].String()
			message := args[2].String()
			k.emitLog(source, level, message)
			return qctx.NewUndefined()
		}))
}

// throwBrainkitError constructs a JS error with real `.code` and `.details`
// properties and throws it. Wire envelope is built by the caller; this is
// the in-process JS bridge path where the error crosses the Compartment
// boundary via direct value, not JSON.
func (k *Kernel) throwBrainkitError(qctx *quickjs.Context, err error) *quickjs.Value {
	var bkErr sdkerrors.BrainkitError
	code := "INTERNAL_ERROR"
	detailsJSON := "{}"
	msg := err.Error()

	if errors.As(err, &bkErr) {
		code = bkErr.Code()
		if d := bkErr.Details(); d != nil {
			if b, e := json.Marshal(d); e == nil {
				detailsJSON = string(b)
			}
		}
	}

	script := fmt.Sprintf(`(function() {
		var e = new Error(%q);
		e.code = %q;
		try { e.details = JSON.parse(%q); } catch(x) {}
		return e;
	})()`, msg, code, detailsJSON)

	errVal := qctx.Eval(script)
	if errVal.IsException() {
		return qctx.ThrowError(err)
	}
	return qctx.Throw(errVal)
}

// enrichHandlerErr promotes a JS-handler-thrown BrainkitError captured via
// globalThis.__pending_handler_err into a typed Go error through the wire
// envelope path. If no pending err is set, returns the fallback error
// unchanged. Reading clears the pending slot so it doesn't leak into the
// next handler invocation.
func (k *Kernel) enrichHandlerErr(qctx *quickjs.Context, fallback error) error {
	probe := qctx.Eval(`(function(){
		var e = globalThis.__pending_handler_err;
		globalThis.__pending_handler_err = null;
		return e ? JSON.stringify(e) : "";
	})()`)
	if probe.IsException() {
		return fallback
	}
	raw := probe.String()
	probe.Free()
	if raw == "" {
		return fallback
	}
	var captured struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal([]byte(raw), &captured); err != nil || captured.Code == "" {
		return fallback
	}
	// Build a synthetic envelope and decode through the canonical path so
	// Go-side callers see the right typed error class.
	return sdkFromEnvelopeFallback(captured.Code, captured.Message, captured.Details, fallback)
}

// sdkFromEnvelopeFallback runs sdk.FromEnvelope with the given fields and
// returns the result; if sdk.FromEnvelope returns nil (shouldn't for a
// non-nil code), returns the fallback.
func sdkFromEnvelopeFallback(code, message string, details map[string]any, fallback error) error {
	env := sdkEnvelopeErr(code, message, details)
	if out := sdkFromEnvelope(env); out != nil {
		return out
	}
	return fallback
}

// redactCredentials strips sensitive fields (API keys, tokens, passwords, secrets)
// from a config struct before returning it to JS. Marshal → strip → unmarshal.
func redactCredentials(config any) any {
	raw, err := json.Marshal(config)
	if err != nil {
		return config
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return config
	}
	sensitiveKeys := map[string]bool{
		"APIKey": true, "apiKey": true, "api_key": true,
		"AuthToken": true, "authToken": true, "auth_token": true,
		"AccessKey": true, "accessKey": true, "access_key": true,
		"SecretKey": true, "secretKey": true, "secret_key": true,
		"Password": true, "password": true,
		"Token": true, "token": true,
		"AdminKey": true, "adminKey": true,
	}
	for k := range m {
		if sensitiveKeys[k] {
			if s, ok := m[k].(string); ok && len(s) > 0 {
				m[k] = "****"
			}
		}
	}
	return m
}
