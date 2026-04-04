package fullstack_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/agents"
	"github.com/brainlet/brainkit/test/suite/bus"
	"github.com/brainlet/brainkit/test/suite/deploy"
	"github.com/brainlet/brainkit/test/suite/gateway"
	"github.com/brainlet/brainkit/test/suite/health"
	"github.com/brainlet/brainkit/test/suite/persistence"
	"github.com/brainlet/brainkit/test/suite/registry"
	"github.com/brainlet/brainkit/test/suite/secrets"
	"github.com/brainlet/brainkit/test/suite/tools"
	"github.com/brainlet/brainkit/test/suite/tracing"
	"github.com/brainlet/brainkit/test/suite/workflows"
)

// TestFullStack_Redis_MongoDB exercises another production combo:
// Redis transport, MongoDB storage, persistence, and tracing.
func TestFullStack_Redis_MongoDB(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t,
		campaigns.Transport("redis"),
		campaigns.Storage("mongodb"),
		campaigns.Persistence(),
		campaigns.Tracing(),
	)
	env := infra.Env(t)

	bus.Run(t, env)
	deploy.Run(t, env)
	tools.Run(t, env)
	agents.Run(t, env)
	health.Run(t, env)
	secrets.Run(t, env)
	registry.Run(t, env)
	workflows.Run(t, env)
	tracing.Run(t, env)
	persistence.Run(t, env)
	gateway.Run(t, env)
}
