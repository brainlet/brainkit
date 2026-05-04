package jsruntime

import (
	"context"
	"encoding/json"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/types"
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
				if err = json.Unmarshal(req.Config, &typeHolder); err != nil {
					return r.throwBrainkitError(qctx, err)
				}
				switch req.Category {
				case "provider":
					cfg, cfgErr := provreg.DecodeAIProviderConfig(typeHolder.Type, req.Config)
					if cfgErr != nil {
						return r.throwBrainkitError(qctx, cfgErr)
					}
					if err = r.registry.ProviderRegistry().RegisterAIProvider(req.Name, provreg.AIProviderRegistration{
						Type:   provreg.AIProviderType(typeHolder.Type),
						Config: cfg,
					}); err != nil {
						return r.throwBrainkitError(qctx, err)
					}
				case "vectorStore":
					cfg, cfgErr := decodeBridgeVectorConfig(typeHolder.Type, req.Config)
					if cfgErr != nil {
						return r.throwBrainkitError(qctx, cfgErr)
					}
					if err = r.storage.AddVector(req.Name, cfg); err != nil {
						return r.throwBrainkitError(qctx, err)
					}
				case "storage":
					cfg, cfgErr := decodeBridgeStorageConfig(typeHolder.Type, req.Config)
					if cfgErr != nil {
						return r.throwBrainkitError(qctx, cfgErr)
					}
					if err = r.storage.AddStorage(req.Name, cfg); err != nil {
						return r.throwBrainkitError(qctx, err)
					}
				default:
					return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "category", Message: "unknown registry category: " + req.Category})
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
					if err = r.storage.RemoveVector(req.Name); err != nil {
						return r.throwBrainkitError(qctx, err)
					}
				case "storage":
					if err = r.storage.RemoveStorage(req.Name); err != nil {
						return r.throwBrainkitError(qctx, err)
					}
				default:
					return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "category", Message: "unknown registry category: " + req.Category})
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			default:
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "action", Message: "unknown control action: " + action})
			}
			return qctx.NewString(string(resp))
		}))
}

func decodeBridgeStorageConfig(typ string, raw json.RawMessage) (types.StorageConfig, error) {
	var cfg struct {
		Type             string `json:"type"`
		Path             string `json:"path"`
		ConnectionString string `json:"connectionString"`
		URI              string `json:"uri"`
		DBName           string `json:"dbName"`
		URL              string `json:"url"`
		Token            string `json:"token"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return types.StorageConfig{}, err
		}
	}
	if cfg.Type == "" {
		cfg.Type = typ
	}
	return types.StorageConfig{
		Type:             cfg.Type,
		Path:             cfg.Path,
		ConnectionString: cfg.ConnectionString,
		URI:              cfg.URI,
		DBName:           cfg.DBName,
		URL:              cfg.URL,
		Token:            cfg.Token,
	}, nil
}

func decodeBridgeVectorConfig(typ string, raw json.RawMessage) (types.VectorConfig, error) {
	var cfg struct {
		Type             string `json:"type"`
		Path             string `json:"path"`
		ConnectionString string `json:"connectionString"`
		URI              string `json:"uri"`
		DBName           string `json:"dbName"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return types.VectorConfig{}, err
		}
	}
	if cfg.Type == "" {
		cfg.Type = typ
	}
	return types.VectorConfig{
		Type:             cfg.Type,
		Path:             cfg.Path,
		ConnectionString: cfg.ConnectionString,
		URI:              cfg.URI,
		DBName:           cfg.DBName,
	}, nil
}
