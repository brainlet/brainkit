package brainkit

import (
	"fmt"

	"github.com/brainlet/brainkit/bus"
)

// ScalingStrategy evaluates bus metrics and pool state to make scaling decisions.
type ScalingStrategy interface {
	Evaluate(metrics bus.BusMetrics, pool PoolInfo) ScalingDecision
}

// ScalingDecision describes a scaling action.
type ScalingDecision struct {
	Action string // "scale-up", "scale-down", "none"
	Delta  int    // instances to add (positive) or remove (negative)
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

// StaticStrategy maintains a fixed instance count.
type StaticStrategy struct {
	Target int
}

func NewStaticStrategy(target int) *StaticStrategy {
	return &StaticStrategy{Target: target}
}

func (s *StaticStrategy) Evaluate(_ bus.BusMetrics, pool PoolInfo) ScalingDecision {
	if pool.Current < s.Target {
		return ScalingDecision{
			Action: "scale-up",
			Delta:  s.Target - pool.Current,
			Reason: "static: restore target count",
		}
	}
	if pool.Current > s.Target {
		return ScalingDecision{
			Action: "scale-down",
			Delta:  pool.Current - s.Target,
			Reason: "static: reduce to target count",
		}
	}
	return ScalingDecision{Action: "none"}
}

// ThresholdStrategy scales based on pending message count.
type ThresholdStrategy struct {
	ScaleUpThreshold   int
	ScaleDownThreshold int
	ScaleUpStep        int
	ScaleDownStep      int
}

func NewThresholdStrategy(scaleUp, scaleDown int) *ThresholdStrategy {
	return &ThresholdStrategy{
		ScaleUpThreshold:   scaleUp,
		ScaleDownThreshold: scaleDown,
		ScaleUpStep:        1,
		ScaleDownStep:      1,
	}
}

func (s *ThresholdStrategy) Evaluate(_ bus.BusMetrics, pool PoolInfo) ScalingDecision {
	pending := pool.Pending

	if pending > s.ScaleUpThreshold {
		if pool.Max > 0 && pool.Current >= pool.Max {
			return ScalingDecision{Action: "none", Reason: "threshold: at max instances"}
		}
		delta := s.ScaleUpStep
		if pool.Max > 0 && pool.Current+delta > pool.Max {
			delta = pool.Max - pool.Current
		}
		if delta <= 0 {
			return ScalingDecision{Action: "none"}
		}
		return ScalingDecision{
			Action: "scale-up",
			Delta:  delta,
			Reason: fmt.Sprintf("threshold: pending=%d > threshold=%d", pending, s.ScaleUpThreshold),
		}
	}

	if pending < s.ScaleDownThreshold && pool.Current > pool.Min {
		delta := s.ScaleDownStep
		if pool.Current-delta < pool.Min {
			delta = pool.Current - pool.Min
		}
		if delta <= 0 {
			return ScalingDecision{Action: "none"}
		}
		return ScalingDecision{
			Action: "scale-down",
			Delta:  delta,
			Reason: fmt.Sprintf("threshold: pending=%d < threshold=%d", pending, s.ScaleDownThreshold),
		}
	}

	return ScalingDecision{Action: "none"}
}
