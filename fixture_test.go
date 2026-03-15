package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/registry"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// loadFixture reads a test fixture file from testdata/.
func loadFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", path, err)
	}
	return string(data)
}

// ═══════════════════════════════════════════════════════════════
// .ts FIXTURES — real modules developers would write
// ═══════════════════════════════════════════════════════════════

func TestFixture_TS_AgentGenerate(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-generate.js")

	result, err := kit.EvalModule(context.Background(), "agent-generate.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text         string `json:"text"`
		HasUsage     bool   `json:"hasUsage"`
		FinishReason string `json:"finishReason"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(strings.ToUpper(out.Text), "FIXTURE_WORKS") {
		t.Errorf("text = %q", out.Text)
	}
	if !out.HasUsage {
		t.Error("expected usage")
	}
	t.Logf("fixture agent-generate: %q finish=%s", out.Text, out.FinishReason)
}

func TestFixture_TS_AgentStream(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-stream.js")

	result, err := kit.EvalModule(context.Background(), "agent-stream.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text              string `json:"text"`
		Chunks            int    `json:"chunks"`
		HasRealTimeTokens bool   `json:"hasRealTimeTokens"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Text == "" {
		t.Error("expected non-empty text")
	}
	if !out.HasRealTimeTokens {
		t.Error("expected real-time token chunks")
	}
	t.Logf("fixture agent-stream: %d chunks, text=%q", out.Chunks, out.Text)
}

func TestFixture_TS_AgentWithLocalTool(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-with-local-tool.js")

	result, err := kit.EvalModule(context.Background(), "agent-with-local-tool.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		ToolCalls int    `json:"toolCalls"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(out.Text, "42") {
		t.Errorf("expected 42: %q", out.Text)
	}
	t.Logf("fixture agent-with-local-tool: %q toolCalls=%d", out.Text, out.ToolCalls)
}

func TestFixture_TS_AgentWithRegisteredTool(t *testing.T) {
	kit := newTestKit(t)

	// Register the "multiply" tool that the fixture expects
	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.multiply", ShortName: "multiply", Namespace: "platform",
		Description: "Multiplies two numbers",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"number","description":"first number"},"b":{"type":"number","description":"second number"}},"required":["a","b"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ A, B float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.A * args.B})
				return result, nil
			},
		},
	})

	code := loadFixture(t, "testdata/ts/agent-with-registered-tool.js")
	result, err := kit.EvalModule(context.Background(), "agent-with-registered-tool.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		ToolCalls int    `json:"toolCalls"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(out.Text, "42") {
		t.Errorf("expected 42: %q", out.Text)
	}
	t.Logf("fixture agent-with-registered-tool: %q toolCalls=%d", out.Text, out.ToolCalls)
}

