package engine

import (
	"context"
	"encoding/json"

	quickjs "github.com/buke/quickjs-go"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	provreg "github.com/brainlet/brainkit/internal/providers"
)

// registerControlBridges adds __go_brainkit_control for local-only registration operations
// (tools.register, tools.unregister, agents.register, agents.unregister, registry.register, registry.unregister).
func (k *Kernel) registerControlBridges(qctx *quickjs.Context) {
	qctx.Globals().Set(js.JSBridgeControl,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "brainkit_control: expected 2 args (action, payload)"})
			}
			action := args[0].String()
			payload := json.RawMessage(args[1].String())

			// RBAC enforcement on registration
			source := k.currentDeploymentSource()
			if action == "tools.register" || action == "tools.unregister" {
				if err := k.checkRegistrationPermission(source, "tool"); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
			} else if action == "agents.register" || action == "agents.unregister" {
				if err := k.checkRegistrationPermission(source, "agent"); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
			}

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
					return k.throwBrainkitError(qctx, err)
				}
				fullName, regErr := k.toolsDomain.Register(context.Background(), req.Name, req.Description, req.InputSchema, k.callerID)
				if regErr != nil {
					return k.throwBrainkitError(qctx, regErr)
				}
				resp, _ = json.Marshal(map[string]string{"registered": fullName})
			case "tools.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				if err = k.toolsDomain.Unregister(context.Background(), req.Name); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "agents.register":
				var req AgentInfo
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				if err = k.agentsDomain.Register(context.Background(), req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]string{"registered": req.Name})
			case "agents.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				if err = k.agentsDomain.Unregister(context.Background(), req.Name); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "registry.register":
				var req struct {
					Category string          `json:"category"`
					Name     string          `json:"name"`
					Config   json.RawMessage `json:"config"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				var typeHolder struct {
					Type string `json:"type"`
				}
				json.Unmarshal(req.Config, &typeHolder)
				switch req.Category {
				case "provider":
					k.providers.RegisterAIProvider(req.Name, provreg.AIProviderRegistration{
						Type: provreg.AIProviderType(typeHolder.Type),
					})
				case "vectorStore":
					k.providers.RegisterVectorStore(req.Name, provreg.VectorStoreRegistration{
						Type: provreg.VectorStoreType(typeHolder.Type),
					})
				case "storage":
					k.providers.RegisterStorage(req.Name, provreg.StorageRegistration{
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
					return k.throwBrainkitError(qctx, err)
				}
				switch req.Category {
				case "provider":
					k.providers.UnregisterAIProvider(req.Name)
				case "vectorStore":
					k.providers.UnregisterVectorStore(req.Name)
				case "storage":
					k.providers.UnregisterStorage(req.Name)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			default:
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "action", Message: "unknown control action: " + action})
			}
			return qctx.NewString(string(resp))
		}))
}
