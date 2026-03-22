package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/brainlet/brainkit/sdk/messages"
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
}

// Deploy evaluates code in a new SES Compartment with isolated globals.
// Resources created inside the compartment are tracked by source name.
// Uses EvalTS internally — handles reentrant calls (IsEvalBusy) and Value.Free.
func (k *Kernel) Deploy(ctx context.Context, source, code string) ([]ResourceInfo, error) {
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

	resources, err := k.ResourcesFrom(source)
	if err != nil {
		log.Printf("[kit] deploy %s: failed to enumerate resources: %v", source, err)
	}

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
func (k *Kernel) Teardown(ctx context.Context, source string) (int, error) {
	removed, err := k.TeardownFile(source)
	if err != nil {
		return 0, err
	}

	// Drop compartment reference (uses EvalTS for proper Value.Free + reentrant safety)
	if _, err := k.EvalTS(ctx, "__teardown_compartment.ts", fmt.Sprintf(
		`delete globalThis.__kit_compartments[%q]; return "ok";`, source)); err != nil {
		log.Printf("[kit] teardown %s: failed to drop compartment: %v", source, err)
	}

	k.mu.Lock()
	delete(k.deployments, source)
	k.mu.Unlock()

	return removed, nil
}

// Redeploy tears down old deployment and deploys new code.
// If teardown fails, it's logged but deploy proceeds (old resources may be gone).
func (k *Kernel) Redeploy(ctx context.Context, source, code string) ([]ResourceInfo, error) {
	if _, err := k.Teardown(ctx, source); err != nil {
		log.Printf("redeploy teardown %s: %v", source, err)
	}
	return k.Deploy(ctx, source, code)
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
