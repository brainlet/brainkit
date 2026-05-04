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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/internal/closejob"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/tracing/tracingmsg"
)

// Config configures the tracing module.
type Config struct {
	// Store is the durable span store to attach to the Kit's tracer.
	// Required — the module's Mount returns nil without it.
	Store TraceStore
}

// Module wires a durable trace store + trace.get / trace.list commands.
type Module struct {
	closeMu sync.Mutex
	mu      sync.RWMutex
	cfg     Config

	store TraceStore

	traceStoreLease       bkmodule.Handle
	traceStoreLeaseActive atomic.Bool
	closing               atomic.Bool
	storeClosing          atomic.Bool
	storeCloseJob         closejob.Job
}

type contextCloseableStore interface {
	CloseContext(context.Context) error
}

// New builds a tracing module.
func New(cfg Config) *Module { return &Module{cfg: cfg} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "tracing" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	leaseTraceStore, err := bkmodule.RequireCapability[bkmodule.LeaseFunc[TraceStore]](host, bkmodule.CapabilityTraceStoreLease)
	if err != nil {
		return fmt.Errorf("tracing: %w", err)
	}
	if m.cfg.Store == nil {
		return nil
	}
	m.mu.Lock()
	m.store = m.cfg.Store
	m.mu.Unlock()
	host.Scope().Defer(func(closeCtx context.Context) error { return m.CloseContext(closeCtx) })
	lease, err := leaseTraceStore(ctx, m.store)
	if err != nil {
		return fmt.Errorf("tracing: %w", err)
	}
	m.mu.Lock()
	m.traceStoreLease = lease
	m.traceStoreLeaseActive.Store(true)
	m.mu.Unlock()
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindStore, "tracing.store", "Attached persistent trace store."))
	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "tracing", func() any {
			return m.DebugSnapshot()
		})
		if err != nil {
			return fmt.Errorf("tracing: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}
	if _, err := host.Commands().Handle(bkmodule.Command(m.handleGet)); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(m.handleList)); err != nil {
		return err
	}
	return nil
}

// Close closes the trace store if it implements io.Closer.
func (m *Module) Close() error {
	return m.CloseContext(context.Background())
}

// CloseContext closes the trace store lease and owned store under caller-owned
// lifecycle cancellation.
func (m *Module) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.closeMu.Lock()
	defer m.closeMu.Unlock()
	m.closing.Store(true)
	defer m.closing.Store(false)
	var err error

	m.mu.RLock()
	lease := m.traceStoreLease
	store := m.store
	m.mu.RUnlock()

	if lease != nil {
		if leaseErr := lease.Close(ctx); leaseErr != nil {
			return leaseErr
		}
		m.mu.Lock()
		m.traceStoreLease = nil
		m.traceStoreLeaseActive.Store(false)
		store = m.store
		m.mu.Unlock()
	} else {
		m.traceStoreLeaseActive.Store(false)
	}
	if store == nil {
		return nil
	}
	done := m.storeCloseJob.Start(func() error {
		m.storeClosing.Store(true)
		defer m.storeClosing.Store(false)
		closeErr := closeTraceStore(ctx, store)
		if closeErr == nil {
			m.mu.Lock()
			if m.store == store {
				m.store = nil
			}
			m.mu.Unlock()
		}
		return closeErr
	})
	if closeErr := m.storeCloseJob.Wait(ctx, done); closeErr != nil {
		err = errors.Join(err, closeErr)
	}
	return err
}

func closeTraceStore(ctx context.Context, store TraceStore) error {
	if c, ok := store.(contextCloseableStore); ok {
		return c.CloseContext(ctx)
	}
	if c, ok := store.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

func (m *Module) handleGet(_ context.Context, req tracingmsg.TraceGetMsg) (*tracingmsg.TraceGetResp, error) {
	store := m.currentStore()
	if store == nil {
		return &tracingmsg.TraceGetResp{Spans: json.RawMessage("[]")}, nil
	}
	spans, err := store.GetTrace(req.TraceID)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(spans)
	return &tracingmsg.TraceGetResp{Spans: data}, nil
}

func (m *Module) handleList(_ context.Context, req tracingmsg.TraceListMsg) (*tracingmsg.TraceListResp, error) {
	store := m.currentStore()
	if store == nil {
		return &tracingmsg.TraceListResp{Traces: json.RawMessage("[]")}, nil
	}
	query := TraceQuery{Source: req.Source, Status: req.Status, Limit: req.Limit}
	if req.MinDuration > 0 {
		query.MinDuration = time.Duration(req.MinDuration) * time.Millisecond
	}
	traces, err := store.ListTraces(query)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(traces)
	return &tracingmsg.TraceListResp{Traces: data}, nil
}

func (m *Module) currentStore() TraceStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store
}
