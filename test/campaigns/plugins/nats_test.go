package plugins_test

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/cross"
)

func TestPlugins_NATS(t *testing.T) {
	campaigns.RequirePodman(t)

	pluginBinary := testutil.BuildTestPlugin(t)

	infra := campaigns.NewInfra(t,
		campaigns.Transport("nats"),
		campaigns.Nodes(2),
		campaigns.Plugins(brainkit.PluginConfig{
			Name:         "testplugin",
			Binary:       pluginBinary,
			StartTimeout: 30 * time.Second,
		}),
	)
	env := infra.Env(t)
	cross.Run(t, env)
}
