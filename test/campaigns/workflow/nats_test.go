package workflow_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/workflows"
)

func TestWorkflow_NATS(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t,
		campaigns.Transport("nats"),
		campaigns.Persistence(),
	)
	env := infra.Env(t)
	workflows.RunStableSubset(t, env)
}
