package engine

import (
	"context"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

type DeploymentInfo struct {
	Source    string               `json:"source"`
	CreatedAt time.Time            `json:"createdAt"`
	Resources []types.ResourceInfo `json:"resources,omitempty"`
	Order     int                  `json:"order"`
}

func (k *Kernel) Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	if k.jsRuntime == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.Deploy(ctx, source, code, opts...)
}

func (k *Kernel) Teardown(ctx context.Context, source string) (int, error) {
	if k.jsRuntime == nil {
		return 0, &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.Teardown(ctx, source)
}

func (k *Kernel) ListDeployments() []DeploymentInfo {
	if k.jsRuntime == nil {
		return []DeploymentInfo{}
	}
	return k.jsRuntime.ListDeployments()
}
