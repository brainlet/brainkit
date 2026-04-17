package sdk

import "encoding/json"

// ── Vector Store Management ──

type VectorAddMsg struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"` // "sqlite", "pgvector", "mongodb"
	Config json.RawMessage `json:"config"`
}

func (VectorAddMsg) BusTopic() string { return "vectors.add" }

type VectorAddResp struct {
	Added bool `json:"added"`
}

type VectorRemoveMsg struct {
	Name string `json:"name"`
}

func (VectorRemoveMsg) BusTopic() string { return "vectors.remove" }

type VectorRemoveResp struct {
	Removed bool `json:"removed"`
}
