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

func TestFixture_TS_RAGChunkText(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/rag-chunk-text.js")
	result, err := kit.EvalModule(context.Background(), "rag-chunk-text.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		ChunkCount     int    `json:"chunkCount"`
		HasMultiple    bool   `json:"hasMultiple"`
		FirstChunkText string `json:"firstChunkText"`
		AllHaveText    bool   `json:"allHaveText"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.HasMultiple {
		t.Errorf("expected multiple chunks, got %d: %s", out.ChunkCount, result)
	}
	if !out.AllHaveText {
		t.Error("some chunks have no text")
	}
	t.Logf("rag-chunk-text: %d chunks, first=%q", out.ChunkCount, out.FirstChunkText)
}

func TestFixture_TS_RAGChunkMarkdown(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/rag-chunk-markdown.js")
	result, err := kit.EvalModule(context.Background(), "rag-chunk-markdown.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		ChunkCount  int      `json:"chunkCount"`
		HasMultiple bool     `json:"hasMultiple"`
		Texts       []string `json:"texts"`
		Metadata    []any    `json:"metadata"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.HasMultiple {
		t.Errorf("expected multiple chunks, got %d: %s", out.ChunkCount, result)
	}
	t.Logf("rag-chunk-markdown: %d chunks, texts=%v", out.ChunkCount, out.Texts)
}

func TestFixture_TS_RAGVectorQueryTool(t *testing.T) {
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

	code := loadFixture(t, "testdata/ts/rag-vector-query-tool.js")
	result, err := kit.EvalModule(context.Background(), "rag-vector-query-tool.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		ChunkCount  int    `json:"chunkCount"`
		ResultCount int    `json:"resultCount"`
		HasResults  bool   `json:"hasResults"`
		TopResult   string `json:"topResult"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.HasResults {
		t.Errorf("no results from vector query: %s", result)
	}
	t.Logf("rag-vector-query: %d chunks, %d results, top=%q",
		out.ChunkCount, out.ResultCount, out.TopResult)
}

func TestFixture_TS_RAGChunkToken(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/rag-chunk-token.js")
	result, err := kit.EvalModule(context.Background(), "rag-chunk-token.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		ChunkCount      int    `json:"chunkCount"`
		HasMultiple     bool   `json:"hasMultiple"`
		AllHaveText     bool   `json:"allHaveText"`
		TotalTextLength int    `json:"totalTextLength"`
		Error           string `json:"error"`
		Stack           string `json:"stack"`
	}
	json.Unmarshal([]byte(result), &out)
	if out.Error != "" {
		t.Fatalf("token chunking error: %s\n%s", out.Error, out.Stack)
	}
	if !out.HasMultiple {
		t.Errorf("expected multiple token chunks, got %d: %s", out.ChunkCount, result)
	}
	t.Logf("rag-chunk-token: %d chunks from %d chars", out.ChunkCount, out.TotalTextLength)
}
