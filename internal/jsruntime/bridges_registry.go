package jsruntime

import (
	"encoding/json"
	"time"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/modulehost/resourcehost"
	quickjs "github.com/buke/quickjs-go"
)

// registerRegistryBridges adds __go_registry_resolve,
// __go_registry_runtime_resolve, __go_registry_has, __go_registry_list, and
// __go_resource_register bridges.
func (r *Runtime) registerRegistryBridges(qctx *quickjs.Context) {
	// __go_registry_resolve(category, name) → configJSON or ""
	qctx.Globals().Set(js.JSBridgeRegistryResolve,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.NewString("")
			}
			return qctx.NewString(r.registryConfigJSON(args[0].String(), args[1].String(), true))
		}))

	// __go_registry_runtime_resolve(category, name) → unredacted configJSON or "".
	// This bridge is for runtime construction only. Public JS registry.resolve
	// keeps using __go_registry_resolve, which redacts credentials before exposing
	// config to user code.
	qctx.Globals().Set(js.JSBridgeRegistryRuntimeResolve,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.NewString("")
			}
			return qctx.NewString(r.registryConfigJSON(args[0].String(), args[1].String(), false))
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
				found = r.registry.ProviderRegistry().HasAIProvider(name)
			case "vectorStore":
				found = r.registry.ProviderRegistry().HasVectorStore(name)
			case "storage":
				found = r.registry.ProviderRegistry().HasStorage(name)
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
				result = r.registry.ProviderRegistry().ListAIProviders()
			case "vectorStore":
				result = r.registry.ProviderRegistry().ListVectorStores()
			case "storage":
				result = r.registry.ProviderRegistry().ListStorages()
			default:
				result = []any{}
			}
			b, _ := json.Marshal(result)
			return qctx.NewString(string(b))
		}))

	// __go_resource_register(type, id, name, source) → registers in Go resource registry
	qctx.Globals().Set(js.JSBridgeResourceRegister,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return qctx.NewUndefined()
			}
			r.deploymentMgr.Resources().Register(resourcehost.Entry{
				Type:      args[0].String(),
				ID:        args[1].String(),
				Name:      args[2].String(),
				Source:    args[3].String(),
				CreatedAt: time.Now(),
			})
			return qctx.NewUndefined()
		}))
}

func (r *Runtime) registryConfigJSON(category, name string, redact bool) string {
	var configJSON []byte
	config := func(v any) any {
		if redact {
			return redactCredentials(v)
		}
		return v
	}
	switch category {
	case "provider":
		if reg, ok := r.registry.ProviderRegistry().GetAIProvider(name); ok {
			configJSON, _ = json.Marshal(map[string]any{
				"type":   string(reg.Type),
				"name":   name,
				"config": config(reg.Config),
			})
		}
	case "vectorStore":
		if reg, ok := r.registry.ProviderRegistry().GetVectorStore(name); ok {
			configJSON, _ = json.Marshal(map[string]any{
				"type":   string(reg.Type),
				"name":   name,
				"config": config(reg.Config),
			})
		}
	case "storage":
		if reg, ok := r.registry.ProviderRegistry().GetStorage(name); ok {
			configJSON, _ = json.Marshal(map[string]any{
				"type":   string(reg.Type),
				"name":   name,
				"config": config(reg.Config),
			})
		}
	}
	return string(configJSON)
}
