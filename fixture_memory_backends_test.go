//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestFixture_TS_MemoryMongoDB(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	t.Logf("MongoDB container running at %s", mongoURL)

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

	code := loadFixture(t, "testdata/ts/memory-mongodb.js")
	result, err := kit.EvalModule(context.Background(), "memory-mongodb.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		Remembers bool   `json:"remembers"`
		Store     string `json:"store"`
		URL       string `json:"url"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Remembers {
		t.Errorf("didn't remember: %q", out.Text)
	}
	t.Logf("fixture memory-mongodb: %q remembers=%v store=%s url=%s", out.Text, out.Remembers, out.Store, out.URL)
}

func TestFixture_TS_MemoryUpstash(t *testing.T) {
	loadEnv(t)
	upstashURL := os.Getenv("UPSTASH_REDIS_REST_URL")
	upstashToken := os.Getenv("UPSTASH_REDIS_REST_TOKEN")
	if upstashURL == "" || upstashToken == "" {
		t.Skip("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN not set")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"UPSTASH_REDIS_REST_URL":   upstashURL,
			"UPSTASH_REDIS_REST_TOKEN": upstashToken,
			"OPENAI_API_KEY":           key,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/memory-upstash.js")
	result, err := kit.EvalModule(context.Background(), "memory-upstash.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		Remembers bool   `json:"remembers"`
		Store     string `json:"store"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Remembers {
		t.Errorf("didn't remember: %q", out.Text)
	}
	t.Logf("fixture memory-upstash: %q remembers=%v store=%s", out.Text, out.Remembers, out.Store)
}

func TestFixture_TS_MemoryPostgres(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
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
		t.Fatalf("could not start Postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	pgURL := fmt.Sprintf("postgresql://test:test@%s:%s/brainlet_test", host, port.Port())
	t.Logf("Postgres container running at %s", pgURL)

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

	code := loadFixture(t, "testdata/ts/memory-postgres.js")
	result, err := kit.EvalModule(context.Background(), "memory-postgres.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		Remembers bool   `json:"remembers"`
		Store     string `json:"store"`
		URL       string `json:"url"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Remembers {
		t.Errorf("didn't remember: %q", out.Text)
	}
	t.Logf("fixture memory-postgres: %q remembers=%v store=%s", out.Text, out.Remembers, out.Store)
}

func TestFixture_TS_MemoryPostgresSCRAM(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "testpass123",
				"POSTGRES_DB":       "brainlet_test",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("could not start Postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	pgURL := fmt.Sprintf("postgresql://test:testpass123@%s:%s/brainlet_test", host, port.Port())
	t.Logf("Postgres SCRAM container at %s", pgURL)

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

	code := loadFixture(t, "testdata/ts/memory-postgres-scram.js")
	result, err := kit.EvalModule(context.Background(), "memory-postgres-scram.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		Remembers bool   `json:"remembers"`
		Auth      string `json:"auth"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Remembers {
		t.Errorf("SCRAM auth memory failed: %q", out.Text)
	}
	t.Logf("fixture memory-postgres-scram: %q remembers=%v auth=%s", out.Text, out.Remembers, out.Auth)
}
