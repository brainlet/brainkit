package vector_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
)

func TestVector_LibSQL(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Vector("libsql"), campaigns.AI())
	infra.RunFixtures(t, "vector/*/libsql")
}
