package brainkit

import "encoding/json"

// Result is the generic return from sandbox Eval.
type Result struct {
	Value json.RawMessage
	Text  string
}

// ResourceInfo describes a tracked resource in the Kit.
type ResourceInfo struct {
	Type      string `json:"type"`      // "agent", "tool", "workflow", "memory", "harness"
	ID        string `json:"id"`        // unique within type
	Name      string `json:"name"`      // display name
	Source    string `json:"source"`    // .ts filename that created it
	CreatedAt int64  `json:"createdAt"` // unix timestamp ms
}
