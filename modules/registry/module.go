// Package registry owns the registry.*, providers.*, storages.*, and vectors.*
// admin bus commands as a hot-mountable Kit module.
package registry

import (
	"context"
	"encoding/json"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Module exposes registry and dynamic provider/storage/vector management
// commands. Construct via New and include in brainkit.Config.Modules when the
// runtime should accept those admin commands over the bus.
type Module struct {
	providers *provreg.ProviderRegistry
	mutations bkmodule.RegistryMutationManager
}

// New creates the registry module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "registry" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers registry/provider/storage/vector command handlers.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	providers, err := bkmodule.RequireCapability[*provreg.ProviderRegistry](host, bkmodule.CapabilityProviderRegistry)
	if err != nil {
		return fmt.Errorf("registry: %w", err)
	}
	mutations, err := bkmodule.RequireCapability[bkmodule.RegistryMutationManager](host, bkmodule.CapabilityRegistryMutation)
	if err != nil {
		return fmt.Errorf("registry: %w", err)
	}

	m.providers = providers
	m.mutations = mutations
	host.Scope().Defer(func(context.Context) error {
		m.providers = nil
		m.mutations = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.Has),
		bkmodule.Command(m.List),
		bkmodule.Command(m.Resolve),
		bkmodule.Command(m.AddProvider),
		bkmodule.Command(m.RemoveProvider),
		bkmodule.Command(m.AddStorage),
		bkmodule.Command(m.RemoveStorage),
		bkmodule.Command(m.AddVector),
		bkmodule.Command(m.RemoveVector),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

// Close detaches the module from Kit capabilities. Command handles are owned by
// the module scope, so unmounting unregisters them.
func (m *Module) Close() error {
	m.providers = nil
	m.mutations = nil
	return nil
}

// Factory is the registered ModuleFactory for registry.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the registry module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "registry",
		Status:  bkmodule.StatusStable,
		Summary: "Registry/provider/storage/vector admin bus commands.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[registrymsg.ProviderAddMsg, registrymsg.ProviderAddResp](),
			bkmodule.CommandMessage[registrymsg.ProviderRemoveMsg, registrymsg.ProviderRemoveResp](),
			bkmodule.CommandMessage[registrymsg.RegistryHasMsg, registrymsg.RegistryHasResp](),
			bkmodule.CommandMessage[registrymsg.RegistryListMsg, registrymsg.RegistryListResp](),
			bkmodule.CommandMessage[registrymsg.RegistryResolveMsg, registrymsg.RegistryResolveResp](),
			bkmodule.CommandMessage[registrymsg.StorageAddMsg, registrymsg.StorageAddResp](),
			bkmodule.CommandMessage[registrymsg.StorageRemoveMsg, registrymsg.StorageRemoveResp](),
			bkmodule.CommandMessage[registrymsg.VectorAddMsg, registrymsg.VectorAddResp](),
			bkmodule.CommandMessage[registrymsg.VectorRemoveMsg, registrymsg.VectorRemoveResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[*provreg.ProviderRegistry](bkmodule.CapabilityProviderRegistry),
			bkmodule.RequiredCapabilityOf[bkmodule.RegistryMutationManager](bkmodule.CapabilityRegistryMutation),
		},
	}
}

func init() { bkmodule.Register("registry", Factory{}) }

// Has handles registry.has.
func (m *Module) Has(_ context.Context, req registrymsg.RegistryHasMsg) (*registrymsg.RegistryHasResp, error) {
	var found bool
	switch req.Category {
	case "provider":
		found = m.providers.HasAIProvider(req.Name)
	case "vectorStore":
		found = m.providers.HasVectorStore(req.Name)
	case "storage":
		found = m.providers.HasStorage(req.Name)
	}
	return &registrymsg.RegistryHasResp{Found: found}, nil
}

// List handles registry.list.
func (m *Module) List(_ context.Context, req registrymsg.RegistryListMsg) (*registrymsg.RegistryListResp, error) {
	var result any
	switch req.Category {
	case "provider":
		result = m.providers.ListAIProviders()
	case "vectorStore":
		result = m.providers.ListVectorStores()
	case "storage":
		result = m.providers.ListStorages()
	default:
		result = []any{}
	}
	b, _ := json.Marshal(result)
	return &registrymsg.RegistryListResp{Items: b}, nil
}

// Resolve handles registry.resolve.
func (m *Module) Resolve(_ context.Context, req registrymsg.RegistryResolveMsg) (*registrymsg.RegistryResolveResp, error) {
	var configJSON []byte
	switch req.Category {
	case "provider":
		if reg, ok := m.providers.GetAIProvider(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
		}
	case "vectorStore":
		if reg, ok := m.providers.GetVectorStore(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
		}
	case "storage":
		if reg, ok := m.providers.GetStorage(req.Name); ok {
			configJSON, _ = json.Marshal(map[string]any{"type": string(reg.Type), "name": req.Name, "config": redactCredentials(reg.Config)})
		}
	}
	return &registrymsg.RegistryResolveResp{Config: configJSON}, nil
}

// AddProvider handles providers.add.
func (m *Module) AddProvider(ctx context.Context, req registrymsg.ProviderAddMsg) (*registrymsg.ProviderAddResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.mutations.AddRegistryProvider(ctx, req.Name, req.Type, req.Config); err != nil {
		return nil, err
	}
	return &registrymsg.ProviderAddResp{Added: true}, nil
}

// RemoveProvider handles providers.remove.
func (m *Module) RemoveProvider(ctx context.Context, req registrymsg.ProviderRemoveMsg) (*registrymsg.ProviderRemoveResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.mutations.RemoveRegistryProvider(ctx, req.Name); err != nil {
		return nil, err
	}
	return &registrymsg.ProviderRemoveResp{Removed: true}, nil
}

// AddStorage handles storages.add.
func (m *Module) AddStorage(ctx context.Context, req registrymsg.StorageAddMsg) (*registrymsg.StorageAddResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.mutations.AddRegistryStorage(ctx, req.Name, req.Type, req.Config); err != nil {
		return nil, err
	}
	return &registrymsg.StorageAddResp{Added: true}, nil
}

// RemoveStorage handles storages.remove.
func (m *Module) RemoveStorage(ctx context.Context, req registrymsg.StorageRemoveMsg) (*registrymsg.StorageRemoveResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.mutations.RemoveRegistryStorage(ctx, req.Name); err != nil {
		return nil, err
	}
	return &registrymsg.StorageRemoveResp{Removed: true}, nil
}

// AddVector handles vectors.add.
func (m *Module) AddVector(ctx context.Context, req registrymsg.VectorAddMsg) (*registrymsg.VectorAddResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.mutations.AddRegistryVector(ctx, req.Name, req.Type, req.Config); err != nil {
		return nil, err
	}
	return &registrymsg.VectorAddResp{Added: true}, nil
}

// RemoveVector handles vectors.remove.
func (m *Module) RemoveVector(ctx context.Context, req registrymsg.VectorRemoveMsg) (*registrymsg.VectorRemoveResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.mutations.RemoveRegistryVector(ctx, req.Name); err != nil {
		return nil, err
	}
	return &registrymsg.VectorRemoveResp{Removed: true}, nil
}

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
