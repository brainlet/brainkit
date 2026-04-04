package storage_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
)

func TestStorage_MongoDB(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Storage("mongodb"), campaigns.AI())
	infra.RunFixtures(t, "memory/mongodb-*", "agent/with-memory-mongodb")
}
