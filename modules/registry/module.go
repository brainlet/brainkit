// Package registry owns the registry.*, providers.*, storages.*, and vectors.*
// admin bus commands as a hot-mountable Kit module.
package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	provreg "github.com/brainlet/brainkit/modules/registry/providerreg"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Module exposes registry and dynamic provider/storage/vector management
// commands. Construct via New and include in brainkit.Config.Modules when the
// runtime should accept those admin commands over the bus.
type Module struct {
	providers *provreg.ProviderRegistry
	storages  storageManager
}

type storageManager interface {
	AddStorage(name string, cfg types.StorageConfig) error
	RemoveStorage(name string) error
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
	storages, err := bkmodule.RequireCapability[storageManager](host, bkmodule.CapabilityStorageManager)
	if err != nil {
		return fmt.Errorf("registry: %w", err)
	}

	m.providers = providers
	m.storages = storages
	host.Scope().Defer(func(context.Context) error {
		m.providers = nil
		m.storages = nil
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
	m.storages = nil
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

// Describe surfaces module metadata for `brainkit modules list`.
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
			bkmodule.RequiredCapabilityOf[storageManager](bkmodule.CapabilityStorageManager),
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
func (m *Module) AddProvider(_ context.Context, req registrymsg.ProviderAddMsg) (*registrymsg.ProviderAddResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	config, err := deserializeProviderConfig(req.Type, req.Config)
	if err != nil {
		return nil, err
	}
	if err := m.providers.RegisterAIProvider(req.Name, provreg.AIProviderRegistration{
		Type:   provreg.AIProviderType(req.Type),
		Config: config,
	}); err != nil {
		return nil, err
	}
	return &registrymsg.ProviderAddResp{Added: true}, nil
}

// RemoveProvider handles providers.remove.
func (m *Module) RemoveProvider(_ context.Context, req registrymsg.ProviderRemoveMsg) (*registrymsg.ProviderRemoveResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	m.providers.UnregisterAIProvider(req.Name)
	return &registrymsg.ProviderRemoveResp{Removed: true}, nil
}

// AddStorage handles storages.add.
func (m *Module) AddStorage(_ context.Context, req registrymsg.StorageAddMsg) (*registrymsg.StorageAddResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	cfg, err := deserializeStorageConfig(req.Type, req.Config)
	if err != nil {
		return nil, err
	}
	if err := m.storages.AddStorage(req.Name, cfg); err != nil {
		return nil, err
	}
	return &registrymsg.StorageAddResp{Added: true}, nil
}

// RemoveStorage handles storages.remove.
func (m *Module) RemoveStorage(_ context.Context, req registrymsg.StorageRemoveMsg) (*registrymsg.StorageRemoveResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.storages.RemoveStorage(req.Name); err != nil {
		return nil, err
	}
	return &registrymsg.StorageRemoveResp{Removed: true}, nil
}

// AddVector handles vectors.add. The existing command registers the requested
// type verbatim; richer runtime vector bridge management is a separate module
// boundary from this bus extraction.
func (m *Module) AddVector(_ context.Context, req registrymsg.VectorAddMsg) (*registrymsg.VectorAddResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	if err := m.providers.RegisterVectorStore(req.Name, provreg.VectorStoreRegistration{
		Type: provreg.VectorStoreType(req.Type),
	}); err != nil {
		return nil, err
	}
	return &registrymsg.VectorAddResp{Added: true}, nil
}

// RemoveVector handles vectors.remove.
func (m *Module) RemoveVector(_ context.Context, req registrymsg.VectorRemoveMsg) (*registrymsg.VectorRemoveResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	m.providers.UnregisterVectorStore(req.Name)
	return &registrymsg.VectorRemoveResp{Removed: true}, nil
}

func deserializeProviderConfig(typ string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	switch typ {
	case "openai":
		var c types.OpenAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "anthropic":
		var c types.AnthropicProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "google":
		var c types.GoogleProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "mistral":
		var c types.MistralProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "cohere":
		var c types.CohereProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "groq":
		var c types.GroqProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "perplexity":
		var c types.PerplexityProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "deepseek":
		var c types.DeepSeekProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "fireworks":
		var c types.FireworksProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "togetherai":
		var c types.TogetherAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "xai":
		var c types.XAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "azure":
		var c types.AzureProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "bedrock":
		var c types.BedrockProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "vertex":
		var c types.VertexProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "huggingface":
		var c types.HuggingFaceProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "cerebras":
		var c types.CerebrasProviderConfig
		return c, json.Unmarshal(raw, &c)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", typ)
	}
}

func deserializeStorageConfig(typ string, raw json.RawMessage) (types.StorageConfig, error) {
	var base struct {
		Path             string `json:"path"`
		ConnectionString string `json:"connectionString"`
		URI              string `json:"uri"`
		DBName           string `json:"dbName"`
		URL              string `json:"url"`
		Token            string `json:"token"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &base); err != nil {
			return types.StorageConfig{}, fmt.Errorf("invalid storage config: %w", err)
		}
	}
	return types.StorageConfig{
		Type:             typ,
		Path:             base.Path,
		ConnectionString: base.ConnectionString,
		URI:              base.URI,
		DBName:           base.DBName,
		URL:              base.URL,
		Token:            base.Token,
	}, nil
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
