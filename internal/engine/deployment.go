package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	js "github.com/brainlet/brainkit/internal/contract"
	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/internal/jsbridge"
	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	typescript "github.com/brainlet/brainkit/vendor_typescript"
)

type DeploymentManager struct {
	mu          syncx.Mutex
	deployments map[string]*deploymentInfo
	deployOrder atomic.Int32

	bridge       *jsbridge.Bridge
	agents       *agentembed.Sandbox
	tracer       *tracing.Tracer
	store        types.KitStore
	errorHandler    func(error, types.ErrorContext)
	logger          *slog.Logger
	resources       *ResourceRegistry
	toolCleanup     func(id string) // called on tool unregister
	agentCleanup    func(id string) // called on agent unregister
	subCleanup      func(id string) // called on subscription cancel
	scheduleCleanup func(id string) // called on schedule cancel

	// currentSource is the .ts file currently being evaluated. Read
	// from arbitrary goroutines (tracing spans, audit source
	// attribution), so the raw slot lives behind an atomic.Value
	// to keep -race quiet. Writers still serialize through
	// setCurrentSource / Deploy.
	currentSource atomic.Value // string
}

type DeploymentManagerConfig struct {
	Bridge          *jsbridge.Bridge
	Agents          *agentembed.Sandbox
	Tracer          *tracing.Tracer
	Store           types.KitStore
	ErrorHandler    func(error, types.ErrorContext)
	Logger          *slog.Logger
	ToolCleanup     func(id string)
	AgentCleanup    func(id string)
	SubCleanup      func(id string)
	ScheduleCleanup func(id string)
}

func NewDeploymentManager(cfg DeploymentManagerConfig) *DeploymentManager {
	return &DeploymentManager{
		deployments:     make(map[string]*deploymentInfo),
		bridge:          cfg.Bridge,
		agents:          cfg.Agents,
		tracer:          cfg.Tracer,
		store:           cfg.Store,
		errorHandler:    cfg.ErrorHandler,
		logger:          cfg.Logger,
		resources:       NewResourceRegistry(),
		toolCleanup:     cfg.ToolCleanup,
		agentCleanup:    cfg.AgentCleanup,
		subCleanup:      cfg.SubCleanup,
		scheduleCleanup: cfg.ScheduleCleanup,
	}
}

// Resources returns the Go-native resource registry for direct registration by bridges.
func (m *DeploymentManager) Resources() *ResourceRegistry {
	return m.resources
}

func (m *DeploymentManager) nextDeployOrder() int {
	return int(m.deployOrder.Add(1))
}

func (m *DeploymentManager) SetDeployOrderSeed(seed int32) {
	m.deployOrder.Store(seed)
}

func (m *DeploymentManager) setCurrentSource(source string) {
	m.currentSource.Store(source)
}

