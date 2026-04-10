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
	"github.com/brainlet/brainkit/internal/rbac"
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
	rbac         *rbac.Manager
	store        types.KitStore
	errorHandler func(error, types.ErrorContext)
	logger       *slog.Logger

	currentSource string
}

type DeploymentManagerConfig struct {
	Bridge       *jsbridge.Bridge
	Agents       *agentembed.Sandbox
	Tracer       *tracing.Tracer
	RBAC         *rbac.Manager
	Store        types.KitStore
	ErrorHandler func(error, types.ErrorContext)
	Logger       *slog.Logger
}

func NewDeploymentManager(cfg DeploymentManagerConfig) *DeploymentManager {
	return &DeploymentManager{
		deployments:  make(map[string]*deploymentInfo),
		bridge:       cfg.Bridge,
		agents:       cfg.Agents,
		tracer:       cfg.Tracer,
		rbac:         cfg.RBAC,
		store:        cfg.Store,
		errorHandler: cfg.ErrorHandler,
		logger:       cfg.Logger,
	}
}

func (m *DeploymentManager) nextDeployOrder() int {
	return int(m.deployOrder.Add(1))
}

func (m *DeploymentManager) SetDeployOrderSeed(seed int32) {
	m.deployOrder.Store(seed)
}

func (m *DeploymentManager) setCurrentSource(source string) {
	m.currentSource = source
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
		cfg.Role = existing.Role
		cfg.PackageName = existing.PackageName
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	originalCode := code

	m.setCurrentSource(source)
	defer m.setCurrentSource("")

	if cfg.Role != "" && m.rbac != nil {
		m.rbac.Assign(source, cfg.Role)
	}

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
		Role:        cfg.Role,
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
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.list(%q))`, filter)
	result, err := m.EvalTS(context.Background(), "__list_resources.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []types.ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	return resources, nil
}

func (m *DeploymentManager) ResourcesFrom(filename string) ([]types.ResourceInfo, error) {
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.listBySource(%q))`, filename)
	result, err := m.EvalTS(context.Background(), "__resources_from.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []types.ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("resources from: %w", err)
	}
	return resources, nil
}

func (m *DeploymentManager) TeardownFile(filename string) (int, error) {
	code := fmt.Sprintf(`
		var resources = globalThis.__kit_registry.listBySource(%q);
		var count = 0;
		for (var i = resources.length - 1; i >= 0; i--) {
			globalThis.__kit_registry.unregister(resources[i].type, resources[i].id);
			count++;
		}
		return JSON.stringify(count);
	`, filename)
	result, err := m.EvalTS(context.Background(), "__teardown_file.ts", code)
	if err != nil {
		return 0, err
	}
	var count int
	if err := json.Unmarshal([]byte(result), &count); err != nil {
		return 0, nil
	}
	return count, nil
}

func (m *DeploymentManager) RemoveResource(resourceType, id string) error {
	code := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.unregister(%q, %q);
		return JSON.stringify(entry !== null);
	`, resourceType, id)
	_, err := m.EvalTS(context.Background(), "__remove_resource.ts", code)
	return err
}

var esImportRe = regexp.MustCompile(`(?m)^import\s+(type\s+)?(\{[^}]*\}|[^\s]+)\s+from\s+"[^"]+";\s*\n?`)

func stripESImports(js string) string {
	return esImportRe.ReplaceAllString(js, "")
}
