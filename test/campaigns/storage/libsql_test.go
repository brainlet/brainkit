package storage_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns"
)

func TestStorage_LibSQL(t *testing.T) {
	campaigns.RequirePodman(t)
	infra := campaigns.NewInfra(t, campaigns.Storage("libsql"), campaigns.AI())
	infra.RunFixtures(t, "memory/storage/libsql*", "agent/memory/libsql")
}
