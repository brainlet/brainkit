package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// redeployPersistedDeployments loads and re-deploys all persisted .ts deployments.
func (k *Kernel) redeployPersistedDeployments() {
	deployments, err := k.config.Store.LoadDeployments()
	if err != nil {
		types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
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
	k.deploymentMgr.SetDeployOrderSeed(maxOrder)

	for _, d := range deployments {
		var opts []types.DeployOption
		opts = append(opts, types.WithRestoring()) // don't re-persist what was just loaded
		if d.Role != "" {
			opts = append(opts, types.WithRole(d.Role))
		}
		if d.PackageName != "" {
			opts = append(opts, types.WithPackageName(d.PackageName))
		}
		if _, err := k.Deploy(context.Background(), d.Source, d.Code, opts...); err != nil {
			types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.DeployError{
				Source: d.Source, Phase: "redeploy", Cause: err,
			}, types.ErrorContext{Operation: "RedeployPersisted", Component: "kernel", Source: d.Source})
		}
	}

	k.logger.Info("redeployed persisted deployments", slog.Int("count", len(deployments)))
}

// upgradeMastraStorage resolves the configured storage backend and upgrades
// the Mastra store holder from InMemoryStore to the real backend.
// Tries "default" first (convention), falls back to first available.
func (k *Kernel) upgradeMastraStorage() {
	raw, err := k.callJS(context.Background(), "__brainkit.storage.upgrade", nil)
	if err != nil {
		types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "UpgradeMastraStorage", Cause: err,
		}, types.ErrorContext{Operation: "UpgradeMastraStorage", Component: "kernel"})
		return
	}
	var parsed struct {
		Upgraded bool   `json:"upgraded"`
		Storage  string `json:"storage"`
	}
	if json.Unmarshal(raw, &parsed) == nil && parsed.Upgraded {
		k.logger.Info("Mastra storage upgraded", slog.String("backend", parsed.Storage))
	}
}

// restartActiveWorkflows calls restartAllActiveWorkflowRuns() on all registered
// workflows. Picks up runs with status "running" or "waiting" from storage,
// reconnects via createRun({runId}), and calls restart() to re-enter from snapshot.
// Called automatically during NewKernel after .ts re-deployment.
func (k *Kernel) restartActiveWorkflows() {
	raw, err := k.callJS(context.Background(), "__brainkit.storage.restartWorkflows", nil)
	if err != nil {
		types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
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
	if json.Unmarshal(raw, &parsed) == nil {
		for _, wfErr := range parsed.Errors {
			types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
				Operation: "RestartWorkflow", Source: wfErr.Workflow, Cause: fmt.Errorf("%s", wfErr.Error),
			}, types.ErrorContext{Operation: "RestartWorkflow", Component: "workflow", Source: wfErr.Workflow})
		}
		if parsed.Restarted > 0 {
			k.logger.Info("restarted active workflows", slog.Int("definitions", parsed.Restarted))
		}
	}
}

// RestartActiveWorkflows is the public Go API for manually triggering workflow recovery.
func (k *Kernel) RestartActiveWorkflows(ctx context.Context) error {
	k.restartActiveWorkflows()
	return nil
}

// subscribeToDeploymentPropagation listens for deploy/teardown events from other replicas.
// Uses the fan-out subscriber so ALL replicas receive these events.
// When a deploy event from a different RuntimeID arrives, loads the deployment from
// the shared KitStore and deploys locally.
func (k *Kernel) subscribeToDeploymentPropagation() {
	// Listen for deploy events
	_, _ = k.remote.SubscribeRawFanOut(context.Background(), "kit.deployed", func(msg sdk.Message) {
		var evt sdk.KitDeployedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			return
		}
		// Skip events from self
		if evt.RuntimeID == k.config.RuntimeID {
			return
		}
		// Load from shared store and deploy locally
		dep, err := k.config.Store.LoadDeployment(evt.Source)
		if err != nil {
			k.logger.Warn("propagation: load deployment failed",
				slog.String("source", evt.Source),
				slog.String("error", err.Error()))
			return
		}
		if _, err := k.Deploy(context.Background(), dep.Source, dep.Code, types.WithRestoring()); err != nil {
			k.logger.Warn("propagation: deploy failed",
				slog.String("source", evt.Source),
				slog.String("error", err.Error()))
		} else {
			k.logger.Info("propagation: deployed from replica",
				slog.String("source", evt.Source),
				slog.String("runtimeID", evt.RuntimeID))
		}
	})

	// Listen for teardown events
	_, _ = k.remote.SubscribeRawFanOut(context.Background(), "kit.teardown.done", func(msg sdk.Message) {
		var evt sdk.KitTeardownedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			return
		}
		if evt.RuntimeID == k.config.RuntimeID {
			return
		}
		if _, err := k.Teardown(context.Background(), evt.Source); err != nil {
			k.logger.Warn("propagation: teardown failed",
				slog.String("source", evt.Source),
				slog.String("error", err.Error()))
		} else {
			k.logger.Info("propagation: torn down from replica",
				slog.String("source", evt.Source),
				slog.String("runtimeID", evt.RuntimeID))
		}
	})
}
