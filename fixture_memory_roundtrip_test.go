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

func TestFixture_TS_MemoryInMemory(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/memory-inmemory.js")

	result, err := kit.EvalModule(context.Background(), "memory-inmemory.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text          string `json:"text"`
		RemembersName bool   `json:"remembersName"`
		RemembersWork bool   `json:"remembersWork"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.RemembersName {
		t.Errorf("didn't remember name: %q", out.Text)
	}
	if !out.RemembersWork {
		t.Errorf("didn't remember work: %q", out.Text)
	}
	t.Logf("fixture memory-inmemory: %q name=%v work=%v", out.Text, out.RemembersName, out.RemembersWork)
}

func TestFixture_TS_MemoryLibSQL(t *testing.T) {
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
	t.Logf("LibSQL container running at %s", libsqlURL)

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"LIBSQL_URL": libsqlURL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/memory-libsql.js")
	result, err := kit.EvalModule(context.Background(), "memory-libsql.js", code)
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
	t.Logf("fixture memory-libsql: %q remembers=%v store=%s url=%s", out.Text, out.Remembers, out.Store, out.URL)
}
