// Package tracing installs a persistent trace store into a Kit's tracer
// and registers the trace.get / trace.list bus commands.
//
// The core Tracer + in-memory ring buffer stay in internal/tracing so span
// creation is always available. This module is what promotes the in-memory
// tracer to durable storage and exposes the query surface.
package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/tracing/tracingmsg"

	_ "modernc.org/sqlite"
)

// Config configures the tracing module.
type Config struct {
	// Store is the durable span store to attach to the Kit's tracer.
	// Required — the module's Mount returns nil without it.
	Store TraceStore
}

// Module wires a durable trace store + trace.get / trace.list commands.
type Module struct {
	cfg   Config
	store TraceStore
}

// New builds a tracing module.
func New(cfg Config) *Module { return &Module{cfg: cfg} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "tracing" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	setTraceStore, err := bkmodule.RequireCapability[func(TraceStore)](host, bkmodule.CapabilitySetTraceStore)
	if err != nil {
		return fmt.Errorf("tracing: %w", err)
	}
	host.Scope().Defer(func(context.Context) error {
		setTraceStore(nil)
		return m.Close()
	})
	if !m.attach(traceCoreFunc(setTraceStore)) {
		return nil
	}
	if _, err := host.Commands().Handle(bkmodule.Command(m.handleGet)); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(m.handleList)); err != nil {
		return err
	}
	return nil
}

type traceCore interface {
	SetTraceStore(TraceStore)
}

type traceCoreFunc func(TraceStore)

func (f traceCoreFunc) SetTraceStore(store TraceStore) { f(store) }

func (m *Module) attach(core traceCore) bool {
	if m.cfg.Store == nil {
		return false
	}
	m.store = m.cfg.Store
	core.SetTraceStore(m.store)
	return true
}

// Close closes the trace store if it implements io.Closer.
func (m *Module) Close() error {
	if m.store == nil {
		return nil
	}
	defer func() { m.store = nil }()
	if c, ok := m.store.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

func (m *Module) handleGet(_ context.Context, req tracingmsg.TraceGetMsg) (*tracingmsg.TraceGetResp, error) {
	if m.store == nil {
		return &tracingmsg.TraceGetResp{Spans: json.RawMessage("[]")}, nil
	}
	spans, err := m.store.GetTrace(req.TraceID)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(spans)
	return &tracingmsg.TraceGetResp{Spans: data}, nil
}

func (m *Module) handleList(_ context.Context, req tracingmsg.TraceListMsg) (*tracingmsg.TraceListResp, error) {
	if m.store == nil {
		return &tracingmsg.TraceListResp{Traces: json.RawMessage("[]")}, nil
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
	return &tracingmsg.TraceListResp{Traces: data}, nil
}

// YAML is the config shape decoded by the registry factory. Empty
// Path falls back to `<FSRoot>/tracing.db`. Zero Retention disables
// cleanup.
type YAML struct {
	Path      string        `yaml:"path"`
	Retention time.Duration `yaml:"retention"`
}

// Factory is the registered ModuleFactory for tracing.
type Factory struct{}

// Build opens the SQLite-backed trace store and returns the module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	path := y.Path
	if path == "" {
		path = filepath.Join(ctx.FSRoot, "tracing.db")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("tracing: open db %q: %w", path, err)
	}
	var opts []SQLiteTraceStoreOption
	if y.Retention > 0 {
		opts = append(opts, WithRetention(y.Retention))
	}
	store, err := NewSQLiteTraceStore(db, opts...)
	if err != nil {
		return nil, fmt.Errorf("tracing: init store %q: %w", path, err)
	}
	return New(Config{Store: store}), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "tracing",
		Status:  bkmodule.StatusBeta,
		Summary: "Persistent span store with trace.get / trace.list.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[tracingmsg.TraceGetMsg, tracingmsg.TraceGetResp](),
			bkmodule.CommandMessage[tracingmsg.TraceListMsg, tracingmsg.TraceListResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[func(TraceStore)](bkmodule.CapabilitySetTraceStore),
		},
	}
}

func init() { bkmodule.Register("tracing", Factory{}) }
