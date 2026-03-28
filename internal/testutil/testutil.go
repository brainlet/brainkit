package testutil

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit"
	provreg "github.com/brainlet/brainkit/kit/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestKernel wraps a Kernel and exposes both sdk.Runtime and *Kernel for setup.
type TestKernel struct {
	*kit.Kernel
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

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:   "test",
		CallerID:    "test-caller",
		FSRoot:      tmpDir,
		AIProviders: aiProviders,
		Storages: map[string]kit.StorageConfig{
			"default": kit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
		},
		EnvVars: envVars,
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	// Register test tools
	err = kit.RegisterTool(k, "echo", registry.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool echo: %v", err)
	}

	err = kit.RegisterTool(k, "add", registry.TypedTool[AddInput]{
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

	n, err := kit.NewNode(kit.NodeConfig{
		Kernel: kit.KernelConfig{
			Namespace:   "test",
			CallerID:    "test-node",
			FSRoot:      tmpDir,
			AIProviders: nodeProviders,
			EnvVars:     nodeEnvVars,
			Storages: map[string]kit.StorageConfig{
				"default": kit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
			},
		},
		Messaging: kit.MessagingConfig{
			Transport: "memory",
		},
	})
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}

	// Register test tools on Node's kernel
	kit.RegisterTool(n.Kernel, "echo", registry.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	kit.RegisterTool(n.Kernel, "add", registry.TypedTool[AddInput]{
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

// MustJSON marshals v to json.RawMessage or panics.
func MustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// NewTestKernelWithStorage creates a Kernel with storage, workspace, and AI providers.
// Used for memory, workflows, and vectors domain tests that need JS runtime storage init.
func NewTestKernelWithStorage(t *testing.T) *TestKernel {
	t.Helper()
	LoadEnv(t)
	tmpDir := t.TempDir()

	storageProviders := make(map[string]provreg.AIProviderRegistration)
	storageEnvVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		storageProviders["openai"] = provreg.AIProviderRegistration{
			Type:   provreg.AIProviderOpenAI,
			Config: provreg.OpenAIProviderConfig{APIKey: key},
		}
		storageEnvVars["OPENAI_API_KEY"] = key
	}

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:   "test",
		CallerID:    "test-storage",
		FSRoot:      tmpDir,
		AIProviders: storageProviders,
		Storages: map[string]kit.StorageConfig{
			"default": kit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
		},
		EnvVars: storageEnvVars,
	})
	if err != nil {
		t.Fatalf("NewKernel (with storage): %v", err)
	}
	t.Cleanup(func() { k.Close() })

	// Register standard test tools
	kit.RegisterTool(k, "echo", registry.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	kit.RegisterTool(k, "add", registry.TypedTool[AddInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	return &TestKernel{k}
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
