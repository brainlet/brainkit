package jsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/internal/jsbridge"
	"github.com/brainlet/brainkit/internal/types"
	runtimecap "github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/modulehost/providerhost"
)

// Runtime owns the optional embedded JavaScript runtime for a Kernel.
type Runtime struct {
	core           runtimecap.CoreHost
	registry       runtimecap.RegistryHost
	toolAgents     runtimecap.ToolAgentHost
	storage        runtimecap.StorageHost
	bus            runtimecap.BusHost
	handlers       runtimecap.HandlerHost
	schedules      runtimecap.ScheduleHost
	cfg            types.KernelConfig
	bridge         *jsbridge.Bridge
	agents         *agentembed.Sandbox
	deploymentMgr  *DeploymentManager
	sourcePreparer SourcePreparer

	mu         sync.Mutex
	closeMu    sync.Mutex
	closing    bool
	closed     bool
	closeMode  runtimeCloseMode
	cleanup    func(context.Context, runtimeCloseMode) error
	bridgeSubs map[string]func()
}

// SourcePreparer converts raw source into JavaScript before runtime deploy.
// It is optional so artifact-only runtime profiles can avoid linking a
// TypeScript compiler.
type SourcePreparer func(source, code string) (string, error)

type enableConfig struct {
	sourcePreparer SourcePreparer
}

// EnableOption configures runtime activation.
type EnableOption func(*enableConfig)

// WithSourcePreparer installs a raw-source preparation hook used for `.ts`
// deployments before JS evaluation.
func WithSourcePreparer(preparer SourcePreparer) EnableOption {
	return func(cfg *enableConfig) { cfg.sourcePreparer = preparer }
}

type runtimeCloseMode int

const (
	runtimeCloseModeUnmount runtimeCloseMode = iota
	runtimeCloseModeShutdown
)

