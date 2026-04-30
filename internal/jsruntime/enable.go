package jsruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/internal/jsbridge"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/modulehost/providerhost"
)

// Runtime owns the optional embedded JS/TS runtime for a Kernel.
type Runtime struct {
	core          runtimecap.CoreHost
	registry      runtimecap.RegistryHost
	toolAgents    runtimecap.ToolAgentHost
	bus           runtimecap.BusHost
	handlers      runtimecap.HandlerHost
	schedules     runtimecap.ScheduleHost
	cfg           types.KernelConfig
	bridge        *jsbridge.Bridge
	agents        *agentembed.Sandbox
	deploymentMgr *DeploymentManager

	mu         sync.Mutex
	bridgeSubs map[string]func()
}

// Enable starts the embedded JS/TS runtime for a light Kernel. It is
// idempotent and can be called during construction or from a hot-mounted
// jsruntime module.
func Enable(ctx context.Context, host runtimecap.Host) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if host.HasJSRuntime() {
		return nil
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
		core:       host,
		registry:   host,
		toolAgents: host,
		bus:        host,
		handlers:   host,
		schedules:  host,
		cfg:        cfg,
		bridge:     agentSandbox.Bridge(),
		agents:     agentSandbox,
		bridgeSubs: map[string]func(){},
	}
	host.SetToolEvaluator(runtime.bridge)

	cleanup := func() {
		host.CloseStorageBridgesExcept(existingStorages)
		host.SetToolEvaluator(nil)
		host.DetachJSRuntime(runtime)
		agentSandbox.Close()
		host.RegisterConfiguredStorages(host.RuntimeConfig(), nil)
		_ = host.RegisterConfiguredVectors(host.RuntimeConfig(), nil)
	}

	runtime.registerBridges()

	bridgeURLs, err := host.InitStorageBridges(cfg)
	if err != nil {
		cleanup()
		return fmt.Errorf("brainkit: start storage: %w", err)
	}

	host.RegisterConfiguredStorages(cfg, bridgeURLs)
	if err := host.RegisterConfiguredVectors(cfg, bridgeURLs); err != nil {
		cleanup()
		return fmt.Errorf("brainkit: register vectors: %w", err)
	}

	runtime.deploymentMgr = runtime.newDeploymentManager(cfg)
	if err := host.AttachJSRuntime(runtime); err != nil {
		cleanup()
		return err
	}

	if err := runtime.initJSRuntimeGlobals(cfg); err != nil {
		cleanup()
		return err
	}

	if len(cfg.Storages) > 0 {
		runtime.upgradeMastraStorage()
	}

	host.SetRuntimeConfigJSRuntime(true)
	runtime.startJobPump()

	host.RestoreJSRuntimeState(cfg)

	go host.ProbeAll()
	return nil
}

func (r *Runtime) newDeploymentManager(cfg types.KernelConfig) *DeploymentManager {
	return NewDeploymentManager(DeploymentManagerConfig{
		Bridge:       r.bridge,
		Agents:       r.agents,
		Tracer:       r.core.Tracer(),
		Store:        cfg.Store,
		ErrorHandler: cfg.ErrorHandler,
		Logger:       r.core.Logger(),
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
		ScheduleCleanup: func(id string) {
			if h := r.schedules.ScheduleHandler(); h != nil {
				_ = h.Unschedule(context.Background(), id)
			}
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

func (r *Runtime) EvalTS(ctx context.Context, source, code string) (string, error) {
	return r.deploymentMgr.EvalTS(ctx, source, code)
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

func (r *Runtime) CallJSSync(fn string, args any) {
	r.evalJSCallSync(fn, args)
}

func (r *Runtime) Interrupt() {
	if r.bridge != nil {
		r.bridge.Interrupt()
	}
}

func (r *Runtime) Close() error {
	r.Interrupt()
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
		r.agents.Close()
	}
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
