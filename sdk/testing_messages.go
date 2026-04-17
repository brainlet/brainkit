package sdk

import "encoding/json"

// ── Testing ──

type TestRunMsg struct {
	Dir     string `json:"dir"`
	Pattern string `json:"pattern,omitempty"` // default "*.test.ts"
	SkipAI  bool   `json:"skipAI,omitempty"`
}

func (TestRunMsg) BusTopic() string { return "test.run" }

type TestRunResp struct {
	Results json.RawMessage `json:"results"`
}
