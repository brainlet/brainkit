// Package runtimehost owns JS-runtime persistence and propagation orchestration
// that sits outside the concrete engine.
package runtimehost

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"sync"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/sdk/systemmsg"
)

// SubscribeFanOut subscribes every runtime replica to a topic.
type SubscribeFanOut func(context.Context, string, func(sdk.Message)) (func(), error)

// Host is the narrow runtime surface needed by persistence and propagation
// orchestration.
type Host interface {
	runtimecap.SourceDeployer
	HasJSRuntime() bool
	SetDeployOrderSeed(seed int32)
}

// Manager owns persistence restoration and deployment propagation for an
// attached JS runtime.
type Manager struct {
	host            Host
	logger          *slog.Logger
	subscribeFanOut SubscribeFanOut

	mu                sync.Mutex
	propagationUnsubs []func()
}

// DebugSnapshot reports runtime-host-owned propagation state for lifecycle
// inspection and teardown tests.
type DebugSnapshot struct {
	PropagationSubscriptions int
}

// New creates a runtime host manager.
func New(host Host, logger *slog.Logger, subscribeFanOut SubscribeFanOut) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		host:            host,
		logger:          logger,
		subscribeFanOut: subscribeFanOut,
	}
}

// InitPersistence runs runtime persistence hooks for the active configuration.
func (m *Manager) InitPersistence(cfg types.KernelConfig) {
	if cfg.Store == nil || !cfg.JSRuntime || !m.runtimeReady() {
		return
	}

	m.RedeployPersistedDeployments(cfg)
	m.SubscribeToDeploymentPropagation(cfg)

	// Schedule restoration is the schedules module's responsibility; it runs
	// on its own mount via the attached Store.
}

// Close releases runtime-host subscriptions. Safe to call more than once.
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	unsubs := m.takePropagationUnsubs()
	for _, unsub := range unsubs {
		unsub()
	}
	return nil
}

func (m *Manager) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return DebugSnapshot{PropagationSubscriptions: len(m.propagationUnsubs)}
}

// RedeployPersistedDeployments loads and re-deploys all persisted .ts deployments.
func (m *Manager) RedeployPersistedDeployments(cfg types.KernelConfig) {
	if cfg.Store == nil || !m.runtimeReady() {
		return
	}
	deployments, err := cfg.Store.LoadDeployments()
	if err != nil {
		types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "LoadDeployments", Cause: err,
		}, types.ErrorContext{Operation: "LoadDeployments", Component: "kernel"})
		return
	}
	if len(deployments) == 0 {
		return
	}

	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].Order < deployments[j].Order
	})

	maxOrder := int32(deployments[len(deployments)-1].Order)
	m.host.SetDeployOrderSeed(maxOrder)

	for _, d := range deployments {
		opts := []types.DeployOption{types.WithRestoring()}
		if d.PackageName != "" {
			opts = append(opts, types.WithPackageName(d.PackageName))
		}
		if d.EffectiveArtifactKind() == types.DeployArtifactNormalizedJS {
			opts = append(opts, types.WithNormalizedJS())
		}
		if _, err := m.host.Deploy(context.Background(), d.Source, d.Code, opts...); err != nil {
			types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.DeployError{
				Source: d.Source, Phase: "redeploy", Cause: err,
			}, types.ErrorContext{Operation: "RedeployPersisted", Component: "kernel", Source: d.Source})
		}
	}

	m.logger.Info("redeployed persisted deployments", slog.Int("count", len(deployments)))
}

// SubscribeToDeploymentPropagation listens for deploy/teardown events from
// other replicas and mirrors them locally from the shared store.
func (m *Manager) SubscribeToDeploymentPropagation(cfg types.KernelConfig) {
	if cfg.Store == nil || !m.runtimeReady() || m.subscribeFanOut == nil {
		return
	}

	deployUnsub, err := m.subscribeFanOut(context.Background(), systemmsg.TopicKitDeployed, func(msg sdk.Message) {
		var evt systemmsg.KitDeployedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			return
		}
		if evt.RuntimeID == cfg.RuntimeID {
			return
		}
		dep, err := cfg.Store.LoadDeployment(evt.Source)
		if err != nil {
			m.logger.Warn("propagation: load deployment failed",
				slog.String("source", evt.Source),
				slog.String("error", err.Error()))
			return
		}
		opts := []types.DeployOption{types.WithRestoring()}
		if dep.PackageName != "" {
			opts = append(opts, types.WithPackageName(dep.PackageName))
		}
		if dep.EffectiveArtifactKind() == types.DeployArtifactNormalizedJS {
			opts = append(opts, types.WithNormalizedJS())
		}
		if _, err := m.host.Deploy(context.Background(), dep.Source, dep.Code, opts...); err != nil {
			m.logger.Warn("propagation: deploy failed",
				slog.String("source", evt.Source),
				slog.String("error", err.Error()))
		} else {
			m.logger.Info("propagation: deployed from replica",
				slog.String("source", evt.Source),
				slog.String("runtimeID", evt.RuntimeID))
		}
	})
	if err != nil {
		m.logger.Warn("propagation: subscribe deploy failed", slog.String("error", err.Error()))
		return
	}

	teardownUnsub, err := m.subscribeFanOut(context.Background(), systemmsg.TopicKitTeardowned, func(msg sdk.Message) {
		var evt systemmsg.KitTeardownedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			return
		}
		if evt.RuntimeID == cfg.RuntimeID {
			return
		}
		if _, err := m.host.Teardown(context.Background(), evt.Source); err != nil {
			m.logger.Warn("propagation: teardown failed",
				slog.String("source", evt.Source),
				slog.String("error", err.Error()))
		} else {
			m.logger.Info("propagation: torn down from replica",
				slog.String("source", evt.Source),
				slog.String("runtimeID", evt.RuntimeID))
		}
	})
	if err != nil {
		deployUnsub()
		m.logger.Warn("propagation: subscribe teardown failed", slog.String("error", err.Error()))
		return
	}
	m.replacePropagationUnsubs([]func(){deployUnsub, teardownUnsub})
}

func (m *Manager) replacePropagationUnsubs(unsubs []func()) {
	m.mu.Lock()
	old := append([]func(){}, m.propagationUnsubs...)
	m.propagationUnsubs = unsubs
	m.mu.Unlock()
	for _, unsub := range old {
		unsub()
	}
}

func (m *Manager) takePropagationUnsubs() []func() {
	m.mu.Lock()
	defer m.mu.Unlock()
	unsubs := append([]func(){}, m.propagationUnsubs...)
	m.propagationUnsubs = nil
	return unsubs
}

func (m *Manager) runtimeReady() bool {
	return m != nil && m.host != nil && m.host.HasJSRuntime()
}
