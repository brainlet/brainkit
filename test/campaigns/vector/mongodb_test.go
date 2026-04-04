package vector_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
)

func TestVector_MongoDB(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Vector("mongodb"), campaigns.AI())
	infra.RunFixtures(t, "vector/mongodb-*")
}
