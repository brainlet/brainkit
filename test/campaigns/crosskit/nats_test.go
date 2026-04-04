package crosskit_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/cross"
)

func TestCrossKit_NATS(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Transport("nats"), campaigns.Nodes(2))
	env := infra.Env(t)
	cross.Run(t, env)
}
