// Package tracing installs a persistent trace store into a Kit's tracer
// and registers the trace.get / trace.list bus commands.
//
// The core Tracer + in-memory ring buffer stay in internal/tracing so span
// creation is always available. This module is what promotes the in-memory
// tracer to durable storage and exposes the query surface.
package tracing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

// Config configures the tracing module.
type Config struct {
	// Store is the durable span store to attach to the Kit's tracer.
	// Required — the module's Init returns nil without it.
	Store TraceStore
}

// Module wires a durable trace store + trace.get / trace.list commands.
type Module struct {
	cfg   Config
	store TraceStore
}

// New builds a tracing module.
func New(cfg Config) *Module { return &Module{cfg: cfg} }

// Name reports the module identifier.
func (m *Module) Name() string { return "tracing" }

// Status reports maturity.
func (m *Module) Status() brainkit.ModuleStatus { return brainkit.ModuleStatusBeta }

// Init installs the module's store into the Kit's tracer and registers
// trace.get / trace.list bus commands.
func (m *Module) Init(k *brainkit.Kit) error {
	if m.cfg.Store == nil {
		return nil
	}
	m.store = m.cfg.Store
	k.SetTraceStore(m.store)

	k.RegisterCommand(brainkit.Command(m.handleGet))
	k.RegisterCommand(brainkit.Command(m.handleList))
	return nil
}

// Close closes the trace store if it implements io.Closer.
func (m *Module) Close() error {
	if c, ok := m.store.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

func (m *Module) handleGet(_ context.Context, req sdk.TraceGetMsg) (*sdk.TraceGetResp, error) {
	if m.store == nil {
		return &sdk.TraceGetResp{Spans: json.RawMessage("[]")}, nil
	}
	spans, err := m.store.GetTrace(req.TraceID)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(spans)
	return &sdk.TraceGetResp{Spans: data}, nil
}

func (m *Module) handleList(_ context.Context, req sdk.TraceListMsg) (*sdk.TraceListResp, error) {
	if m.store == nil {
		return &sdk.TraceListResp{Traces: json.RawMessage("[]")}, nil
	}
	query := TraceQuery{Source: req.Source, Status: req.Status, Limit: req.Limit}
	if req.MinDuration > 0 {
		query.MinDuration = time.Duration(req.MinDuration) * time.Millisecond
	}
	traces, err := m.store.ListTraces(query)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(traces)
	return &sdk.TraceListResp{Traces: data}, nil
}