// getCurrentSource returns the source currently being evaluated,
// or "" when nothing is in flight.
func (m *DeploymentManager) getCurrentSource() string {
	v := m.currentSource.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

func (m *DeploymentManager) Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	existing, err := m.validateAndPrepareDeploy(ctx, source)
	if err != nil {
		return nil, err
	}

	span := m.tracer.StartSpan("kit.deploy:"+source, ctx)
	span.SetSource(source)
	defer func() { span.End(nil) }()

	var cfg types.DeployConfig
	if existing != nil {
		cfg.PackageName = existing.PackageName
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	originalCode := code

	m.setCurrentSource(source)
	defer m.setCurrentSource("")

	jsCode, err := m.transpileIfTS(source, code)
	if err != nil {
		return nil, err
	}

	if err := m.evaluateInCompartment(ctx, source, jsCode); err != nil {
		return nil, err
	}

	resources := m.trackDeployment(source)
	m.persistDeployment(ctx, source, originalCode, resources, cfg)

	return resources, nil
}

func (m *DeploymentManager) Teardown(ctx context.Context, source string) (int, error) {
	span := m.tracer.StartSpan("kit.teardown:"+source, ctx)
	span.SetSource(source)
	defer span.End(nil)

	removed, err := m.TeardownFile(source)
	if err != nil {
		return 0, err
	}

	if _, err := m.EvalTS(ctx, "__teardown_compartment.ts", fmt.Sprintf(
		`delete globalThis.%s[%q]; return "ok";`, js.JSCompartments, source)); err != nil {
		m.logger.Warn("teardown: failed to drop compartment", slog.String("source", source), slog.String("error", err.Error()))
	}

	m.mu.Lock()
	delete(m.deployments, source)
	m.mu.Unlock()

	if m.store != nil {
		m.store.DeleteDeployment(source)
	}

	return removed, nil
}

func (m *DeploymentManager) ListDeployments() []deploymentInfo {
	m.mu.Lock()
	sources := make([]string, 0, len(m.deployments))
	for s := range m.deployments {
		sources = append(sources, s)
	}
	m.mu.Unlock()

	result := make([]deploymentInfo, 0, len(sources))
	for _, s := range sources {
		resources, _ := m.ResourcesFrom(s)
		m.mu.Lock()
		d, ok := m.deployments[s]
		m.mu.Unlock()
		if ok {
			result = append(result, deploymentInfo{
				Source:    d.Source,
				CreatedAt: d.CreatedAt,
				Resources: resources,
			})
		}
	}
	return result
}

func (m *DeploymentManager) validateAndPrepareDeploy(ctx context.Context, source string) (*types.PersistedDeployment, error) {
	if strings.TrimSpace(source) == "" {
		return nil, &sdkerrors.ValidationError{Field: "source", Message: "is required"}
	}
	m.mu.Lock()
	_, exists := m.deployments[source]
	m.mu.Unlock()
	if !exists {
		return nil, nil
	}

	var existing *types.PersistedDeployment
	if m.store != nil {
		deps, _ := m.store.LoadDeployments()
		for _, d := range deps {
			if d.Source == source {
				d := d
				existing = &d
				break
			}
		}
	}

	m.Teardown(ctx, source)
	return existing, nil
}

func (m *DeploymentManager) transpileIfTS(source, code string) (string, error) {
	if !strings.HasSuffix(source, ".ts") {
		return code, nil
	}
	js, err := typescript.Transpile(code, typescript.TranspileOptions{FileName: source})
	if err != nil {
		return "", &sdkerrors.DeployError{Source: source, Phase: "transpile", Cause: err}
	}
	return stripESImports(js), nil
}

func (m *DeploymentManager) evaluateInCompartment(ctx context.Context, source, code string) error {
	evalCode := fmt.Sprintf(`
		if (typeof globalThis.Compartment !== "function") {
			throw new Error("SES not available — Compartment not found after lockdown");
		}
		var __endowments = globalThis.__kitEndowments(%q);
		var __c = new globalThis.Compartment({ __options__: true, globals: __endowments });
		globalThis.%s[%q] = __c;
		await __c.evaluate('(async () => { ' + %q + ' })()');
		return "ok";
	`, source, js.JSCompartments, source, code)

	_, err := m.EvalTS(ctx, "__deploy_"+source, evalCode)
	if err != nil {
		m.TeardownFile(source)
		m.EvalTS(ctx, "__deploy_cleanup.ts", fmt.Sprintf(
			`delete globalThis.%s[%q]; return "ok";`, js.JSCompartments, source))
		return &sdkerrors.DeployError{Source: source, Phase: "eval", Cause: err}
	}
	return nil
}

func (m *DeploymentManager) trackDeployment(source string) []types.ResourceInfo {
	resources, err := m.ResourcesFrom(source)
	if err != nil {
		m.logger.Warn("deploy: failed to enumerate resources", slog.String("source", source), slog.String("error", err.Error()))
	}

	order := m.nextDeployOrder()
	now := time.Now()
	m.mu.Lock()
	if m.deployments == nil {
		m.deployments = make(map[string]*deploymentInfo)
	}
	m.deployments[source] = &deploymentInfo{
		Source:    source,
		CreatedAt: now,
		Resources: resources,
		Order:     order,
	}
	m.mu.Unlock()

	return resources
}

func (m *DeploymentManager) persistDeployment(ctx context.Context, source, originalCode string, resources []types.ResourceInfo, cfg types.DeployConfig) {
	if m.store == nil || cfg.Restoring {
		return
	}
	m.mu.Lock()
	order := 0
	if d, ok := m.deployments[source]; ok {
		order = d.Order
	}
	m.mu.Unlock()

	if err := m.store.SaveDeployment(types.PersistedDeployment{
		Source:      source,
		Code:        originalCode,
		Order:       order,
		DeployedAt:  time.Now(),
		PackageName: cfg.PackageName,
	}); err != nil {
		m.persistenceError(ctx, "SaveDeployment", source, err)
	}
}

func (m *DeploymentManager) persistenceError(ctx context.Context, operation, source string, err error) {
	typedErr := &sdkerrors.PersistenceError{Operation: operation, Source: source, Cause: err}
	types.InvokeErrorHandler(m.errorHandler, typedErr, types.ErrorContext{
		Operation: operation, Component: "persistence", Source: source,
	})
}

func (m *DeploymentManager) EvalTS(ctx context.Context, filename, code string) (string, error) {
	wrapped := fmt.Sprintf(`(async () => {
		return await globalThis.__kitRunWithSource(%q, async () => {
			const { bus, kit, model, provider, storage, vectorStore, registry, tools, fs, mcp, output, secrets } = globalThis.__kit;
			%s
		});
	})()`, filename, code)

	if m.bridge.IsEvalBusy() {
		return m.bridge.EvalOnJSThread(filename, wrapped)
	}
	return m.agents.Eval(ctx, filename, wrapped)
}

func (m *DeploymentManager) EvalModule(ctx context.Context, filename, code string) (string, error) {
	m.bridge.Eval("__clear_result.js", `delete globalThis.__module_result`)
	val, err := m.bridge.EvalAsyncModule(filename, code)
	if err != nil {
		return "", fmt.Errorf("brainkit: eval module: %w", err)
	}
	if val != nil {
		val.Free()
	}
	result, err := m.bridge.Eval("__get_result.js",
		`typeof globalThis.__module_result !== 'undefined' ? String(globalThis.__module_result) : ""`)
	if err != nil {
		return "", err
	}
	defer result.Free()
	return result.String(), nil
}

func (m *DeploymentManager) ListResources(resourceType ...string) ([]types.ResourceInfo, error) {
	filter := ""
	if len(resourceType) > 0 {
		filter = resourceType[0]
	}
	entries := m.resources.List(filter)
	return entriesToResourceInfos(entries), nil
}

func (m *DeploymentManager) ResourcesFrom(filename string) ([]types.ResourceInfo, error) {
	entries := m.resources.ListBySource(filename)
	return entriesToResourceInfos(entries), nil
}

// TeardownFile removes all resources for a source.
// 1. Atomically removes entries from Go registry (returns them for cleanup dispatch)
// 2. Runs type-specific Go cleanup (tool unregister, agent unregister, bus unsub, etc.)
// 3. Sweeps stale JS refs + bus subscription handlers
func (m *DeploymentManager) TeardownFile(filename string) (int, error) {
	removed := m.resources.RemoveBySource(filename)
	if len(removed) == 0 {
		return 0, nil
	}

	// Dispatch Go-side cleanup by type
	m.dispatchCleanups(removed)

	// Sweep stale JS-side state (__kit_refs entries + __bus_subs handlers).
	// This is memory cleanup, not correctness — Go already cancelled the real subscriptions.
	m.sweepJSRefs(removed)

	return len(removed), nil
}

func (m *DeploymentManager) RemoveResource(resourceType, id string) error {
	entry, ok := m.resources.Unregister(resourceType, id)
	if !ok {
		return nil
	}
	m.dispatchCleanups([]ResourceEntry{entry})
	m.sweepJSRefs([]ResourceEntry{entry})
	return nil
}

// dispatchCleanups runs Go-native cleanup for each removed resource.
// No JS eval — all cleanup targets are Go subsystems.
func (m *DeploymentManager) dispatchCleanups(entries []ResourceEntry) {
	for _, entry := range entries {
		switch entry.Type {
		case "tool":
			if m.toolCleanup != nil {
				m.toolCleanup(entry.ID)
			}
		case "agent":
			if m.agentCleanup != nil {
				m.agentCleanup(entry.ID)
			}
		case "subscription":
			if m.subCleanup != nil {
				m.subCleanup(entry.ID)
			}
		case "schedule":
			if m.scheduleCleanup != nil {
				m.scheduleCleanup(entry.ID)
			}
		// workflow, memory, topic — no cleanup needed
		}
	}
}

// sweepJSRefs removes stale entries from JS-side __kit_refs and __bus_subs maps.
// Single batch eval — one JS call regardless of entry count.
func (m *DeploymentManager) sweepJSRefs(entries []ResourceEntry) {
	if len(entries) == 0 {
		return
	}
	// Build list of keys to remove from JS
	var keys []string
	var subIDs []string
	for _, e := range entries {
		keys = append(keys, e.Key())
		if e.Type == "subscription" {
			subIDs = append(subIDs, e.ID)
		}
	}
	keysJSON, _ := json.Marshal(keys)   // []string — cannot fail
	subIDsJSON, _ := json.Marshal(subIDs) // []string — cannot fail
	code := fmt.Sprintf(`
		var keys = %s;
		var subIDs = %s;
		var refs = globalThis.__kit_refs;
		var subs = globalThis.__bus_subs;
		var reg = globalThis.__kit_registry;
		if (refs) { for (var i = 0; i < keys.length; i++) delete refs[keys[i]]; }
		if (subs) { for (var i = 0; i < subIDs.length; i++) delete subs[subIDs[i]]; }
		if (reg && reg.cleanups) { for (var i = 0; i < keys.length; i++) delete reg.cleanups[keys[i]]; }
		return "ok";
	`, string(keysJSON), string(subIDsJSON))
	sweepCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := m.EvalTS(sweepCtx, "__sweep_refs.ts", code); err != nil {
		m.logger.Warn("sweepJSRefs: JS eval failed", slog.String("error", err.Error()))
	}
}

func entriesToResourceInfos(entries []ResourceEntry) []types.ResourceInfo {
	infos := make([]types.ResourceInfo, len(entries))
	for i, e := range entries {
		infos[i] = types.ResourceInfo{
			Type:      e.Type,
			ID:        e.ID,
			Name:      e.Name,
			Source:    e.Source,
			CreatedAt: e.CreatedAt.UnixMilli(),
		}
	}
	return infos
}

var esImportRe = regexp.MustCompile(`(?m)^import\s+(type\s+)?(\{[^}]*\}|[^\s]+)\s+from\s+"[^"]+";\s*\n?`)

func stripESImports(js string) string {
	return esImportRe.ReplaceAllString(js, "")
}
