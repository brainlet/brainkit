package brainkit

import (
	"context"
	"fmt"
	"log"
	"time"
)

type deploymentInfo struct {
	Source    string         `json:"source"`
	CreatedAt time.Time     `json:"createdAt"`
	Resources []ResourceInfo `json:"resources,omitempty"`
}

// Deploy evaluates code in a new SES Compartment with isolated globals.
// Resources created inside the compartment are tracked by source name.
// Uses EvalTS internally — handles reentrant calls (IsEvalBusy) and Value.Free.
func (k *Kit) Deploy(ctx context.Context, source, code string) ([]ResourceInfo, error) {
	k.mu.Lock()
	if k.deployments != nil {
		if _, exists := k.deployments[source]; exists {
			k.mu.Unlock()
			return nil, fmt.Errorf("%s is already deployed (use Redeploy)", source)
		}
	}
	k.mu.Unlock()

	// EvalTS handles: IsEvalBusy reentrant guard, async promise resolution,
	// Value.Free on return. Wraps code in Compartment with per-source endowments.
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
		// Cleanup any partial resources created before the error
		k.TeardownFile(source)
		// Remove compartment reference if it was stored
		k.EvalTS(ctx, "__deploy_cleanup.ts", fmt.Sprintf(
			`delete globalThis.__kit_compartments[%q]; return "ok";`, source))
		return nil, fmt.Errorf("deploy %s: %w", source, err)
	}

	resources, _ := k.ResourcesFrom(source)

	k.mu.Lock()
	if k.deployments == nil {
		k.deployments = make(map[string]*deploymentInfo)
	}
	k.deployments[source] = &deploymentInfo{
		Source:    source,
		CreatedAt: time.Now(),
		Resources: resources,
	}
	k.mu.Unlock()

	return resources, nil
}

// Teardown removes all resources from a deployed file and drops the compartment.
// Idempotent — returns 0 if source was not deployed.
func (k *Kit) Teardown(ctx context.Context, source string) (int, error) {
	removed, err := k.TeardownFile(source)
	if err != nil {
		return 0, err
	}

	// Drop compartment reference (uses EvalTS for proper Value.Free + reentrant safety)
	k.EvalTS(ctx, "__teardown_compartment.ts", fmt.Sprintf(
		`delete globalThis.__kit_compartments[%q]; return "ok";`, source))

	k.mu.Lock()
	delete(k.deployments, source)
	k.mu.Unlock()

	return removed, nil
}

// Redeploy tears down old deployment and deploys new code.
// If teardown fails, it's logged but deploy proceeds (old resources may be gone).
func (k *Kit) Redeploy(ctx context.Context, source, code string) ([]ResourceInfo, error) {
	if _, err := k.Teardown(ctx, source); err != nil {
		log.Printf("redeploy teardown %s: %v", source, err)
	}
	return k.Deploy(ctx, source, code)
}

// ListDeployments returns all currently deployed files with their resources.
func (k *Kit) ListDeployments() []deploymentInfo {
	k.mu.Lock()
	sources := make([]string, 0, len(k.deployments))
	for s := range k.deployments {
		sources = append(sources, s)
	}
	k.mu.Unlock()

	result := make([]deploymentInfo, 0, len(sources))
	for _, s := range sources {
		resources, _ := k.ResourcesFrom(s)
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
