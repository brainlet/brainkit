package sdk

import "encoding/json"

// ── Requests ──

type RegistryHasMsg struct {
	Category string `json:"category"` // "provider", "vectorStore", "storage"
	Name     string `json:"name"`
}

func (RegistryHasMsg) BusTopic() string { return "registry.has" }

type RegistryListMsg struct {
	Category string `json:"category"`
}

func (RegistryListMsg) BusTopic() string { return "registry.list" }

type RegistryResolveMsg struct {
	Category string `json:"category"`
	Name     string `json:"name"`
}

func (RegistryResolveMsg) BusTopic() string { return "registry.resolve" }

// ── Responses ──

type RegistryHasResp struct {
	ResultMeta
	Found bool `json:"found"`
}


type RegistryListResp struct {
	ResultMeta
	Items json.RawMessage `json:"items"`
}


type RegistryResolveResp struct {
	ResultMeta
	Config json.RawMessage `json:"config"`
}

