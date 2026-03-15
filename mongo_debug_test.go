package brainkit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDebug_MongoDriverConnect(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7",
			ExposedPorts: []string{"27017/tcp"},
			WaitingFor:   wait.ForListeningPort("27017/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("could not start MongoDB container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "27017")
	mongoURL := fmt.Sprintf("mongodb://%s:%s/?directConnection=true&serverSelectionTimeoutMS=10000&connectTimeoutMS=5000", host, port.Port())
	t.Logf("MongoDB at %s", mongoURL)

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"MONGODB_URL": mongoURL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	result, err := kit.EvalTS(ctx, "mongo-test.js", `
		const url = globalThis.process.env.MONGODB_URL;
		const embed = globalThis.__agent_embed;
		const steps = [];

		try {
			// Step 1: Create store
			const store = new embed.MongoDBStore({
				id: "test",
				url: url,
				dbName: "brainlet_test",
			});
			steps.push("store_created");

			// Step 2: Init store
			await store.init();
			steps.push("store_inited");

			// Step 3: Get the memory sub-store
			const memStore = await store.getStore("memory");
			steps.push("memStore:" + (memStore ? "ok" : "null"));

			// Step 4: Check memStore structure
			steps.push("memStore_type:" + memStore.constructor.name);
			steps.push("memStore_keys:" + Object.getOwnPropertyNames(Object.getPrototypeOf(memStore)).slice(0,5).join(","));

			// Step 5: Try getThreadById
			try {
				const thread = await memStore.getThreadById({ threadId: "test-1" });
				steps.push("thread:" + (thread ? "found" : "null"));
			} catch(e2) {
				steps.push("getThread_ERROR:" + e2.message + " | stack:" + (e2.stack || "").split("\\n").slice(0,3).join(" | "));
			}

			return JSON.stringify(steps);
		} catch (e) {
			steps.push("ERROR:" + e.message);
			return JSON.stringify(steps);
		}
	`)
	if err != nil {
		t.Logf("EvalTS error: %v", err)
	} else {
		t.Logf("Result: %s", result)
	}
}
