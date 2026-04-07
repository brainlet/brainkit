package types

// MetricsSnapshot is a point-in-time copy of bus metrics data.
// Defined here so ScalingStrategy doesn't import internal/transport.
type MetricsSnapshot struct {
	Published map[string]int `json:"published"`
	Handled   map[string]int `json:"handled"`
	Errors    map[string]int `json:"errors"`
}

// ScalingStrategy evaluates metrics and pool state to make scaling decisions.
type ScalingStrategy interface {
	Evaluate(metrics MetricsSnapshot, pool PoolInfo) ScalingDecision
}

// ScalingDecision describes a scaling action.
type ScalingDecision struct {
	Action string // "scale-up", "scale-down", "none"
	Delta  int
	Reason string
}

// PoolInfo describes the current state of a pool.
type PoolInfo struct {
	Name    string
	Current int
	Min     int
	Max     int // 0 = unlimited
	Pending int
}
