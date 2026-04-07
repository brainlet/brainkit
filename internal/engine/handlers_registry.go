package engine

import (
	"context"
	"encoding/json"

	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/sdk/messages"
)

// RegistryDomain handles registry.has, registry.list, registry.resolve bus commands.
type RegistryDomain struct {
	providers *provreg.ProviderRegistry
}

func newRegistryDomain(providers *provreg.ProviderRegistry) *RegistryDomain {
	return &RegistryDomain{providers: providers}
}

func (d *RegistryDomain) Has(_ context.Context, req messages.RegistryHasMsg) (*messages.RegistryHasResp, error) {
	var found bool
	switch req.Category {
	case "provider":
		found = d.providers.HasAIProvider(req.Name)
	case "vectorStore":
		found = d.providers.HasVectorStore(req.Name)
	case "storage":
		found = d.providers.HasStorage(req.Name)
	}
	return &messages.RegistryHasResp{Found: found}, nil
}

func (d *RegistryDomain) List(_ context.Context, req messages.RegistryListMsg) (*messages.RegistryListResp, error) {
	var result any
	switch req.Category {
	case "provider":
		result = d.providers.ListAIProviders()
	case "vectorStore":
		result = d.providers.ListVectorStores()
	case "storage":
		result = d.providers.ListStorages()
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
		if reg, ok := d.providers.GetAIProvider(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
		}
	case "vectorStore":
		if reg, ok := d.providers.GetVectorStore(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
		}
	case "storage":
		if reg, ok := d.providers.GetStorage(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
		}
	}
	return &messages.RegistryResolveResp{Config: configJSON}, nil
}
