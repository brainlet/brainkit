package messages

import "encoding/json"

// ── Storage Management ──

type StorageAddMsg struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`   // "sqlite", "postgres", "mongodb", "upstash", "memory"
	Config json.RawMessage `json:"config"` // storage-specific config JSON
}

func (StorageAddMsg) BusTopic() string { return "storages.add" }

type StorageAddResp struct {
	ResultMeta
	Added bool `json:"added"`
}

type StorageRemoveMsg struct {
	Name string `json:"name"`
}

func (StorageRemoveMsg) BusTopic() string { return "storages.remove" }

type StorageRemoveResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}
