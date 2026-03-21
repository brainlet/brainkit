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

func TestFixture_TS_MemorySemanticRecall(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"LIBSQL_URL":     libsqlURL,
			"OPENAI_API_KEY": key,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/memory-semantic-recall.js")
	result, err := kit.EvalModule(context.Background(), "memory-semantic-recall.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text          string `json:"text"`
		RemembersRust bool   `json:"remembersRust"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.RemembersRust {
		t.Errorf("semantic recall failed — didn't remember Rust: %q", out.Text)
	}
	t.Logf("semantic-recall: %q remembers=%v", out.Text, out.RemembersRust)
}

func TestFixture_TS_MemoryWorking(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"LIBSQL_URL":     libsqlURL,
			"OPENAI_API_KEY": key,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/memory-working.js")
	result, err := kit.EvalModule(context.Background(), "memory-working.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text          string `json:"text"`
		KnowsName     bool   `json:"knowsName"`
		KnowsLocation bool   `json:"knowsLocation"`
		KnowsLanguage bool   `json:"knowsLanguage"`
	}
	json.Unmarshal([]byte(result), &out)

	known := 0
	if out.KnowsName {
		known++
	}
	if out.KnowsLocation {
		known++
	}
	if out.KnowsLanguage {
		known++
	}

	if known < 2 {
		t.Errorf("working memory: only knew %d/3 facts: name=%v location=%v language=%v text=%q",
			known, out.KnowsName, out.KnowsLocation, out.KnowsLanguage, out.Text)
	}
	t.Logf("working-memory: %q name=%v location=%v language=%v", out.Text, out.KnowsName, out.KnowsLocation, out.KnowsLanguage)
}
