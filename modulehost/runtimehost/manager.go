// Package runtimehost owns JS-runtime persistence and propagation orchestration
// that sits outside the concrete engine.
package runtimehost

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/sdk/systemmsg"
)

const defaultPropagationCloseTimeout = 10 * time.Second

// PropagationHandle is the lifecycle-aware close contract for runtime-host
// deployment propagation subscriptions.
type PropagationHandle interface {
	Close(context.Context) error
}

// SubscribeFanOut subscribes every runtime replica to a topic.
type SubscribeFanOut func(context.Context, string, func(sdk.Message)) (PropagationHandle, error)

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

	mu                       sync.Mutex
	propagationSubscriptions []propagationSubscription
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultPropagationCloseTimeout)
	defer cancel()
	return m.CloseContext(ctx)
}

// CloseContext cancels and waits for runtime-host propagation subscriptions.
// Timed-out subscriptions are retained so a later close can retry cleanup.
func (m *Manager) CloseContext(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	subs := m.takePropagationSubscriptions()
	failed, err := closePropagationSubscriptions(ctx, subs)
	if len(failed) > 0 {
		m.appendPropagationSubscriptions(failed)
	}
	return err
}

func (m *Manager) DebugSnapshot() DebugSnapshot {
	if m == nil {
		return DebugSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return DebugSnapshot{PropagationSubscriptions: len(m.propagationSubscriptions)}
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

	deploySubCtx, deployCancel := context.WithCancel(context.Background())
	deployHandle, err := m.subscribeFanOut(deploySubCtx, systemmsg.TopicKitDeployed, func(msg sdk.Message) {
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
		if _, err := m.host.Deploy(deploySubCtx, dep.Source, dep.Code, opts...); err != nil {
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
		deployCancel()
		m.logger.Warn("propagation: subscribe deploy failed", slog.String("error", err.Error()))
		return
	}
	deploySub := propagationSubscription{handle: deployHandle, cancel: deployCancel}

	teardownSubCtx, teardownCancel := context.WithCancel(context.Background())
	teardownHandle, err := m.subscribeFanOut(teardownSubCtx, systemmsg.TopicKitTeardowned, func(msg sdk.Message) {
		var evt systemmsg.KitTeardownedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			return
		}
		if evt.RuntimeID == cfg.RuntimeID {
			return
		}
		if _, err := m.host.Teardown(teardownSubCtx, evt.Source); err != nil {
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
		teardownCancel()
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultPropagationCloseTimeout)
		if closeErr := deploySub.Close(closeCtx); closeErr != nil {
			m.logger.Warn("propagation: close deploy subscription failed", slog.String("error", closeErr.Error()))
		}
		cancel()
		m.logger.Warn("propagation: subscribe teardown failed", slog.String("error", err.Error()))
		return
	}
	m.replacePropagationSubscriptions([]propagationSubscription{
		deploySub,
		{handle: teardownHandle, cancel: teardownCancel},
	})
}

func (m *Manager) replacePropagationSubscriptions(subs []propagationSubscription) {
	m.mu.Lock()
	old := append([]propagationSubscription{}, m.propagationSubscriptions...)
	m.propagationSubscriptions = subs
	m.mu.Unlock()

	if len(old) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultPropagationCloseTimeout)
	failed, err := closePropagationSubscriptions(ctx, old)
	cancel()
	if err != nil {
		m.logger.Warn("propagation: close replaced subscriptions failed", slog.String("error", err.Error()))
	}
	if len(failed) > 0 {
		m.appendPropagationSubscriptions(failed)
	}
}

func (m *Manager) takePropagationSubscriptions() []propagationSubscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	subs := append([]propagationSubscription{}, m.propagationSubscriptions...)
	m.propagationSubscriptions = nil
	return subs
}

func (m *Manager) appendPropagationSubscriptions(subs []propagationSubscription) {
	m.mu.Lock()
	m.propagationSubscriptions = append(m.propagationSubscriptions, subs...)
	m.mu.Unlock()
}

func (m *Manager) runtimeReady() bool {
	return m != nil && m.host != nil && m.host.HasJSRuntime()
}

type propagationSubscription struct {
	handle PropagationHandle
	cancel context.CancelFunc
}

func (s propagationSubscription) Close(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.handle == nil {
		return nil
	}
	return s.handle.Close(ctx)
}

func closePropagationSubscriptions(ctx context.Context, subs []propagationSubscription) ([]propagationSubscription, error) {
	var err error
	var failed []propagationSubscription
	for _, sub := range subs {
		if closeErr := sub.Close(ctx); closeErr != nil {
			err = errors.Join(err, closeErr)
			failed = append(failed, sub)
		}
	}
	return failed, err
}
