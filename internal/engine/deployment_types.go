package engine

import (
	"context"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

type deploymentInfo struct {
	Source    string               `json:"source"`
	CreatedAt time.Time            `json:"createdAt"`
	Resources []types.ResourceInfo `json:"resources,omitempty"`
	Order     int                  `json:"order"`
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

func (k *Kernel) Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	return k.deploymentMgr.Deploy(ctx, source, code, opts...)
}

func (k *Kernel) Teardown(ctx context.Context, source string) (int, error) {
	return k.deploymentMgr.Teardown(ctx, source)
}

func (k *Kernel) ListDeployments() []deploymentInfo {
	return k.deploymentMgr.ListDeployments()
}
