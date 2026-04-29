// Package registrymsg contains the typed bus API for modules/registry.
package registrymsg

import "encoding/json"

// RegistryHasMsg checks whether a named provider, vector store, or storage
// exists in the runtime registry.
type RegistryHasMsg struct {
	Category string `json:"category"` // "provider", "vectorStore", "storage"
	Name     string `json:"name"`
}

func (RegistryHasMsg) BusTopic() string { return "registry.has" }

// RegistryListMsg lists registry entries in one category.
type RegistryListMsg struct {
	Category string `json:"category"`
}

func (RegistryListMsg) BusTopic() string { return "registry.list" }

// RegistryResolveMsg resolves one registry entry into a redacted config.
type RegistryResolveMsg struct {
	Category string `json:"category"`
	Name     string `json:"name"`
}

func (RegistryResolveMsg) BusTopic() string { return "registry.resolve" }

// RegistryHasResp reports existence of one registry entry.
type RegistryHasResp struct {
	Found bool `json:"found"`
}

// RegistryListResp carries a JSON array of category-specific registry entries.
type RegistryListResp struct {
	Items json.RawMessage `json:"items"`
}

// RegistryResolveResp carries one category-specific redacted config object.
type RegistryResolveResp struct {
	Config json.RawMessage `json:"config"`
}

// ProviderAddMsg dynamically adds an AI provider registration.
type ProviderAddMsg struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`   // "openai", "anthropic", etc.
	Config json.RawMessage `json:"config"` // provider-specific config JSON
}

func (ProviderAddMsg) BusTopic() string { return "providers.add" }

// ProviderAddResp reports provider registration success.
type ProviderAddResp struct {
	Added bool `json:"added"`
}

// ProviderRemoveMsg dynamically removes an AI provider registration.
type ProviderRemoveMsg struct {
	Name string `json:"name"`
}

func (ProviderRemoveMsg) BusTopic() string { return "providers.remove" }

// ProviderRemoveResp reports provider removal success.
type ProviderRemoveResp struct {
	Removed bool `json:"removed"`
}

// StorageAddMsg dynamically adds a storage registration.
type StorageAddMsg struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`   // "sqlite", "postgres", "mongodb", "upstash", "memory"
	Config json.RawMessage `json:"config"` // storage-specific config JSON
}

func (StorageAddMsg) BusTopic() string { return "storages.add" }

// StorageAddResp reports storage registration success.
type StorageAddResp struct {
	Added bool `json:"added"`
}

// StorageRemoveMsg dynamically removes a storage registration.
type StorageRemoveMsg struct {
	Name string `json:"name"`
}

func (StorageRemoveMsg) BusTopic() string { return "storages.remove" }

// StorageRemoveResp reports storage removal success.
type StorageRemoveResp struct {
	Removed bool `json:"removed"`
}

// VectorAddMsg dynamically adds a vector store registration.
type VectorAddMsg struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"` // "sqlite", "pgvector", "mongodb"
	Config json.RawMessage `json:"config"`
}

func (VectorAddMsg) BusTopic() string { return "vectors.add" }

// VectorAddResp reports vector store registration success.
type VectorAddResp struct {
	Added bool `json:"added"`
}

// VectorRemoveMsg dynamically removes a vector store registration.
type VectorRemoveMsg struct {
	Name string `json:"name"`
}

func (VectorRemoveMsg) BusTopic() string { return "vectors.remove" }

// VectorRemoveResp reports vector store removal success.
type VectorRemoveResp struct {
	Removed bool `json:"removed"`
}
