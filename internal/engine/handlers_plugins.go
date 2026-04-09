package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/sdk"
)

// PluginLifecycleDomain handles plugin.start/stop/restart/list/status bus commands.
type PluginLifecycleDomain struct {
	node *Node
}

func newPluginLifecycleDomain(n *Node) *PluginLifecycleDomain {
	return &PluginLifecycleDomain{node: n}
}

func (d *PluginLifecycleDomain) Start(ctx context.Context, req sdk.PluginStartMsg) (*sdk.PluginStartResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	binary := req.Binary
	// If no binary specified, look up from installed plugins
	if binary == "" && d.node.Kernel.config.Store != nil {
		installed, err := d.node.Kernel.config.Store.LoadInstalledPlugins()
		if err == nil {
			for _, p := range installed {
				if p.Name == req.Name {
					binary = p.BinaryPath
					break
				}
			}
		}
	}
	if binary == "" {
		return nil, fmt.Errorf("plugin.start: no binary path for plugin %q (not installed and no binary specified)", req.Name)
	}

	cfg := types.PluginConfig{
		Name:   req.Name,
		Binary: binary,
		Env:    req.Env,
		Config: req.Config,
		Role:   req.Role,
	}
	if err := d.node.StartPlugin(ctx, cfg); err != nil {
		return nil, err
	}

	pid := 0
	for _, p := range d.node.ListRunningPlugins() {
		if p.Name == req.Name {
			pid = p.PID
			break
		}
	}
	return &sdk.PluginStartResp{Started: true, Name: req.Name, PID: pid}, nil
}

func (d *PluginLifecycleDomain) Stop(ctx context.Context, req sdk.PluginStopMsg) (*sdk.PluginStopResp, error) {
	if err := d.node.StopPlugin(ctx, req.Name); err != nil {
		return nil, err
	}
	return &sdk.PluginStopResp{Stopped: true}, nil
}

func (d *PluginLifecycleDomain) Restart(ctx context.Context, req sdk.PluginRestartMsg) (*sdk.PluginRestartResp, error) {
	if err := d.node.RestartPlugin(ctx, req.Name); err != nil {
		return nil, err
	}
	pid := 0
	for _, p := range d.node.ListRunningPlugins() {
		if p.Name == req.Name {
			pid = p.PID
			break
		}
	}
	return &sdk.PluginRestartResp{Restarted: true, PID: pid}, nil
}

func (d *PluginLifecycleDomain) List(_ context.Context, _ sdk.PluginListRunningMsg) (*sdk.PluginListRunningResp, error) {
	running := d.node.ListRunningPlugins()
	infos := make([]sdk.RunningPluginInfo, 0, len(running))
	for _, p := range running {
		infos = append(infos, sdk.RunningPluginInfo{
			Name:     p.Name,
			PID:      p.PID,
			Uptime:   p.Uptime.Round(time.Second).String(),
			Status:   p.Status,
			Restarts: p.Restarts,
		})
	}
	return &sdk.PluginListRunningResp{Plugins: infos}, nil
}

func (d *PluginLifecycleDomain) Status(_ context.Context, req sdk.PluginStatusMsg) (*sdk.PluginStatusResp, error) {
	for _, p := range d.node.ListRunningPlugins() {
		if p.Name == req.Name {
			return &sdk.PluginStatusResp{
				Name:     p.Name,
				PID:      p.PID,
				Status:   p.Status,
				Uptime:   p.Uptime.Round(time.Second).String(),
				Restarts: p.Restarts,
			}, nil
		}
	}
	return nil, fmt.Errorf("plugin %q not running", req.Name)
}
