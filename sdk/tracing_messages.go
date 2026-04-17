package sdk

import "encoding/json"

// ── Trace Queries ──

type TraceGetMsg struct {
	TraceID string `json:"traceId"`
}

func (TraceGetMsg) BusTopic() string { return "trace.get" }

type TraceGetResp struct {
	Spans json.RawMessage `json:"spans"` // []Span as JSON
}

type TraceListMsg struct {
	Source      string `json:"source,omitempty"`
	Status      string `json:"status,omitempty"` // "error" to filter
	MinDuration int    `json:"minDuration,omitempty"` // ms
	Limit       int    `json:"limit,omitempty"`
}

func (TraceListMsg) BusTopic() string { return "trace.list" }

type TraceListResp struct {
	Traces json.RawMessage `json:"traces"` // []TraceSummary as JSON
}
