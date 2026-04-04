package crosskit_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/brainlet/brainkit/test/suite/cross"
)

func TestCrossKit_Redis(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Transport("redis"), campaigns.Nodes(2))
	env := infra.Env(t)
	cross.Run(t, env)
}
