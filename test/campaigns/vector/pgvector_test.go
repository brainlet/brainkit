package vector_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
)

func TestVector_PgVector(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Vector("pgvector"), campaigns.AI())
	infra.RunFixtures(t, "vector/*/pgvector")
}
