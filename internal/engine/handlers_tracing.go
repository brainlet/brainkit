package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/sdk/messages"
)

// TracingDomain handles trace.get and trace.list bus commands.
type TracingDomain struct {
	store tracing.TraceStore
}

func newTracingDomain(store tracing.TraceStore) *TracingDomain {
	return &TracingDomain{store: store}
}

func (d *TracingDomain) Get(_ context.Context, req messages.TraceGetMsg) (*messages.TraceGetResp, error) {
	if d.store == nil {
		return &messages.TraceGetResp{Spans: json.RawMessage("[]")}, nil
	}
	spans, err := d.store.GetTrace(req.TraceID)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(spans)
	return &messages.TraceGetResp{Spans: data}, nil
}

func (d *TracingDomain) List(_ context.Context, req messages.TraceListMsg) (*messages.TraceListResp, error) {
	if d.store == nil {
		return &messages.TraceListResp{Traces: json.RawMessage("[]")}, nil
	}
	query := tracing.TraceQuery{Source: req.Source, Status: req.Status, Limit: req.Limit}
	if req.MinDuration > 0 {
		query.MinDuration = time.Duration(req.MinDuration) * time.Millisecond
	}
	traces, err := d.store.ListTraces(query)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(traces)
	return &messages.TraceListResp{Traces: data}, nil
}
