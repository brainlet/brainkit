package plugins_test

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/cross"
)

func TestPlugins_Redis(t *testing.T) {
	campaigns.RequirePodman(t)

	pluginBinary := testutil.BuildTestPlugin(t)

	infra := campaigns.NewInfra(t,
		campaigns.Transport("redis"),
		campaigns.Nodes(2),
		campaigns.Plugins(pluginsmod.PluginConfig{
			Name:         "testplugin",
			Binary:       pluginBinary,
			StartTimeout: 30 * time.Second,
		}),
	)
	env := infra.Env(t)
	cross.Run(t, env)
}
