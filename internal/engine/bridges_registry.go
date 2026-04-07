package engine

import (
	"encoding/json"

	quickjs "github.com/buke/quickjs-go"
	js "github.com/brainlet/brainkit/internal/contract"
)

// registerRegistryBridges adds __go_registry_resolve, __go_registry_has, __go_registry_list bridges.
func (k *Kernel) registerRegistryBridges(qctx *quickjs.Context) {
	// __go_registry_resolve(category, name) → configJSON or ""
	qctx.Globals().Set(js.JSBridgeRegistryResolve,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.NewString("")
			}
			category := args[0].String()
			name := args[1].String()

			var configJSON []byte
			switch category {
			case "provider":
				if reg, ok := k.providers.GetAIProvider(name); ok {
					configJSON, _ = json.Marshal(map[string]any{
						"type":   string(reg.Type),
						"name":   name,
						"config": redactCredentials(reg.Config),
					})
				}
			case "vectorStore":
				if reg, ok := k.providers.GetVectorStore(name); ok {
					configJSON, _ = json.Marshal(map[string]any{
						"type":   string(reg.Type),
						"name":   name,
						"config": redactCredentials(reg.Config),
					})
				}
			case "storage":
				if reg, ok := k.providers.GetStorage(name); ok {
					configJSON, _ = json.Marshal(map[string]any{
						"type":   string(reg.Type),
						"name":   name,
						"config": redactCredentials(reg.Config),
					})
				}
			}
			if configJSON == nil {
				return qctx.NewString("")
			}
			return qctx.NewString(string(configJSON))
		}))

	// __go_registry_has(category, name) → "true" or "false"
	qctx.Globals().Set(js.JSBridgeRegistryHas,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.NewString("false")
			}
			category := args[0].String()
			name := args[1].String()
			var found bool
			switch category {
			case "provider":
				found = k.providers.HasAIProvider(name)
			case "vectorStore":
				found = k.providers.HasVectorStore(name)
			case "storage":
				found = k.providers.HasStorage(name)
			}
			if found {
				return qctx.NewString("true")
			}
			return qctx.NewString("false")
		}))

	// __go_registry_list(category) → JSON array
	qctx.Globals().Set(js.JSBridgeRegistryList,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.NewString("[]")
			}
			category := args[0].String()
			var result any
			switch category {
			case "provider":
				result = k.providers.ListAIProviders()
			case "vectorStore":
				result = k.providers.ListVectorStores()
			case "storage":
				result = k.providers.ListStorages()
			default:
				result = []any{}
			}
			b, _ := json.Marshal(result)
			return qctx.NewString(string(b))
		}))
}
