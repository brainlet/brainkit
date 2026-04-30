package jsruntime

import (
	"context"
	"encoding/json"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/modulehost/agenthost"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	quickjs "github.com/buke/quickjs-go"
)

// registerControlBridges adds __go_brainkit_control for local-only registration operations
// (tools.register, tools.unregister, agents.register, agents.unregister, registry.register, registry.unregister).
func (r *Runtime) registerControlBridges(qctx *quickjs.Context) {
	qctx.Globals().Set(js.JSBridgeControl,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "brainkit_control: expected 2 args (action, payload)"})
			}
			action := args[0].String()
			payload := json.RawMessage(args[1].String())

			var resp json.RawMessage
			var err error
			switch action {
			case "tools.register":
				var req struct {
					Name        string          `json:"name"`
					Description string          `json:"description"`
					InputSchema json.RawMessage `json:"inputSchema"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				fullName, regErr := r.toolAgents.ToolsDomain().Register(context.Background(), req.Name, req.Description, req.InputSchema, r.core.CallerID())
				if regErr != nil {
					return r.throwBrainkitError(qctx, regErr)
				}
				resp, _ = json.Marshal(map[string]string{"registered": fullName})
			case "tools.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				if err = r.toolAgents.ToolsDomain().Unregister(context.Background(), req.Name); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "agents.register":
				var req agenthost.AgentInfo
				if err = json.Unmarshal(payload, &req); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				if err = r.toolAgents.AgentsDomain().Register(context.Background(), req); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]string{"registered": req.Name})
			case "agents.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				if err = r.toolAgents.AgentsDomain().Unregister(context.Background(), req.Name); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "registry.register":
				var req struct {
					Category string          `json:"category"`
					Name     string          `json:"name"`
					Config   json.RawMessage `json:"config"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				var typeHolder struct {
					Type string `json:"type"`
				}
				json.Unmarshal(req.Config, &typeHolder)
				switch req.Category {
				case "provider":
					r.registry.ProviderRegistry().RegisterAIProvider(req.Name, provreg.AIProviderRegistration{
						Type: provreg.AIProviderType(typeHolder.Type),
					})
				case "vectorStore":
					r.registry.ProviderRegistry().RegisterVectorStore(req.Name, provreg.VectorStoreRegistration{
						Type: provreg.VectorStoreType(typeHolder.Type),
					})
				case "storage":
					r.registry.ProviderRegistry().RegisterStorage(req.Name, provreg.StorageRegistration{
						Type: provreg.StorageType(typeHolder.Type),
					})
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "registry.unregister":
				var req struct {
					Category string `json:"category"`
					Name     string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				switch req.Category {
				case "provider":
					r.registry.ProviderRegistry().UnregisterAIProvider(req.Name)
				case "vectorStore":
					r.registry.ProviderRegistry().UnregisterVectorStore(req.Name)
				case "storage":
					r.registry.ProviderRegistry().UnregisterStorage(req.Name)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			default:
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "action", Message: "unknown control action: " + action})
			}
			return qctx.NewString(string(resp))
		}))
}