func TestFixture_TS_AIGenerate(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-generate.js")

	result, err := kit.EvalModule(context.Background(), "ai-generate.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text     string `json:"text"`
		HasUsage bool   `json:"hasUsage"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(strings.ToUpper(out.Text), "DIRECT") {
		t.Errorf("text = %q", out.Text)
	}
	t.Logf("fixture ai-generate: %q", out.Text)
}

func TestFixture_TS_AIGenerateObject(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-generate-object.js")

	result, err := kit.EvalModule(context.Background(), "ai-generate-object.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Object       map[string]interface{} `json:"object"`
		HasName      bool                   `json:"hasName"`
		HasAge       bool                   `json:"hasAge"`
		HasHobbies   bool                   `json:"hasHobbies"`
		HasUsage     bool                   `json:"hasUsage"`
		FinishReason string                 `json:"finishReason"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.HasName {
		t.Errorf("object missing name: %v", out.Object)
	}
	if !out.HasAge {
		t.Errorf("object missing age: %v", out.Object)
	}
	if !out.HasHobbies {
		t.Errorf("object missing hobbies: %v", out.Object)
	}
	if !out.HasUsage {
		t.Error("expected usage")
	}
	t.Logf("fixture ai-generate-object: %v finish=%s", out.Object, out.FinishReason)
}

func TestFixture_TS_AgentWithMemory(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-with-memory.js")

	result, err := kit.EvalModule(context.Background(), "agent-with-memory.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		Remembers bool   `json:"remembers"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Remembers {
		t.Errorf("agent didn't remember: %q", out.Text)
	}
	t.Logf("fixture agent-with-memory: %q remembers=%v", out.Text, out.Remembers)
}

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

	// No POSTGRES_HOST_AUTH_METHOD — defaults to scram-sha-256
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

func TestFixture_TS_VectorPgVector(t *testing.T) {
	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// pgvector/pgvector image has the vector extension pre-installed
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

	// On Community Edition: createIndex fails (no Atlas Search), but upsert works
	if out.Atlas {
		t.Logf("vector-mongodb: Atlas Search available, index created")
	} else if out.Upserted == 2 {
		t.Logf("vector-mongodb: community edition, upsert works, reason=%s", out.Reason)
	} else {
		t.Errorf("vector-mongodb: unexpected result: %+v", out)
	}
}

func TestFixture_TS_AIStream(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-stream.js")

	result, err := kit.EvalModule(context.Background(), "ai-stream.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text              string `json:"text"`
		Chunks            int    `json:"chunks"`
		HasRealTimeTokens bool   `json:"hasRealTimeTokens"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Text == "" {
		t.Error("expected non-empty text")
	}
	if !out.HasRealTimeTokens {
		t.Error("expected real-time token chunks")
	}
	t.Logf("fixture ai-stream: %d chunks, text=%q", out.Chunks, out.Text)
}

func TestFixture_TS_WorkflowBasic(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-basic.js")

	result, err := kit.EvalModule(context.Background(), "workflow-basic.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Status   string `json:"status"`
		Result   any    `json:"result"`
		Expected string `json:"expected"`
		Match    bool   `json:"match"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Match {
		t.Errorf("workflow result mismatch: status=%s result=%v expected=%s", out.Status, out.Result, out.Expected)
	}
	t.Logf("workflow-basic: status=%s match=%v", out.Status, out.Match)
}

func TestFixture_TS_WorkflowWithAgent(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/workflow-with-agent.js")

	result, err := kit.EvalModule(context.Background(), "workflow-with-agent.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Status    string `json:"status"`
		Result    any    `json:"result"`
		HasAnswer bool   `json:"hasAnswer"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.HasAnswer {
		t.Errorf("workflow+agent: status=%s result=%v", out.Status, out.Result)
	}
	t.Logf("workflow-with-agent: status=%s hasAnswer=%v result=%v", out.Status, out.HasAnswer, out.Result)
}

func TestFixture_TS_AIEmbed(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-embed.js")

	result, err := kit.EvalModule(context.Background(), "ai-embed.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Single struct {
			Dimensions int  `json:"dimensions"`
			HasValues  bool `json:"hasValues"`
		} `json:"single"`
		Multi struct {
			Count      int  `json:"count"`
			Dimensions int  `json:"dimensions"`
			AllVectors bool `json:"allVectors"`
		} `json:"multi"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Single.HasValues || out.Single.Dimensions == 0 {
		t.Errorf("single embed failed: dims=%d hasValues=%v", out.Single.Dimensions, out.Single.HasValues)
	}
	if out.Multi.Count != 3 || !out.Multi.AllVectors {
		t.Errorf("multi embed failed: count=%d allVectors=%v", out.Multi.Count, out.Multi.AllVectors)
	}
	t.Logf("ai-embed: single=%d dims, multi=%d vectors × %d dims", out.Single.Dimensions, out.Multi.Count, out.Multi.Dimensions)
}

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
	if out.KnowsName { known++ }
	if out.KnowsLocation { known++ }
	if out.KnowsLanguage { known++ }

	if known < 2 {
		t.Errorf("working memory: only knew %d/3 facts: name=%v location=%v language=%v text=%q",
			known, out.KnowsName, out.KnowsLocation, out.KnowsLanguage, out.Text)
	}
	t.Logf("working-memory: %q name=%v location=%v language=%v", out.Text, out.KnowsName, out.KnowsLocation, out.KnowsLanguage)
}

func TestFixture_TS_BusSubscribe(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/bus-subscribe.js")

	result, err := kit.EvalModule(context.Background(), "bus-subscribe.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		SubID  bool  `json:"subId"`
		Count  int   `json:"count"`
		Values []int `json:"values"`
		Topics []string `json:"topics"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.SubID {
		t.Error("no subscription ID returned")
	}
	if out.Count != 2 {
		t.Errorf("expected 2 messages, got %d (values: %v)", out.Count, out.Values)
	}
	t.Logf("bus-subscribe: subId=%v count=%d values=%v topics=%v", out.SubID, out.Count, out.Values, out.Topics)
}

func TestFixture_TS_ToolsRegisterList(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/tools-register-list.js")

	result, err := kit.EvalModule(context.Background(), "tools-register-list.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Registered bool     `json:"registered"`
		ToolCount  int      `json:"toolCount"`
		Found      bool     `json:"found"`
		Names      []string `json:"names"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Found {
		t.Errorf("registered tool not found in list: count=%d names=%v", out.ToolCount, out.Names)
	}
	t.Logf("tools-register-list: registered=%v found=%v count=%d names=%v", out.Registered, out.Found, out.ToolCount, out.Names)
}

func TestFixture_TS_ToolsCall(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Register the "uppercase" tool that the fixture expects
	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.uppercase", ShortName: "uppercase", Namespace: "platform",
		Description: "Converts text to uppercase",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ Text string }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]string{"result": strings.ToUpper(args.Text)})
				return result, nil
			},
		},
	})

	code := loadFixture(t, "testdata/ts/tools-call.js")
	result, err := kit.EvalModule(context.Background(), "tools-call.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct{ Result string }
	json.Unmarshal([]byte(result), &out)

	if out.Result != "HELLO BRAINLET" {
		t.Errorf("result = %q", out.Result)
	}
	t.Logf("fixture tools-call: %q", out.Result)
}

