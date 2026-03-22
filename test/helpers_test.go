package test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
)

// loadEnv reads the .env file from the project root and sets env vars.
func loadEnv(t *testing.T) {
	t.Helper()
	// Walk up from test/ to find .env
	envPath := filepath.Join("..", ".env")
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

// testKernel wraps a Kernel and exposes both sdk.Runtime and *Kernel for setup.
type testKernel struct {
	*kit.Kernel
}

// newTestKernelFull creates a Kernel with workspace, storage, AI provider, and a registered Go tool.
func newTestKernelFull(t *testing.T) *testKernel {
	t.Helper()
	loadEnv(t)
	tmpDir := t.TempDir()

	providers := make(map[string]kit.ProviderConfig)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers["openai"] = kit.ProviderConfig{APIKey: key}
	}

	storages := map[string]kit.StorageConfig{
		"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
	}

	// Pass provider API keys as env vars too — the embedded JS runtime's
	// ModelRouterEmbeddingModel reads from process.env.OPENAI_API_KEY etc.
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		envVars["OPENAI_API_KEY"] = key
	}

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-caller",
		WorkspaceDir: tmpDir,
		Providers:    providers,
		Storages:     storages,
		EnvVars:      envVars,
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	// Register test tools
	err = kit.RegisterTool(k, "echo", registry.TypedTool[echoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool echo: %v", err)
	}

	err = kit.RegisterTool(k, "add", registry.TypedTool[addInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input addInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool add: %v", err)
	}

	return &testKernel{k}
}

// newTestKernel creates a Kernel as sdk.Runtime.
func newTestKernel(t *testing.T) sdk.Runtime {
	t.Helper()
	return newTestKernelFull(t)
}

// newTestNode creates a Node with memory transport.
func newTestNode(t *testing.T) sdk.Runtime {
	t.Helper()
	loadEnv(t)
	tmpDir := t.TempDir()

	providers := make(map[string]kit.ProviderConfig)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers["openai"] = kit.ProviderConfig{APIKey: key}
	}

	n, err := kit.NewNode(kit.NodeConfig{
		Kernel: kit.KernelConfig{
			Namespace:    "test",
			CallerID:     "test-node",
			WorkspaceDir: tmpDir,
			Providers:    providers,
			Storages: map[string]kit.StorageConfig{
				"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
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
	kit.RegisterTool(n.Kernel, "echo", registry.TypedTool[echoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	kit.RegisterTool(n.Kernel, "add", registry.TypedTool[addInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input addInput) (any, error) {
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

// hasAIKey returns true if an AI provider key is available.
func hasAIKey() bool {
	return os.Getenv("OPENAI_API_KEY") != ""
}

// --- Test types ---

type echoInput struct {
	Message string `json:"message"`
}

type addInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
