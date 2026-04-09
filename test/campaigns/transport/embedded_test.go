package transport_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/agents"
	"github.com/brainlet/brainkit/test/suite/bus"
	"github.com/brainlet/brainkit/test/suite/deploy"
	"github.com/brainlet/brainkit/test/suite/fs"
	"github.com/brainlet/brainkit/test/suite/gateway"
	"github.com/brainlet/brainkit/test/suite/health"
	"github.com/brainlet/brainkit/test/suite/mcp"
	"github.com/brainlet/brainkit/test/suite/persistence"
	"github.com/brainlet/brainkit/test/suite/registry"
	"github.com/brainlet/brainkit/test/suite/scheduling"
	"github.com/brainlet/brainkit/test/suite/secrets"
	"github.com/brainlet/brainkit/test/suite/tools"
	"github.com/brainlet/brainkit/test/suite/tracing"
	"github.com/brainlet/brainkit/test/suite/workflows"
)

// TestTransport_Embedded tests all domains on the embedded NATS transport.
// No Podman needed — embedded NATS runs in-process.
func TestTransport_Embedded(t *testing.T) {
	infra := campaigns.NewInfra(t, campaigns.Transport("embedded"))
	env := infra.Env(t)

	bus.Run(t, env)
	deploy.Run(t, env)
	tools.Run(t, env)
	agents.Run(t, env)
	scheduling.Run(t, env)
	health.Run(t, env)
	secrets.Run(t, env)
	registry.Run(t, env)
	mcp.Run(t, env)
	workflows.Run(t, env)
	tracing.Run(t, env)
	fs.Run(t, env)
	persistence.Run(t, env)
	gateway.Run(t, env)
}
