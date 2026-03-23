package kit

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk/messages"
)

// RegistryDomain handles registry query operations via the command catalog.
// This makes registry operations available from ALL surfaces (Go, TS, WASM, Plugin).
type RegistryDomain struct {
	kit *Kernel
}

func newRegistryDomain(k *Kernel) *RegistryDomain {
	return &RegistryDomain{kit: k}
}

func (d *RegistryDomain) Has(_ context.Context, req messages.RegistryHasMsg) (*messages.RegistryHasResp, error) {
	var found bool
	switch req.Category {
	case "provider":
		found = d.kit.providers.HasAIProvider(req.Name)
	case "vectorStore":
		found = d.kit.providers.HasVectorStore(req.Name)
	case "storage":
		found = d.kit.providers.HasStorage(req.Name)
	}
	return &messages.RegistryHasResp{Found: found}, nil
}

func (d *RegistryDomain) List(_ context.Context, req messages.RegistryListMsg) (*messages.RegistryListResp, error) {
	var result any
	switch req.Category {
	case "provider":
		result = d.kit.providers.ListAIProviders()
	case "vectorStore":
		result = d.kit.providers.ListVectorStores()
	case "storage":
		result = d.kit.providers.ListStorages()
	default:
		result = []any{}
	}
	b, _ := json.Marshal(result)
	return &messages.RegistryListResp{Items: b}, nil
}

func (d *RegistryDomain) Resolve(_ context.Context, req messages.RegistryResolveMsg) (*messages.RegistryResolveResp, error) {
	var configJSON []byte
	switch req.Category {
	case "provider":
		if reg, ok := d.kit.providers.GetAIProvider(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{
				"type": string(reg.Type), "name": req.Name, "config": reg.Config,
			})
		}
	case "vectorStore":
		if reg, ok := d.kit.providers.GetVectorStore(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{
				"type": string(reg.Type), "name": req.Name, "config": reg.Config,
			})
		}
	case "storage":
		if reg, ok := d.kit.providers.GetStorage(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{
				"type": string(reg.Type), "name": req.Name, "config": reg.Config,
			})
		}
	}
	return &messages.RegistryResolveResp{Config: configJSON}, nil
}
