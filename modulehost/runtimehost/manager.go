// Package runtimehost owns JS-runtime persistence and propagation orchestration
// that sits outside the concrete engine.
package runtimehost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

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
	runtimecap.Deployer
	HasJSRuntime() bool
	SetDeployOrderSeed(seed int32)
	CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error)
}

// Manager owns persistence restoration, deployment propagation, and workflow
// restart orchestration for an attached JS runtime.
type Manager struct {
	host            Host
	logger          *slog.Logger
	subscribeFanOut SubscribeFanOut
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

	if len(cfg.Storages) > 0 {
		m.RestartActiveWorkflows(cfg)
	}
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
		if _, err := m.host.Deploy(context.Background(), d.Source, d.Code, opts...); err != nil {
			types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.DeployError{
				Source: d.Source, Phase: "redeploy", Cause: err,
			}, types.ErrorContext{Operation: "RedeployPersisted", Component: "kernel", Source: d.Source})
		}
	}

	m.logger.Info("redeployed persisted deployments", slog.Int("count", len(deployments)))
}

// RestartActiveWorkflows calls restartAllActiveWorkflowRuns() on all registered
// workflows and reports non-fatal recovery errors through the configured handler.
func (m *Manager) RestartActiveWorkflows(cfg types.KernelConfig) {
	if !m.runtimeReady() {
		return
	}
	raw, err := m.host.CallJS(context.Background(), "__brainkit.storage.restartWorkflows", nil)
	if err != nil {
		types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "RestartActiveWorkflows", Cause: err,
		}, types.ErrorContext{Operation: "RestartActiveWorkflows", Component: "kernel"})
		return
	}
	var parsed struct {
		Restarted int `json:"restarted"`
		Errors    []struct {
			Workflow string `json:"workflow"`
			Error    string `json:"error"`
		} `json:"errors"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return
	}
	for _, wfErr := range parsed.Errors {
		types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "RestartWorkflow", Source: wfErr.Workflow, Cause: fmt.Errorf("%s", wfErr.Error),
		}, types.ErrorContext{Operation: "RestartWorkflow", Component: "workflow", Source: wfErr.Workflow})
	}
	if parsed.Restarted > 0 {
		m.logger.Info("restarted active workflows", slog.Int("definitions", parsed.Restarted))
	}
}

// SubscribeToDeploymentPropagation listens for deploy/teardown events from
// other replicas and mirrors them locally from the shared store.
func (m *Manager) SubscribeToDeploymentPropagation(cfg types.KernelConfig) {
	if cfg.Store == nil || !m.runtimeReady() || m.subscribeFanOut == nil {
		return
	}

	_, _ = m.subscribeFanOut(context.Background(), systemmsg.TopicKitDeployed, func(msg sdk.Message) {
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
		if _, err := m.host.Deploy(context.Background(), dep.Source, dep.Code, types.WithRestoring()); err != nil {
			m.logger.Warn("propagation: deploy failed",
				slog.String("source", evt.Source),
				slog.String("error", err.Error()))
		} else {
			m.logger.Info("propagation: deployed from replica",
				slog.String("source", evt.Source),
				slog.String("runtimeID", evt.RuntimeID))
		}
	})

	_, _ = m.subscribeFanOut(context.Background(), systemmsg.TopicKitTeardowned, func(msg sdk.Message) {
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
}

func (m *Manager) runtimeReady() bool {
	return m != nil && m.host != nil && m.host.HasJSRuntime()
}
