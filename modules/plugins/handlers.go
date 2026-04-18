package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// LifecycleDomain handles plugin.start/stop/restart/list/status bus commands.
type LifecycleDomain struct {
	mod *Module
}

func newLifecycleDomain(m *Module) *LifecycleDomain {
	return &LifecycleDomain{mod: m}
}

func (d *LifecycleDomain) Start(ctx context.Context, req sdk.PluginStartMsg) (*sdk.PluginStartResp, error) {
	if req.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "name", Message: "is required"}
	}
	binary := req.Binary
	// If no binary specified, look up from installed plugins
	if binary == "" {
		if store := d.mod.kit.Store(); store != nil {
			installed, err := store.LoadInstalledPlugins()
			if err == nil {
				for _, p := range installed {
					if p.Name == req.Name {
						binary = p.BinaryPath
						break
					}
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
	}
	if err := d.mod.StartPlugin(ctx, cfg); err != nil {
		return nil, err
	}

	pid := 0
	for _, p := range d.mod.ListRunningPlugins() {
		if p.Name == req.Name {
			pid = p.PID
			break
		}
	}
	return &sdk.PluginStartResp{Started: true, Name: req.Name, PID: pid}, nil
}

func (d *LifecycleDomain) Stop(ctx context.Context, req sdk.PluginStopMsg) (*sdk.PluginStopResp, error) {
	if err := d.mod.StopPlugin(ctx, req.Name); err != nil {
		return nil, err
	}
	return &sdk.PluginStopResp{Stopped: true}, nil
}

func (d *LifecycleDomain) Restart(ctx context.Context, req sdk.PluginRestartMsg) (*sdk.PluginRestartResp, error) {
	if err := d.mod.RestartPlugin(ctx, req.Name); err != nil {
		return nil, err
	}
	pid := 0
	for _, p := range d.mod.ListRunningPlugins() {
		if p.Name == req.Name {
			pid = p.PID
			break
		}
	}
	return &sdk.PluginRestartResp{Restarted: true, PID: pid}, nil
}

func (d *LifecycleDomain) List(_ context.Context, _ sdk.PluginListRunningMsg) (*sdk.PluginListRunningResp, error) {
	running := d.mod.ListRunningPlugins()
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

func (d *LifecycleDomain) Status(_ context.Context, req sdk.PluginStatusMsg) (*sdk.PluginStatusResp, error) {
	for _, p := range d.mod.ListRunningPlugins() {
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
