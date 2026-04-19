package testutil

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/modules/schedules"
	"github.com/brainlet/brainkit/sdk"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestKit wraps a Kit for test convenience.
type TestKit struct {
	*brainkit.Kit
}

// EchoInput is the input type for the echo test tool.
type EchoInput struct {
	Message string `json:"message"`
}

// AddInput is the input type for the add test tool.
type AddInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

// projectRoot walks up from the current working directory to find the project root
// (the directory containing go.mod).
func projectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LoadEnv reads the .env file from the project root and sets env vars.
func LoadEnv(t *testing.T) {
	t.Helper()
	root := projectRoot()
	if root == "" {
		return
	}
	envPath := filepath.Join(root, ".env")
	f, err := os.Open(envPath)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
}

// NewTestKitFull creates a Kit with workspace, storage, AI provider, and registered Go tools.
func NewTestKitFull(t *testing.T) *TestKit {
	t.Helper()
	LoadEnv(t)
	tmpDir := t.TempDir()

	var providers []brainkit.ProviderConfig
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, brainkit.OpenAI(key))
		envVars["OPENAI_API_KEY"] = key
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test",
		CallerID:  "test-caller",
		FSRoot:    tmpDir,
		Providers: providers,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
			"mem":     brainkit.InMemoryStorage(),
		},
		Vectors: map[string]brainkit.VectorConfig{
			"default": brainkit.SQLiteVector(filepath.Join(tmpDir, "brainkit.db")),
		},
		// In-memory schedules module so fixtures can exercise
		// bus.schedule / bus.unschedule without a persistence backend.
		Modules: []brainkit.Module{schedules.NewModule(schedules.Config{})},
		EnvVars: envVars,
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })

	err = brainkit.RegisterTool(kit, "echo", tools.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool echo: %v", err)
	}

	err = brainkit.RegisterTool(kit, "add", tools.TypedTool[AddInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool add: %v", err)
	}

	return &TestKit{kit}
}

// NewTestKit creates a Kit as sdk.Runtime.
func NewTestKit(t *testing.T) sdk.Runtime {
	t.Helper()
	return NewTestKitFull(t)
}

// NewTestNode creates a Kit with memory transport.
func NewTestNode(t *testing.T) sdk.Runtime {
	t.Helper()
	LoadEnv(t)
	tmpDir := t.TempDir()

	var providers []brainkit.ProviderConfig
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, brainkit.OpenAI(key))
		envVars["OPENAI_API_KEY"] = key
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test",
		CallerID:  "test-node",
		FSRoot:    tmpDir,
		Transport: brainkit.Memory(),
		Providers: providers,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
		},
		EnvVars: envVars,
	})
	if err != nil {
		t.Fatalf("brainkit.New (node): %v", err)
	}

	brainkit.RegisterTool(kit, "echo", tools.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	brainkit.RegisterTool(kit, "add", tools.TypedTool[AddInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	t.Cleanup(func() { kit.Close() })
	return kit
}

// HasAIKey returns true if an AI provider key is available.
func HasAIKey() bool {
	return os.Getenv("OPENAI_API_KEY") != ""
}

// BuildTestMCP compiles the testmcp binary and returns its path.
func BuildTestMCP(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "testmcp")
	root := projectRoot()
	cmd := exec.Command("go", "build", "-o", binary, "./test/testmcp/")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build testmcp: %v", err)
	}
	return binary
}

// StartPgVectorContainer starts a pgvector container and returns the connection string.
func StartPgVectorContainer(t *testing.T) string {
	t.Helper()
	addr := StartContainer(t,
		"pgvector/pgvector:pg16",
		"5432/tcp",
		nil,
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=test",
		"POSTGRES_PASSWORD=test",
		"POSTGRES_DB=brainkit",
	)
	return "postgresql://test:test@" + addr + "/brainkit"
}

// ConcurrentDo runs fn in n goroutines and waits for all to complete.
func ConcurrentDo(t *testing.T, n int, fn func(i int)) {
	t.Helper()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("goroutine %d panicked: %v", idx, r)
				}
			}()
			fn(idx)
		}(i)
	}
	wg.Wait()
}

// WaitForBusMessage subscribes to a topic, waits for one message, returns it.
func WaitForBusMessage(t *testing.T, rt sdk.Runtime, topic string, timeout time.Duration) sdk.Message {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan sdk.Message, 1)
	unsub, err := rt.SubscribeRaw(ctx, topic, func(msg sdk.Message) {
		select {
		case ch <- msg:
		default:
		}
	})
	if err != nil {
		t.Fatalf("WaitForBusMessage: subscribe %s: %v", topic, err)
	}
	defer unsub()

	select {
	case msg := <-ch:
		return msg
	case <-ctx.Done():
		t.Fatalf("WaitForBusMessage: timeout waiting for message on %s", topic)
		return sdk.Message{}
	}
}
