package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/sdk"
	typescript "github.com/brainlet/brainkit/vendor_typescript"
)

// LifecycleDomain handles Deploy/Teardown/Redeploy/List operations.
type LifecycleDomain struct {
	kit *Kernel
}

func newLifecycleDomain(k *Kernel) *LifecycleDomain {
	return &LifecycleDomain{kit: k}
}

func (d *LifecycleDomain) Deploy(ctx context.Context, req sdk.KitDeployMsg) (*sdk.KitDeployResp, error) {
	var opts []DeployOption
	if req.Role != "" {
		opts = append(opts, WithRole(req.Role))
	}
	if req.PackageName != "" {
		opts = append(opts, WithPackageName(req.PackageName))
	}
	resources, err := d.kit.Deploy(ctx, req.Source, req.Code, opts...)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, sdk.KitDeployedEvent{
		Source:    req.Source,
		RuntimeID: d.kit.config.RuntimeID,
		Resources: resourceInfosToMessages(resources),
	})
	return &sdk.KitDeployResp{
		Deployed:  true,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

func (d *LifecycleDomain) Teardown(ctx context.Context, req sdk.KitTeardownMsg) (*sdk.KitTeardownResp, error) {
	removed, err := d.kit.Teardown(ctx, req.Source)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, sdk.KitTeardownedEvent{
		Source:    req.Source,
		RuntimeID: d.kit.config.RuntimeID,
		Removed:   removed,
	})
	return &sdk.KitTeardownResp{Removed: removed}, nil
}

func (d *LifecycleDomain) Redeploy(ctx context.Context, req sdk.KitRedeployMsg) (*sdk.KitRedeployResp, error) {
	resources, err := d.kit.Deploy(ctx, req.Source, req.Code)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, sdk.KitDeployedEvent{
		Source:    req.Source,
		Resources: resourceInfosToMessages(resources),
	})
	return &sdk.KitRedeployResp{
		Deployed:  true,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

func (d *LifecycleDomain) List(_ context.Context, _ sdk.KitListMsg) (*sdk.KitListResp, error) {
	deployments := d.kit.ListDeployments()
	out := make([]sdk.DeploymentInfo, 0, len(deployments))
	for _, deployment := range deployments {
		out = append(out, sdk.DeploymentInfo{
			Source:    deployment.Source,
			CreatedAt: deployment.CreatedAt.Format(time.RFC3339),
			Resources: resourceInfosToMessages(deployment.Resources),
		})
	}
	return &sdk.KitListResp{Deployments: out}, nil
}

func resourceInfosToMessages(resources []ResourceInfo) []sdk.ResourceInfo {
	out := make([]sdk.ResourceInfo, 0, len(resources))
	for _, resource := range resources {
		out = append(out, sdk.ResourceInfo{
			Type:      resource.Type,
			ID:        resource.ID,
			Name:      resource.Name,
			Source:    resource.Source,
			CreatedAt: resource.CreatedAt,
		})
	}
	return out
}

func (d *LifecycleDomain) publishLifecycleEvent(ctx context.Context, event sdk.BrainkitMessage) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return d.kit.publish(ctx, event.BusTopic(), payload)
}

type deploymentInfo struct {
	Source    string         `json:"source"`
	CreatedAt time.Time     `json:"createdAt"`
	Resources []ResourceInfo `json:"resources,omitempty"`
	Order     int            `json:"order"`
}

