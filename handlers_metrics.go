package brainkit

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk/messages"
)

// MetricsDomain handles metrics.get bus command.
type MetricsDomain struct {
	kernel *Kernel
}

func newMetricsDomain(kernel *Kernel) *MetricsDomain {
	return &MetricsDomain{kernel: kernel}
}

func (d *MetricsDomain) Get(_ context.Context, _ messages.MetricsGetMsg) (*messages.MetricsGetResp, error) {
	data, _ := json.Marshal(d.kernel.Metrics())
	return &messages.MetricsGetResp{Metrics: data}, nil
}
