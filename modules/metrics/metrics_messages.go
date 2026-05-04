package metrics

import "encoding/json"

// MetricsGetMsg requests a runtime metrics snapshot over the bus.
type MetricsGetMsg struct{}

func (MetricsGetMsg) BusTopic() string { return "metrics.get" }

// MetricsGetResp wraps the JSON-encoded runtime metrics payload.
type MetricsGetResp struct {
	Metrics json.RawMessage `json:"metrics"`
}
