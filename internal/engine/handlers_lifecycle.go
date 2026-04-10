package engine

import (
	"context"
	"encoding/json"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// LifecycleDomain handles Deploy/Teardown/Redeploy/List operations.
type LifecycleDomain struct {
	deployer  Deployer
	bus       BusPublisher
	audit     *auditpkg.Recorder
	runtimeID string
}

func newLifecycleDomain(deployer Deployer, bus BusPublisher, audit *auditpkg.Recorder, runtimeID string) *LifecycleDomain {
	return &LifecycleDomain{deployer: deployer, bus: bus, audit: audit, runtimeID: runtimeID}
}

func (d *LifecycleDomain) Deploy(ctx context.Context, req sdk.KitDeployMsg) (*sdk.KitDeployResp, error) {
	var opts []types.DeployOption
	if req.Role != "" {
		opts = append(opts, types.WithRole(req.Role))
	}
	if req.PackageName != "" {
		opts = append(opts, types.WithPackageName(req.PackageName))
	}
	resources, err := d.deployer.Deploy(ctx, req.Source, req.Code, opts...)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, sdk.KitDeployedEvent{
		Source:    req.Source,
		RuntimeID: d.runtimeID,
		Resources: resourceInfosToMessages(resources),
	})
	d.audit.Deployed(req.Source, len(resources))
	return &sdk.KitDeployResp{
		Deployed:  true,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

func (d *LifecycleDomain) Teardown(ctx context.Context, req sdk.KitTeardownMsg) (*sdk.KitTeardownResp, error) {
	removed, err := d.deployer.Teardown(ctx, req.Source)
	if err != nil {
		return nil, err
	}
	_ = d.publishLifecycleEvent(ctx, sdk.KitTeardownedEvent{
		Source:    req.Source,
		RuntimeID: d.runtimeID,
		Removed:   removed,
	})
	d.audit.Teardown(req.Source)
	return &sdk.KitTeardownResp{Removed: removed}, nil
}

func (d *LifecycleDomain) Redeploy(ctx context.Context, req sdk.KitRedeployMsg) (*sdk.KitRedeployResp, error) {
	resources, err := d.deployer.Deploy(ctx, req.Source, req.Code)
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
	deployments := d.deployer.ListDeployments()
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

func resourceInfosToMessages(resources []types.ResourceInfo) []sdk.ResourceInfo {
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
	_, err = d.bus.PublishRaw(ctx, event.BusTopic(), payload)
	return err
}

type deploymentInfo struct {
	Source    string               `json:"source"`
	CreatedAt time.Time            `json:"createdAt"`
	Resources []types.ResourceInfo `json:"resources,omitempty"`
	Order     int                  `json:"order"`
}

func (k *Kernel) Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	return k.deploymentMgr.Deploy(ctx, source, code, opts...)
}

func (k *Kernel) Teardown(ctx context.Context, source string) (int, error) {
	return k.deploymentMgr.Teardown(ctx, source)
}

func (k *Kernel) ListDeployments() []deploymentInfo {
	return k.deploymentMgr.ListDeployments()
}
