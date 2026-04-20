package workflow_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/workflows"
)

func TestWorkflow_Embedded(t *testing.T) {
	infra := campaigns.NewInfra(t,
		campaigns.Transport("embedded"),
		campaigns.Persistence(),
	)
	env := infra.Env(t)
	workflows.RunStableSubset(t, env)
}