// Deploy evaluates code in a new SES Compartment with isolated globals.
// Resources created inside the compartment are tracked by source name.
// Uses EvalTS internally — handles reentrant calls (IsEvalBusy) and Value.Free.
func (k *Kernel) Deploy(ctx context.Context, source, code string, opts ...DeployOption) ([]ResourceInfo, error) {
	// Phase 1: Validate + teardown if already deployed (idempotent)
	// Captures metadata from existing deployment for merge.
	existing, err := k.validateAndPrepareDeploy(ctx, source)
	if err != nil {
		return nil, err
	}

	// Tracing wraps the entire deploy
	span := k.tracer.StartSpan("kit.deploy:"+source, ctx)
	span.SetSource(source)
	defer func() { span.End(nil) }()

	var cfg DeployConfig
	// Merge existing metadata first (previous role, packageName)
	if existing != nil {
		cfg.Role = existing.Role
		cfg.PackageName = existing.PackageName
	}
	// Explicit opts override existing metadata
	for _, opt := range opts {
		opt(&cfg)
	}
	originalCode := code

	// Set Go-side source for RBAC — subscribe calls during deployment capture this
	k.setCurrentSource(source)
	defer k.setCurrentSource("")

	if cfg.Role != "" && k.rbac != nil {
		k.rbac.Assign(source, cfg.Role)
	}

	// Phase 2: Transpile (.ts → JS + strip imports)
	jsCode, err := k.transpileIfTS(source, code)
	if err != nil {
		return nil, err
	}

	// Phase 3: Evaluate in SES Compartment
	if err := k.evaluateInCompartment(ctx, source, jsCode); err != nil {
		return nil, err
	}

	// Phase 4: Track deployment + enumerate resources
	resources := k.trackDeployment(source)

	// Phase 5: Persist to KitStore
	k.persistDeployment(ctx, source, originalCode, resources, cfg)

	return resources, nil
}

// validateAndPrepareDeploy checks source is non-empty. If already deployed, captures
// persisted metadata (role, packageName) and tears down the existing deployment.
// Returns the persisted metadata if available (nil if fresh deploy).
func (k *Kernel) validateAndPrepareDeploy(ctx context.Context, source string) (*PersistedDeployment, error) {
	if strings.TrimSpace(source) == "" {
		return nil, &sdkerrors.ValidationError{Field: "source", Message: "is required"}
	}
	k.mu.Lock()
	_, exists := k.deployments[source]
	k.mu.Unlock()
	if !exists {
		return nil, nil
	}

	// Capture persisted metadata before teardown
	var existing *PersistedDeployment
	if k.config.Store != nil {
		deps, _ := k.config.Store.LoadDeployments()
		for _, d := range deps {
			if d.Source == source {
				d := d
				existing = &d
				break
			}
		}
	}

	k.Teardown(ctx, source)
	return existing, nil
}

// transpileIfTS converts .ts source to JS, stripping type annotations and ES imports.
// Returns the original code unchanged for .js sources.
func (k *Kernel) transpileIfTS(source, code string) (string, error) {
	if !strings.HasSuffix(source, ".ts") {
		return code, nil
	}
	js, err := typescript.Transpile(code, typescript.TranspileOptions{FileName: source})
	if err != nil {
		return "", &sdkerrors.DeployError{Source: source, Phase: "transpile", Cause: err}
	}
	return stripESImports(js), nil
}

// evaluateInCompartment creates a SES Compartment and evaluates the JS code inside it.
// On failure, cleans up partial resources and the compartment reference.
func (k *Kernel) evaluateInCompartment(ctx context.Context, source, code string) error {
	evalCode := fmt.Sprintf(`
		if (typeof globalThis.Compartment !== "function") {
			throw new Error("SES not available — Compartment not found after lockdown");
		}
		var __endowments = globalThis.__kitEndowments(%q);
		var __c = new globalThis.Compartment({ __options__: true, globals: __endowments });
		globalThis.`+js.JSCompartments+`[%q] = __c;
		await __c.evaluate('(async () => { ' + %q + ' })()');
		return "ok";
	`, source, source, code)

	_, err := k.EvalTS(ctx, "__deploy_"+source, evalCode)
	if err != nil {
		k.TeardownFile(source)
		k.EvalTS(ctx, "__deploy_cleanup.ts", fmt.Sprintf(
			`delete globalThis.`+js.JSCompartments+`[%q]; return "ok";`, source))
		return &sdkerrors.DeployError{Source: source, Phase: "eval", Cause: err}
	}
	return nil
}

