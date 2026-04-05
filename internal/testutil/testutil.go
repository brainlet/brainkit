package testutil

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit"
	provreg "github.com/brainlet/brainkit/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestKernel wraps a Kernel and exposes both sdk.Runtime and *Kernel for setup.
type TestKernel struct {
	*brainkit.Kernel
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
		return // no .env, that's fine
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

// NewTestKernelFull creates a Kernel with workspace, storage, AI provider, and a registered Go tool.
func NewTestKernelFull(t *testing.T) *TestKernel {
	t.Helper()
	LoadEnv(t)
	tmpDir := t.TempDir()

	aiProviders := make(map[string]provreg.AIProviderRegistration)
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		aiProviders["openai"] = provreg.AIProviderRegistration{
			Type:   provreg.AIProviderOpenAI,
			Config: provreg.OpenAIProviderConfig{APIKey: key},
		}
		envVars["OPENAI_API_KEY"] = key
	}

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:   "test",
		CallerID:    "test-caller",
		FSRoot:      tmpDir,
		AIProviders: aiProviders,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
			"mem":     brainkit.InMemoryStorage(),
		},
		Vectors: map[string]brainkit.VectorConfig{
			"default": brainkit.SQLiteVector(filepath.Join(tmpDir, "brainkit.db")),
		},
		EnvVars: envVars,
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	// Register test tools
	err = brainkit.RegisterTool(k, "echo", registry.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool echo: %v", err)
	}

	err = brainkit.RegisterTool(k, "add", registry.TypedTool[AddInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool add: %v", err)
	}

	return &TestKernel{k}
}

// NewTestKernel creates a Kernel as sdk.Runtime.
func NewTestKernel(t *testing.T) sdk.Runtime {
	t.Helper()
	return NewTestKernelFull(t)
}

// NewTestNode creates a Node with memory transport.
func NewTestNode(t *testing.T) sdk.Runtime {
	t.Helper()
	LoadEnv(t)
	tmpDir := t.TempDir()

	nodeProviders := make(map[string]provreg.AIProviderRegistration)
	nodeEnvVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		nodeProviders["openai"] = provreg.AIProviderRegistration{
			Type:   provreg.AIProviderOpenAI,
			Config: provreg.OpenAIProviderConfig{APIKey: key},
		}
		nodeEnvVars["OPENAI_API_KEY"] = key
	}

	n, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace:   "test",
			CallerID:    "test-node",
			FSRoot:      tmpDir,
			AIProviders: nodeProviders,
			EnvVars:     nodeEnvVars,
			Storages: map[string]brainkit.StorageConfig{
				"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
			},
		},
		Messaging: brainkit.MessagingConfig{
			Transport: "memory",
		},
	})
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}

	// Register test tools on Node's kernel
	brainkit.RegisterTool(n.Kernel, "echo", registry.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	brainkit.RegisterTool(n.Kernel, "add", registry.TypedTool[AddInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	if err := n.Start(context.Background()); err != nil {
		n.Close()
		t.Fatalf("Node.Start: %v", err)
	}
	t.Cleanup(func() { n.Close() })
	return n
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
	return fmt.Sprintf("postgresql://test:test@%s/brainkit", addr)
}

// ConcurrentDo runs fn in n goroutines and waits for all to complete.
// Captures panics and reports which goroutine (by index) failed.
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

// WaitForBusMessage subscribes to a topic, waits for one message, unsubscribes, returns it.
// Fails with timeout if no message arrives within the deadline.
func WaitForBusMessage(t *testing.T, k *brainkit.Kernel, topic string, timeout time.Duration) messages.Message {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan messages.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, topic, func(msg messages.Message) {
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
		return messages.Message{} // unreachable
	}
}

