package sdk

import "encoding/json"

// ── Provider Management ──

type ProviderAddMsg struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`   // "openai", "anthropic", etc.
	Config json.RawMessage `json:"config"` // provider-specific config JSON
}

func (ProviderAddMsg) BusTopic() string { return "providers.add" }

type ProviderAddResp struct {
	ResultMeta
	Added bool `json:"added"`
}

type ProviderRemoveMsg struct {
	Name string `json:"name"`
}

func (ProviderRemoveMsg) BusTopic() string { return "providers.remove" }

type ProviderRemoveResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}
