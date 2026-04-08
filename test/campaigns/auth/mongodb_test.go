package auth_test

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMongoDB_SCRAM_SHA256(t *testing.T) {
	campaigns.RequirePodman(t)
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
		wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
		"MONGO_INITDB_ROOT_USERNAME=scramuser", "MONGO_INITDB_ROOT_PASSWORD=scrampass")
	waitForTCP(t, addr, 15*time.Second)

	k := newKit(t, map[string]string{
		"MONGODB_URL":     "mongodb://scramuser:scrampass@" + addr,
		"MONGODB_LOG_ALL": "off",
	})

	result := evalStore(t, k, "mongodb-scram-sha256", `
		var store = new embed.MongoDBStore({
			id: "mongo-scram256-test",
			url: process.env.MONGODB_URL,
			dbName: "authtest",
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

func TestMongoDB_SCRAM_SHA1(t *testing.T) {
	campaigns.RequirePodman(t)
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t, "mongo:7", "27017/tcp",
		[]string{"mongod", "--setParameter", "authenticationMechanisms=SCRAM-SHA-1"},
		wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
		"MONGO_INITDB_ROOT_USERNAME=sha1user", "MONGO_INITDB_ROOT_PASSWORD=sha1pass")
	waitForTCP(t, addr, 15*time.Second)

	k := newKit(t, map[string]string{
		"MONGODB_URL":     "mongodb://sha1user:sha1pass@" + addr + "/?authMechanism=SCRAM-SHA-1",
		"MONGODB_LOG_ALL": "off",
	})

	result := evalStore(t, k, "mongodb-scram-sha1", `
		var store = new embed.MongoDBStore({
			id: "mongo-scram1-test",
			url: process.env.MONGODB_URL,
			dbName: "authtest",
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

func TestMongoDB_NoAuth(t *testing.T) {
	campaigns.RequirePodman(t)
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
		wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second))
	waitForTCP(t, addr, 15*time.Second)

	k := newKit(t, map[string]string{
		"MONGODB_URL": "mongodb://" + addr,
	})

	result := evalStore(t, k, "mongodb-noauth", `
		var store = new embed.MongoDBStore({
			id: "mongo-noauth-test",
			url: process.env.MONGODB_URL,
			dbName: "authtest",
		});
	`)
	require.Contains(t, result, `"ok":true`)
}