// trackDeployment records the deployment and enumerates its resources.
func (k *Kernel) trackDeployment(source string) []ResourceInfo {
	resources, err := k.ResourcesFrom(source)
	if err != nil {
		k.logger.Warn("deploy: failed to enumerate resources", slog.String("source", source), slog.String("error", err.Error()))
	}

	order := k.nextDeployOrder()
	now := time.Now()
	k.mu.Lock()
	if k.deployments == nil {
		k.deployments = make(map[string]*deploymentInfo)
	}
	k.deployments[source] = &deploymentInfo{
		Source:    source,
		CreatedAt: now,
		Resources: resources,
		Order:     order,
	}
	k.mu.Unlock()

	return resources
}

// persistDeployment saves the deployment to KitStore for restart recovery.
// Skips when restoring from persistence (cfg.Restoring) to avoid overwriting metadata.
func (k *Kernel) persistDeployment(ctx context.Context, source, originalCode string, resources []ResourceInfo, cfg DeployConfig) {
	if k.config.Store == nil || cfg.Restoring {
		return
	}
	// Read back the order from the tracked deployment
	k.mu.Lock()
	order := 0
	if d, ok := k.deployments[source]; ok {
		order = d.Order
	}
	k.mu.Unlock()

	if err := k.config.Store.SaveDeployment(PersistedDeployment{
		Source:      source,
		Code:        originalCode,
		Order:       order,
		DeployedAt:  time.Now(),
		Role:        cfg.Role,
		PackageName: cfg.PackageName,
	}); err != nil {
		k.persistenceError(ctx, "SaveDeployment", source, err)
	}
}

// Teardown removes all resources from a deployed file and drops the compartment.
// Idempotent — returns 0 if source was not deployed.
func (k *Kernel) Teardown(ctx context.Context, source string) (int, error) {
	span := k.tracer.StartSpan("kit.teardown:"+source, ctx)
	span.SetSource(source)
	defer span.End(nil)

	removed, err := k.TeardownFile(source)
	if err != nil {
		return 0, err
	}

	// Drop compartment reference (uses EvalTS for proper Value.Free + reentrant safety)
	if _, err := k.EvalTS(ctx, "__teardown_compartment.ts", fmt.Sprintf(
		`delete globalThis.`+js.JSCompartments+`[%q]; return "ok";`, source)); err != nil {
		k.logger.Warn("teardown: failed to drop compartment", slog.String("source", source), slog.String("error", err.Error()))
	}

	k.mu.Lock()
	delete(k.deployments, source)
	k.mu.Unlock()

	// Remove from persistence
	if k.config.Store != nil {
		k.config.Store.DeleteDeployment(source)
	}

	return removed, nil
}



// ListDeployments returns all currently deployed files with their resources.
func (k *Kernel) ListDeployments() []deploymentInfo {
	k.mu.Lock()
	sources := make([]string, 0, len(k.deployments))
	for s := range k.deployments {
		sources = append(sources, s)
	}
	k.mu.Unlock()

	result := make([]deploymentInfo, 0, len(sources))
	for _, s := range sources {
		resources, _ := k.ResourcesFrom(s) // best-effort: deployment info still returned if this fails
		k.mu.Lock()
		d, ok := k.deployments[s]
		k.mu.Unlock()
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

// esImportRe matches ES import lines (value and type imports).
// Stripped because kit.Deploy runs code in a SES Compartment where
// all symbols are injected as endowments (globals), not ES modules.
var esImportRe = regexp.MustCompile(`(?m)^import\s+(type\s+)?(\{[^}]*\}|[^\s]+)\s+from\s+"[^"]+";\s*\n?`)

// stripESImports removes ES import/export-from lines from transpiled JS.
func stripESImports(js string) string {
	return esImportRe.ReplaceAllString(js, "")
}
