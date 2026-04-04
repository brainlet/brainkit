package storage_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
)

func TestStorage_Postgres(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Storage("postgres"), campaigns.AI())
	infra.RunFixtures(t, "memory/postgres-*", "agent/with-memory-postgres")
}
