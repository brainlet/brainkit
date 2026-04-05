package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	typescript "github.com/brainlet/brainkit/vendor_typescript"
)

// LifecycleDomain handles Deploy/Teardown/Redeploy/List operations.
type LifecycleDomain struct {
	kit *Kernel
}

func newLifecycleDomain(k *Kernel) *LifecycleDomain {
	return &LifecycleDomain{kit: k}
}

func (d *LifecycleDomain) Deploy(ctx context.Context, req messages.KitDeployMsg) (*messages.KitDeployResp, error) {
	resources, err := d.kit.Deploy(ctx, req.Source, req.Code)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, messages.KitDeployedEvent{
		Source:    req.Source,
		Resources: resourceInfosToMessages(resources),
	})
	return &messages.KitDeployResp{
		Deployed:  true,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

func (d *LifecycleDomain) Teardown(ctx context.Context, req messages.KitTeardownMsg) (*messages.KitTeardownResp, error) {
	removed, err := d.kit.Teardown(ctx, req.Source)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, messages.KitTeardownedEvent{
		Source:  req.Source,
		Removed: removed,
	})
	return &messages.KitTeardownResp{Removed: removed}, nil
}

func (d *LifecycleDomain) Redeploy(ctx context.Context, req messages.KitRedeployMsg) (*messages.KitRedeployResp, error) {
	resources, err := d.kit.Redeploy(ctx, req.Source, req.Code)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, messages.KitDeployedEvent{
		Source:    req.Source,
		Resources: resourceInfosToMessages(resources),
	})
	return &messages.KitRedeployResp{
		Deployed:  true,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

func (d *LifecycleDomain) List(_ context.Context, _ messages.KitListMsg) (*messages.KitListResp, error) {
	deployments := d.kit.ListDeployments()
	out := make([]messages.DeploymentInfo, 0, len(deployments))
	for _, deployment := range deployments {
		out = append(out, messages.DeploymentInfo{
			Source:    deployment.Source,
			CreatedAt: deployment.CreatedAt.Format(time.RFC3339),
			Resources: resourceInfosToMessages(deployment.Resources),
		})
	}
	return &messages.KitListResp{Deployments: out}, nil
}

func resourceInfosToMessages(resources []ResourceInfo) []messages.ResourceInfo {
	out := make([]messages.ResourceInfo, 0, len(resources))
	for _, resource := range resources {
		out = append(out, messages.ResourceInfo{
			Type:      resource.Type,
			ID:        resource.ID,
			Name:      resource.Name,
			Source:    resource.Source,
			CreatedAt: resource.CreatedAt,
		})
	}
	return out
}

func (d *LifecycleDomain) publishLifecycleEvent(ctx context.Context, event messages.BrainkitMessage) error {
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
// DeployOption configures a Deploy call.
type DeployOption func(*deployConfig)

type deployConfig struct {
	role        string
	packageName string
	restoring   bool
}

// WithRestoring marks this Deploy as a restore from persistence — skips re-persist.
// Short-term fix: long-term the restore path should be a separate method.
func WithRestoring() DeployOption {
	return func(c *deployConfig) { c.restoring = true }
}

// WithRole assigns an RBAC role to the deployment.
func WithRole(role string) DeployOption {
	return func(c *deployConfig) { c.role = role }
}

// WithPackageName tags the deployment as part of a package.
func WithPackageName(name string) DeployOption {
	return func(c *deployConfig) { c.packageName = name }
}

func (k *Kernel) Deploy(ctx context.Context, source, code string, opts ...DeployOption) ([]ResourceInfo, error) {
	// Phase 1: Validate
	if err := k.validateDeploy(source); err != nil {
		return nil, err
	}

	// Tracing wraps the entire deploy
	span := k.tracer.StartSpan("kit.deploy:"+source, ctx)
	span.SetSource(source)
	defer func() { span.End(nil) }()

	var cfg deployConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	originalCode := code

	// Set Go-side source for RBAC — subscribe calls during deployment capture this
	k.setCurrentSource(source)
	defer k.setCurrentSource("")

	if cfg.role != "" && k.rbac != nil {
		k.rbac.Assign(source, cfg.role)
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

// validateDeploy checks source is non-empty and not already deployed.
func (k *Kernel) validateDeploy(source string) error {
	if strings.TrimSpace(source) == "" {
		return &sdkerrors.ValidationError{Field: "source", Message: "is required"}
	}
	k.mu.Lock()
	_, exists := k.deployments[source]
	k.mu.Unlock()
	if exists {
		return &sdk.AlreadyExistsError{Resource: "deployment", Name: source, Hint: "use Redeploy"}
	}
	return nil
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
		globalThis.__kit_compartments[%q] = __c;
		await __c.evaluate('(async () => { ' + %q + ' })()');
		return "ok";
	`, source, source, code)

	_, err := k.EvalTS(ctx, "__deploy_"+source, evalCode)
	if err != nil {
		k.TeardownFile(source)
		k.EvalTS(ctx, "__deploy_cleanup.ts", fmt.Sprintf(
			`delete globalThis.__kit_compartments[%q]; return "ok";`, source))
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
// Skips when restoring from persistence (cfg.restoring) to avoid overwriting metadata.
func (k *Kernel) persistDeployment(ctx context.Context, source, originalCode string, resources []ResourceInfo, cfg deployConfig) {
	if k.config.Store == nil || cfg.restoring {
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
		Role:        cfg.role,
		PackageName: cfg.packageName,
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
		`delete globalThis.__kit_compartments[%q]; return "ok";`, source)); err != nil {
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

// Redeploy tears down old deployment and deploys new code.
// Preserves original metadata (role, packageName, order) across the teardown+deploy cycle.
func (k *Kernel) Redeploy(ctx context.Context, source, code string, opts ...DeployOption) ([]ResourceInfo, error) {
	span := k.tracer.StartSpan("kit.redeploy:"+source, ctx)
	span.SetSource(source)
	defer span.End(nil)

	// Capture original metadata before teardown
	k.mu.Lock()
	originalOrder := 0
	if d, ok := k.deployments[source]; ok {
		originalOrder = d.Order
	}
	k.mu.Unlock()

	// Read persisted metadata (in-memory deploymentInfo doesn't store role/packageName)
	originalRole := ""
	originalPkgName := ""
	if k.config.Store != nil {
		deps, _ := k.config.Store.LoadDeployments()
		for _, d := range deps {
			if d.Source == source {
				originalRole = d.Role
				originalPkgName = d.PackageName
				break
			}
		}
	}

	if _, err := k.Teardown(ctx, source); err != nil {
		k.logger.Warn("redeploy: teardown failed", slog.String("source", source), slog.String("error", err.Error()))
	}

	// Merge original metadata with any explicit opts (explicit opts win)
	var mergedOpts []DeployOption
	if originalRole != "" {
		mergedOpts = append(mergedOpts, WithRole(originalRole))
	}
	if originalPkgName != "" {
		mergedOpts = append(mergedOpts, WithPackageName(originalPkgName))
	}
	mergedOpts = append(mergedOpts, opts...) // explicit opts override originals
	resources, err := k.Deploy(ctx, source, code, mergedOpts...)
	if err != nil {
		return nil, err
	}

	// Restore original order (Deploy assigned a new one).
	// Only update order — Deploy already persisted the correct role/packageName.
	if originalOrder > 0 {
		k.mu.Lock()
		if d, ok := k.deployments[source]; ok {
			d.Order = originalOrder
		}
		k.mu.Unlock()
		if k.config.Store != nil {
			// Read back what Deploy saved, just update the order
			deps, _ := k.config.Store.LoadDeployments()
			for _, d := range deps {
				if d.Source == source {
					d.Order = originalOrder
					k.config.Store.SaveDeployment(d)
					break
				}
			}
		}
	}

	return resources, nil
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
