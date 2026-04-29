package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/types"
	agentsmod "github.com/brainlet/brainkit/modules/agents"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/sdk"
)

// RuntimeConfig returns a copy of the kernel config for optional runtime
// attachments.
func (k *Kernel) RuntimeConfig() types.KernelConfig { return k.config }

// SetRuntimeConfigJSRuntime marks whether the optional JS runtime is active.
func (k *Kernel) SetRuntimeConfigJSRuntime(active bool) { k.config.JSRuntime = active }

// ToolsDomain exposes the local tool domain to optional runtime attachments.
func (k *Kernel) ToolsDomain() *toolsmod.Domain { return k.toolsDomain }

// AgentsDomain exposes the local agent registry to optional runtime attachments.
func (k *Kernel) AgentsDomain() *agentsmod.Domain { return k.agentsDomain }

// SetToolEvaluator installs the JS evaluator used by JS-registered tools.
func (k *Kernel) SetToolEvaluator(eval JSEvaluator) {
	if k.toolsDomain != nil {
		k.toolsDomain.SetEvaluator(eval)
	}
}

// ValidateEvent validates a publish payload against the event catalog.
func (k *Kernel) ValidateEvent(topic string, payload json.RawMessage) error {
	return k.events.Validate(topic, payload)
}

// PublishEvent publishes a fire-and-forget event on the kernel bus.
func (k *Kernel) PublishEvent(ctx context.Context, topic string, payload json.RawMessage) error {
	return k.publish(ctx, topic, payload)
}

// SubscribeEvent subscribes to a topic on the kernel bus.
func (k *Kernel) SubscribeEvent(topic string, handler func(sdk.Message)) (func(), error) {
	return k.subscribe(topic, handler)
}

// InvokeCommand invokes a mounted command directly from the JS bridge.
func (k *Kernel) InvokeCommand(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error) {
	spec, ok := k.catalog.Lookup(topic)
	if !ok || spec.invokeKernel == nil {
		return nil, fmt.Errorf("unknown topic: %s", topic)
	}
	return spec.invokeKernel(ctx, k, payload)
}

// EnterHandler marks a bridge-dispatched handler as active.
func (k *Kernel) EnterHandler() bool { return k.enterHandler() }

// ExitHandler marks a bridge-dispatched handler as complete.
func (k *Kernel) ExitHandler() { k.exitHandler() }

// HandleHandlerFailure applies retry/dead-letter/error-response handling.
func (k *Kernel) HandleHandlerFailure(msg sdk.Message, topic string, err error) {
	k.handleHandlerFailure(msg, topic, err)
}

// EmitLog writes a runtime log entry through the configured log handler.
func (k *Kernel) EmitLog(source, level, message string) { k.emitLog(source, level, message) }

// IsClosed reports whether the kernel has entered close.
func (k *Kernel) IsClosed() bool {
	k.mu.Lock()
	closed := k.closed
	k.mu.Unlock()
	return closed
}

// IncrementPumpCycles increments the runtime job-pump metric.
func (k *Kernel) IncrementPumpCycles() { k.pumpCycles.Add(1) }

// StopStreamHeartbeat stops a stream reply heartbeat.
func (k *Kernel) StopStreamHeartbeat(replyTo string) {
	if k.streamTracker != nil {
		k.streamTracker.StopHeartbeat(replyTo)
	}
}

// StartStreamHeartbeat starts a stream reply heartbeat.
func (k *Kernel) StartStreamHeartbeat(replyTo, correlationID string) {
	if k.streamTracker != nil {
		k.streamTracker.StartHeartbeat(replyTo, correlationID)
	}
}

// ScheduleHandler returns the active schedule handler, if mounted.
func (k *Kernel) ScheduleHandler() types.ScheduleHandler { return k.scheduleHandler }

// ExistingStorageBridgeNames snapshots active storage bridge names.
func (k *Kernel) ExistingStorageBridgeNames() map[string]bool {
	if k.storageHost == nil {
		return map[string]bool{}
	}
	return k.storageHost.ExistingNames()
}

// CloseStorageBridgesExcept closes storage bridges not present in keep.
func (k *Kernel) CloseStorageBridgesExcept(keep map[string]bool) {
	if k.storageHost != nil {
		k.storageHost.CloseExcept(keep)
	}
}

// InitStorageBridges starts configured storage bridges for the runtime.
func (k *Kernel) InitStorageBridges(cfg types.KernelConfig) (map[string]string, error) {
	return k.initStorages(cfg)
}

// RegisterConfiguredStorages registers configured storages in the provider registry.
func (k *Kernel) RegisterConfiguredStorages(cfg types.KernelConfig, bridgeURLs map[string]string) {
	k.registerStorages(cfg, bridgeURLs)
}

// RegisterConfiguredVectors registers configured vectors in the provider registry.
func (k *Kernel) RegisterConfiguredVectors(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	return k.registerVectors(cfg, bridgeURLs)
}

// RestoreJSRuntimeState restores persisted JS deployments and workflow state.
func (k *Kernel) RestoreJSRuntimeState(cfg types.KernelConfig) { k.initPersistence(cfg) }

// RuntimeShutdownContext returns the kernel shutdown context.
func (k *Kernel) RuntimeShutdownContext() context.Context { return k.shutdownCtx }
