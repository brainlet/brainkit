package jsruntime

import (
	"context"
	"encoding/json"
	"fmt"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulehost/providerhost"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

func (r *Runtime) initJSRuntimeGlobals(cfg types.KernelConfig) error {
	if r.bridge == nil {
		return fmt.Errorf("brainkit: js runtime bridge is not initialized")
	}
	obsEnabled := cfg.Observability.Enabled == nil || *cfg.Observability.Enabled
	obsStrategy := cfg.Observability.Strategy
	if obsStrategy == "" {
		obsStrategy = "realtime"
	}
	obsServiceName := cfg.Observability.ServiceName
	if obsServiceName == "" {
		obsServiceName = "brainkit"
	}
	r.bridge.Eval("__obs_config.js", fmt.Sprintf(
		`globalThis.`+js.JSObsConfig+` = { enabled: %v, strategy: %q, serviceName: %q }`,
		obsEnabled, obsStrategy, obsServiceName,
	))

	if len(cfg.AIProviders) > 0 {
		provMap := make(map[string]map[string]string)
		for name, reg := range cfg.AIProviders {
			creds := providerhost.ExtractProviderCredentials(reg)
			entry := map[string]string{"APIKey": creds.APIKey}
			if creds.BaseURL != "" {
				entry["BaseURL"] = creds.BaseURL
			}
			provMap[name] = entry
		}
		provJSON, _ := json.Marshal(provMap)
		r.bridge.Eval("__providers.js", fmt.Sprintf(
			`globalThis.`+js.JSProviders+` = %s;`, string(provJSON),
		))
	}

	return r.loadRuntime()
}

func (r *Runtime) evalJSCall(ctx context.Context, fn string, args any) (json.RawMessage, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("callJS %s: marshal args: %w", fn, err)
	}
	code := fmt.Sprintf("return JSON.stringify(await %s(JSON.parse(%q)))", fn, string(argsJSON))
	result, err := r.EvalTS(ctx, "__dispatch__.ts", code)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(result), nil
}

func (r *Runtime) evalJSCallSync(fn string, args any) {
	if r.bridge == nil {
		return
	}
	argsJSON, _ := json.Marshal(args)
	r.bridge.Eval("__dispatch_sync__.js", fmt.Sprintf("%s(JSON.parse(%q))", fn, string(argsJSON)))
}

func (r *Runtime) upgradeMastraStorage() {
	raw, err := r.CallJS(context.Background(), "__brainkit.storage.upgrade", nil)
	if err != nil {
		types.InvokeErrorHandler(r.cfg.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "UpgradeMastraStorage", Cause: err,
		}, types.ErrorContext{Operation: "UpgradeMastraStorage", Component: "kernel"})
		return
	}
	var parsed struct {
		Upgraded bool   `json:"upgraded"`
		Storage  string `json:"storage"`
	}
	if json.Unmarshal(raw, &parsed) == nil && parsed.Upgraded {
		r.host.Logger().Info("Mastra storage upgraded", "backend", parsed.Storage)
	}
}