// Enable starts the embedded JavaScript runtime for a light Kernel. It is
// idempotent and can be called during construction or from a hot-mounted
// jsruntime module.
func Enable(ctx context.Context, host runtimecap.EnableHost, opts ...EnableOption) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if host.HasJSRuntime() {
		return nil
	}
	var enableCfg enableConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&enableCfg)
		}
	}

	cfg := host.RuntimeConfig()
	cfg.JSRuntime = true

	providers := make(map[string]agentembed.ProviderConfig)
	for name, reg := range cfg.AIProviders {
		pc := providerhost.ExtractProviderCredentials(reg)
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}

	var audioSink jsbridge.AudioSink
	if cfg.AudioSink != nil {
		s, ok := cfg.AudioSink.(jsbridge.AudioSink)
		if !ok {
			return fmt.Errorf("brainkit: AudioSink does not implement jsbridge.AudioSink")
		}
		audioSink = s
	}

	agentSandbox, err := agentembed.NewSandbox(agentembed.SandboxConfig{
		Providers:    providers,
		EnvVars:      cfg.EnvVars,
		MaxStackSize: cfg.MaxStackSize,
		CWD:          cfg.FSRoot,
		AudioSink:    audioSink,
		FetchSpanHook: func(method, url string) func(int, error) {
			tracer := host.Tracer()
			if tracer == nil {
				return nil
			}
			span := tracer.StartSpan("fetch", context.Background())
			span.SetAttribute("method", method)
			span.SetAttribute("url", url)
			return func(statusCode int, err error) {
				if statusCode > 0 {
					span.SetAttribute("status", strconv.Itoa(statusCode))
				}
				span.End(err)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("brainkit: create runtime: %w", err)
	}

	existingStorages := host.ExistingStorageBridgeNames()
	if host.HasJSRuntime() {
		agentSandbox.Close()
		return nil
	}

	runtime := &Runtime{
		core:           host,
		registry:       host,
		toolAgents:     host,
		storage:        host,
		bus:            host,
		handlers:       host,
		schedules:      host,
		cfg:            cfg,
		bridge:         agentSandbox.Bridge(),
		agents:         agentSandbox,
		sourcePreparer: enableCfg.sourcePreparer,
		bridgeSubs:     map[string]func(){},
	}
	toolEvaluatorLease, err := host.LeaseToolEvaluator(ctx, runtime.bridge)
	if err != nil {
		agentSandbox.Close()
		return fmt.Errorf("brainkit: lease tool evaluator: %w", err)
	}

	cleanup := func(ctx context.Context, mode runtimeCloseMode) error {
		var err error
		err = errors.Join(err, host.CloseStorageBridgesExceptContext(ctx, existingStorages))
		err = errors.Join(err, toolEvaluatorLease.Close(ctx))
		err = errors.Join(err, agentSandbox.CloseContext(ctx))
		if mode == runtimeCloseModeUnmount {
			err = errors.Join(err, host.RestoreConfiguredStorageRegistry(host.RuntimeConfig(), nil))
		}
		return err
	}
	runtime.cleanup = cleanup
	cleanupActivation := func() error {
		err := runtime.Unmount(ctx)
		host.DetachJSRuntime(runtime)
		host.SetRuntimeConfigJSRuntime(false)
		return err
	}

	runtime.registerBridges()

	bridgeURLs, err := host.InitStorageBridges(cfg)
	if err != nil {
		return errors.Join(fmt.Errorf("brainkit: start storage: %w", err), cleanupActivation())
	}

	if err := host.RegisterConfiguredStorages(cfg, bridgeURLs); err != nil {
		return errors.Join(fmt.Errorf("brainkit: register storages: %w", err), cleanupActivation())
	}
	if err := host.RegisterConfiguredVectors(cfg, bridgeURLs); err != nil {
		return errors.Join(fmt.Errorf("brainkit: register vectors: %w", err), cleanupActivation())
	}

	runtime.deploymentMgr = runtime.newDeploymentManager(cfg)
	if err := host.AttachJSRuntime(runtime); err != nil {
		return errors.Join(err, cleanupActivation())
	}

	if err := runtime.initJSRuntimeGlobals(cfg); err != nil {
		return errors.Join(err, cleanupActivation())
	}

	if len(cfg.Storages) > 0 {
		runtime.upgradeMastraStorage()
	}

	host.SetRuntimeConfigJSRuntime(true)
	runtime.startJobPump()

	host.RestoreJSRuntimeState(cfg)
	return nil
}

func (r *Runtime) newDeploymentManager(cfg types.KernelConfig) *DeploymentManager {
	return NewDeploymentManager(DeploymentManagerConfig{
		Bridge:         r.bridge,
		Agents:         r.agents,
		Tracer:         r.core.Tracer(),
		Store:          cfg.Store,
		ErrorHandler:   cfg.ErrorHandler,
		Logger:         r.core.Logger(),
		SourcePreparer: r.sourcePreparer,
		ToolCleanup: func(id string) {
			r.toolAgents.ToolsDomain().Unregister(context.Background(), id)
		},
		AgentCleanup: func(id string) {
			r.toolAgents.AgentsDomain().Unregister(context.Background(), id)
		},
		SubCleanup: func(id string) {
			cancel := r.removeBridgeSub(id)
			if cancel != nil {
				cancel()
			}
		},
		ScheduleCleanup: func(id string) error {
			if h := r.schedules.ScheduleHandler(); h != nil {
				return h.Unschedule(context.Background(), id)
			}
			return nil
		},
	})
}

func (r *Runtime) Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	return r.deploymentMgr.Deploy(ctx, source, code, opts...)
}

func (r *Runtime) Teardown(ctx context.Context, source string) (int, error) {
	return r.deploymentMgr.Teardown(ctx, source)
}

func (r *Runtime) ListDeployments() []runtimecap.DeploymentInfo {
	return r.deploymentMgr.ListDeployments()
}

func (r *Runtime) EvalJS(ctx context.Context, source, code string) (string, error) {
	return r.deploymentMgr.EvalJS(ctx, source, code)
}

func (r *Runtime) EvalModule(ctx context.Context, source, code string) (string, error) {
	return r.deploymentMgr.EvalModule(ctx, source, code)
}

func (r *Runtime) ListResources(resourceType ...string) ([]types.ResourceInfo, error) {
	return r.deploymentMgr.ListResources(resourceType...)
}

func (r *Runtime) ResourcesFrom(filename string) ([]types.ResourceInfo, error) {
	return r.deploymentMgr.ResourcesFrom(filename)
}

func (r *Runtime) TeardownFile(filename string) (int, error) {
	return r.deploymentMgr.TeardownFile(filename)
}

func (r *Runtime) RemoveResource(resourceType, id string) error {
	return r.deploymentMgr.RemoveResource(resourceType, id)
}

func (r *Runtime) CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error) {
	return r.evalJSCall(ctx, fn, args)
}