func TestFixture_TS_SandboxContext(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/sandbox-context.js")

	result, err := kit.EvalModule(context.Background(), "sandbox-context.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		ID        string `json:"id"`
		Namespace string `json:"namespace"`
		CallerID  string `json:"callerID"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.ID == "" {
		t.Error("empty id")
	}
	if out.Namespace != "test" {
		t.Errorf("namespace = %q", out.Namespace)
	}
	t.Logf("fixture sandbox-context: %+v", out)
}

// ═══════════════════════════════════════════════════════════════
// AS/WASM FIXTURES — compile and run AssemblyScript
// ═══════════════════════════════════════════════════════════════

func TestFixture_AS_Return42(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/return-42.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile
	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := kit.Bus.Request(ctx, "wasm.compile", kit.callerID, compilePayload)
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct{ ModuleID string `json:"moduleId"` }
	json.Unmarshal(compileResp.Payload, &compiled)

	// Run
	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := kit.Bus.Request(ctx, "wasm.run", kit.callerID, runPayload)
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 42 {
		t.Errorf("exitCode = %d, want 42", result.ExitCode)
	}
	t.Logf("fixture as/return-42: exitCode=%d", result.ExitCode)
}

func TestFixture_AS_Fibonacci(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/fibonacci.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := kit.Bus.Request(ctx, "wasm.compile", kit.callerID, compilePayload)
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct{ ModuleID string `json:"moduleId"` }
	json.Unmarshal(compileResp.Payload, &compiled)

	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := kit.Bus.Request(ctx, "wasm.run", kit.callerID, runPayload)
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 55 {
		t.Errorf("exitCode = %d, want 55 (fib(10))", result.ExitCode)
	}
	t.Logf("fixture as/fibonacci: fib(10)=%d", result.ExitCode)
}

func TestFixture_AS_Arithmetic(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/arithmetic.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := kit.Bus.Request(ctx, "wasm.compile", kit.callerID, compilePayload)
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct{ ModuleID string `json:"moduleId"` }
	json.Unmarshal(compileResp.Payload, &compiled)

	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := kit.Bus.Request(ctx, "wasm.run", kit.callerID, runPayload)
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 43 {
		t.Errorf("exitCode = %d, want 43 (add(multiply(6,7),1))", result.ExitCode)
	}
	t.Logf("fixture as/arithmetic: add(multiply(6,7),1)=%d", result.ExitCode)
}

// ═══════════════════════════════════════════════════════════════
// COMPOSITION FIXTURE — .ts uses everything including WASM
// ═══════════════════════════════════════════════════════════════

func TestFixture_TS_FullComposition(t *testing.T) {
	kit := newTestKit(t)

	// Register the "reverse" tool the fixture expects
	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.reverse", ShortName: "reverse", Namespace: "platform",
		Description: "Reverses a string",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ Text string }
				json.Unmarshal(input, &args)
				runes := []rune(args.Text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				result, _ := json.Marshal(map[string]string{"result": string(runes)})
				return result, nil
			},
		},
	})

	code := loadFixture(t, "testdata/ts/full-composition.js")
	result, err := kit.EvalModule(context.Background(), "full-composition.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Sandbox struct {
			Ns string `json:"ns"`
			Id string `json:"id"`
		} `json:"sandbox"`
		AIText       string `json:"aiText"`
		Reversed     string `json:"reversed"`
		HasLocalTool bool   `json:"hasLocalTool"`
		WasmExitCode int    `json:"wasmExitCode"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Sandbox.Ns != "test" {
		t.Errorf("namespace = %q", out.Sandbox.Ns)
	}
	if out.AIText == "" {
		t.Error("ai.generate returned empty text")
	}
	if out.Reversed != "telniarb" {
		t.Errorf("reversed = %q, want telniarb", out.Reversed)
	}
	if !out.HasLocalTool {
		t.Error("createTool failed")
	}
	if out.WasmExitCode != 99 {
		t.Errorf("wasm exitCode = %d, want 99", out.WasmExitCode)
	}
	t.Logf("fixture full-composition: ai=%q reversed=%q tool=%v wasm=%d",
		out.AIText, out.Reversed, out.HasLocalTool, out.WasmExitCode)
}
