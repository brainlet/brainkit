//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestFixture_TS_VectorMethods(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/tursodatabase/libsql-server:latest",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/health").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("could not start LibSQL container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8080")
	libsqlURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	kit, err := New(Config{
		Namespace: "test",
		EnvVars: map[string]string{
			"LIBSQL_URL": libsqlURL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/vector-methods.js")
	result, err := kit.EvalModule(context.Background(), "vector-methods.js", code)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("raw: %s", result)

	var out struct {
		ListIndexes struct {
			HasA  bool `json:"hasA"`
			HasB  bool `json:"hasB"`
			Count int  `json:"count"`
		} `json:"listIndexes"`
		DescribeIndex any `json:"describeIndex"`
		Query         struct {
			TopId string `json:"topId"`
			Count int    `json:"count"`
		} `json:"query"`
		AfterDelete struct {
			Count int      `json:"count"`
			Ids   []string `json:"ids"`
		} `json:"afterDelete"`
		IndexesAfter []string `json:"indexesAfter"`
		AllPassed    bool     `json:"allPassed"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.ListIndexes.HasA || !out.ListIndexes.HasB {
		t.Errorf("listIndexes missing: hasA=%v hasB=%v", out.ListIndexes.HasA, out.ListIndexes.HasB)
	}
	if out.Query.TopId != "v1" {
		t.Errorf("query top: expected v1, got %s", out.Query.TopId)
	}
	if out.AfterDelete.Count != 1 {
		t.Errorf("after delete: expected 1, got %d (%v)", out.AfterDelete.Count, out.AfterDelete.Ids)
	}
	t.Logf("vector-methods: listIndexes=%v describe=%v query=%v afterDelete=%v indexesAfter=%v",
		out.ListIndexes, out.DescribeIndex, out.Query, out.AfterDelete, out.IndexesAfter)
}

func TestFixture_TS_VectorPgVector(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "pgvector/pgvector:pg16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":             "test",
				"POSTGRES_PASSWORD":         "test",
				"POSTGRES_DB":               "brainlet_test",
				"POSTGRES_HOST_AUTH_METHOD": "trust",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("could not start pgvector container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	pgURL := fmt.Sprintf("postgresql://test:test@%s:%s/brainlet_test", host, port.Port())

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"POSTGRES_URL":   pgURL,
			"OPENAI_API_KEY": key,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/vector-pgvector.js")
	result, err := kit.EvalModule(context.Background(), "vector-pgvector.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		ResultCount int    `json:"resultCount"`
		TopID       string `json:"topId"`
		TopLabel    string `json:"topLabel"`
		SecondID    string `json:"secondId"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.ResultCount < 2 {
		t.Errorf("expected 2+ results, got %d", out.ResultCount)
	}
	if out.TopLabel != "x" && out.TopLabel != "xy" {
		t.Errorf("expected top result to be x or xy, got %q", out.TopLabel)
	}
	t.Logf("pgvector: %d results, top=%s(%s) second=%s", out.ResultCount, out.TopID, out.TopLabel, out.SecondID)
}

func TestFixture_TS_VectorMongoDB(t *testing.T) {
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
	mongoURL := fmt.Sprintf("mongodb://%s:%s/?directConnection=true", host, port.Port())

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"MONGODB_URL":    mongoURL,
			"OPENAI_API_KEY": key,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/vector-mongodb.js")
	result, err := kit.EvalModule(context.Background(), "vector-mongodb.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Created  bool   `json:"created"`
		Atlas    bool   `json:"atlas"`
		Upserted int    `json:"upserted"`
		Reason   string `json:"reason"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Atlas {
		t.Logf("vector-mongodb: Atlas Search available, index created")
	} else if out.Upserted == 2 {
		t.Logf("vector-mongodb: community edition, upsert works, reason=%s", out.Reason)
	} else {
		t.Errorf("vector-mongodb: unexpected result: %+v", out)
	}
}
