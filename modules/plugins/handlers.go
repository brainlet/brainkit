package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// LifecycleDomain handles plugin.start/stop/restart/list/status bus commands.
type LifecycleDomain struct {
	mod *Module
}

func newLifecycleDomain(m *Module) *LifecycleDomain {
	return &LifecycleDomain{mod: m}
}

func (d *LifecycleDomain) Start(ctx context.Context, req pluginmsg.PluginStartMsg) (*pluginmsg.PluginStartResp, error) {
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

	cfg := PluginConfig{
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
	return &pluginmsg.PluginStartResp{Started: true, Name: req.Name, PID: pid}, nil
}

func (d *LifecycleDomain) Stop(ctx context.Context, req pluginmsg.PluginStopMsg) (*pluginmsg.PluginStopResp, error) {
	if err := d.mod.StopPlugin(ctx, req.Name); err != nil {
		return nil, err
	}
	return &pluginmsg.PluginStopResp{Stopped: true}, nil
}

func (d *LifecycleDomain) Restart(ctx context.Context, req pluginmsg.PluginRestartMsg) (*pluginmsg.PluginRestartResp, error) {
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
	return &pluginmsg.PluginRestartResp{Restarted: true, PID: pid}, nil
}

func (d *LifecycleDomain) List(_ context.Context, _ pluginmsg.PluginListRunningMsg) (*pluginmsg.PluginListRunningResp, error) {
	running := d.mod.ListRunningPlugins()
	infos := make([]pluginmsg.RunningPluginInfo, 0, len(running))
	for _, p := range running {
		infos = append(infos, pluginmsg.RunningPluginInfo{
			Name:     p.Name,
			PID:      p.PID,
			Uptime:   p.Uptime.Round(time.Second).String(),
			Status:   p.Status,
			Restarts: p.Restarts,
		})
	}
	return &pluginmsg.PluginListRunningResp{Plugins: infos}, nil
}

func (d *LifecycleDomain) Status(_ context.Context, req pluginmsg.PluginStatusMsg) (*pluginmsg.PluginStatusResp, error) {
	for _, p := range d.mod.ListRunningPlugins() {
		if p.Name == req.Name {
			return &pluginmsg.PluginStatusResp{
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
