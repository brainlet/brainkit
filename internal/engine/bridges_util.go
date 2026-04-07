package engine

import (
	"encoding/json"
	"errors"
	"fmt"

	quickjs "github.com/buke/quickjs-go"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/sdkerrors"
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

// throwBrainkitError constructs a JS error and throws it.
// Encodes code and details INTO the error message string as "[CODE] message {{details_json}}".
// The rewrapErrors wrappers in kit_runtime.js parse this back out when the error
// crosses the SES Compartment boundary (where custom properties like .code are stripped).
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

	// Encode code + details in message: "[PERMISSION_DENIED] message {{json}}"
	// rewrapErrors in kit_runtime.js parses this format back into BrainkitError.
	encodedMsg := "[" + code + "] " + msg
	if detailsJSON != "{}" {
		encodedMsg += " {{" + detailsJSON + "}}"
	}

	script := fmt.Sprintf(`(function() {
		var e = new Error(%q);
		e.code = %q;
		try { e.details = JSON.parse(%q); } catch(x) {}
		return e;
	})()`, encodedMsg, code, detailsJSON)

	errVal := qctx.Eval(script)
	if errVal.IsException() {
		// Fallback if JS construction fails
		return qctx.ThrowError(err)
	}
	return qctx.Throw(errVal)
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