func (r *Runtime) Interrupt() {
	if r.bridge != nil {
		r.bridge.Interrupt()
	}
}

func (r *Runtime) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.Shutdown(ctx)
}

func (r *Runtime) CloseContext(ctx context.Context) error {
	return r.Shutdown(ctx)
}

// Unmount strictly detaches the runtime from a live kernel. It returns cleanup
// errors so hot-remount callers can fail loudly instead of leaving stale
// runtime-owned resources attached.
func (r *Runtime) Unmount(ctx context.Context) error {
	return r.closeWithMode(ctx, runtimeCloseModeUnmount)
}

// Shutdown closes the runtime for process teardown. It keeps cleanup
// best-effort for state that only matters to a live, reusable kernel.
func (r *Runtime) Shutdown(ctx context.Context) error {
	return r.closeWithMode(ctx, runtimeCloseModeShutdown)
}

func (r *Runtime) closeWithMode(ctx context.Context, mode runtimeCloseMode) error {
	if ctx == nil {
		ctx = context.Background()
	}
	r.closeMu.Lock()
	defer r.closeMu.Unlock()
	if r.closed {
		return nil
	}
	r.closing = true
	r.closeMode = mode

	var err error
	if mode == runtimeCloseModeUnmount {
		if r.deploymentMgr != nil {
			err = errors.Join(err, r.deploymentMgr.UnmountAll(ctx))
		}
		r.Interrupt()
	} else {
		r.Interrupt()
		if r.deploymentMgr != nil {
			r.deploymentMgr.UnloadAll()
		}
	}

	r.mu.Lock()
	subs := make([]func(), 0, len(r.bridgeSubs))
	for _, cancel := range r.bridgeSubs {
		subs = append(subs, cancel)
	}
	r.bridgeSubs = map[string]func(){}
	r.mu.Unlock()
	for _, cancel := range subs {
		cancel()
	}
	if r.agents != nil {
		r.toolAgents.AgentsDomain().UnregisterAllForKit(r.agents.ID())
	}
	if r.cleanup != nil {
		cleanupErr := r.cleanup(ctx, mode)
		if mode == runtimeCloseModeUnmount {
			err = errors.Join(err, cleanupErr)
		}
	} else if r.agents != nil {
		cleanupErr := r.agents.CloseContext(ctx)
		if mode == runtimeCloseModeUnmount {
			err = errors.Join(err, cleanupErr)
		}
	}
	if err != nil {
		return err
	}
	r.closed = true
	r.closing = false
	return nil
}

func (r *Runtime) CurrentSource() string {
	return r.deploymentMgr.getCurrentSource()
}

func (r *Runtime) SetCurrentSource(source string) {
	r.deploymentMgr.setCurrentSource(source)
}

func (r *Runtime) NextDeployOrder() int {
	return r.deploymentMgr.nextDeployOrder()
}

func (r *Runtime) SetDeployOrderSeed(seed int32) {
	r.deploymentMgr.SetDeployOrderSeed(seed)
}

func (r *Runtime) addBridgeSub(id string, cancel func()) {
	r.mu.Lock()
	r.bridgeSubs[id] = cancel
	r.mu.Unlock()
}

func (r *Runtime) removeBridgeSub(id string) func() {
	r.mu.Lock()
	cancel := r.bridgeSubs[id]
	delete(r.bridgeSubs, id)
	r.mu.Unlock()
	return cancel
}
