package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit"
	provreg "github.com/brainlet/brainkit/kit/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// podmanAvailable returns true if Podman is installed and running.
func podmanAvailable() bool {
	if _, err := exec.LookPath("podman"); err != nil {
		return false
	}
	out, err := exec.Command("podman", "info").CombinedOutput()
	return err == nil && len(out) > 0
}

// allBackends returns backends available for testing.
// GoChannel and SQLite always included. Others require Podman.
func allBackends(t *testing.T) []string {
	backends := []string{"memory", "sql-sqlite"}
	if podmanAvailable() {
		backends = append(backends, "nats", "amqp", "redis", "sql-postgres")
	} else {
		t.Log("Podman not available — skipping NATS, AMQP, Redis, Postgres backends")
	}
	return backends
}

// createTestTransport creates a transport for the given backend.
// For Podman-based backends, starts the container and returns URL.
func createTestTransport(t *testing.T, backend string) *messaging.Transport {
	t.Helper()
	cfg := transportConfigForBackend(t, backend)
	transport, err := messaging.NewTransportSet(cfg)
	if err != nil {
		t.Fatalf("create transport %s: %v", backend, err)
	}
	t.Cleanup(func() { transport.Close() })
	return transport
}

// newTestKernelWithTransport creates a Kernel with the given backend transport.
func newTestKernelWithTransport(t *testing.T, backend string) sdk.Runtime {
	t.Helper()
	loadEnv(t)
	tmpDir := t.TempDir()
	cfg := transportConfigForBackend(t, backend)

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-" + backend,
		WorkspaceDir: tmpDir,
		Transport:    mustCreateTransport(t, cfg),
	})
	if err != nil {
		t.Fatalf("NewKernel(%s): %v", backend, err)
	}
	t.Cleanup(func() { k.Close() })

	// Register test tools
	kit.RegisterTool(k, "echo", registry.TypedTool[echoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	kit.RegisterTool(k, "add", registry.TypedTool[addInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input addInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	return k
}

// newKitWithNamespace creates a Kernel with a specific namespace on the given backend.
// For cross-Kit tests, two Kits share the same transport but different namespaces.
func newKitWithNamespace(t *testing.T, namespace, backend string) sdk.Runtime {
	t.Helper()
	loadEnv(t)
	tmpDir := t.TempDir()
	cfg := transportConfigForBackend(t, backend)

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    namespace,
		CallerID:     namespace + "-caller",
		WorkspaceDir: tmpDir,
		Transport:    mustCreateTransport(t, cfg),
	})
	if err != nil {
		t.Fatalf("NewKernel(%s, ns=%s): %v", backend, namespace, err)
	}
	t.Cleanup(func() { k.Close() })

	kit.RegisterTool(k, "echo", registry.TypedTool[echoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message, "from": namespace}, nil
		},
	})

	return k
}

// mustCreateTransport creates a transport or fails the test.
func mustCreateTransport(t *testing.T, cfg messaging.TransportConfig) *messaging.Transport {
	t.Helper()
	transport, err := messaging.NewTransportSet(cfg)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	// Don't cleanup transport here — Kernel owns it via ownsTransport=false
	return transport
}

// transportConfigForBackend returns a TransportConfig for the given backend.
// For Podman-based backends, starts the container and returns the URL.
func transportConfigForBackend(t *testing.T, backend string) messaging.TransportConfig {
	t.Helper()
	switch backend {
	case "memory", "":
		return messaging.TransportConfig{Type: "memory"}
	case "sql-sqlite":
		return messaging.TransportConfig{
			Type:       "sql-sqlite",
			SQLitePath: filepath.Join(t.TempDir(), "transport.db"),
		}
	case "nats":
		url := startContainer(t, "nats:latest", "4222/tcp", []string{"-js"},
			wait.ForLog("Server is ready").WithStartupTimeout(30*time.Second))
		return messaging.TransportConfig{Type: "nats", NATSURL: url, NATSName: "test"}
	case "amqp":
		url := startContainer(t, "rabbitmq:management", "5672/tcp", nil,
			wait.ForLog("Ready to start client connection listeners").WithStartupTimeout(60*time.Second))
		return messaging.TransportConfig{Type: "amqp", AMQPURL: fmt.Sprintf("amqp://guest:guest@%s/", url)}
	case "redis":
		url := startContainer(t, "redis:latest", "6379/tcp", nil,
			wait.ForLog("Ready to accept connections").WithStartupTimeout(30*time.Second))
		return messaging.TransportConfig{Type: "redis", RedisURL: fmt.Sprintf("redis://%s/0", url)}
	case "sql-postgres":
		url := startContainer(t, "postgres:16", "5432/tcp", nil,
			wait.ForLog("database system is ready to accept connections").WithStartupTimeout(60*time.Second),
			"POSTGRES_USER=test", "POSTGRES_PASSWORD=test", "POSTGRES_DB=brainkit",
		)
		return messaging.TransportConfig{
			Type:        "sql-postgres",
			PostgresURL: fmt.Sprintf("postgres://test:test@%s/brainkit?sslmode=disable", url),
		}
	default:
		t.Fatalf("unknown backend: %s", backend)
		return messaging.TransportConfig{}
	}
}

// startContainer starts a Podman container and returns "host:port".
func startContainer(t *testing.T, image, port string, cmd []string, strategy wait.Strategy, envVars ...string) string {
	t.Helper()

	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	if os.Getenv("DOCKER_HOST") == "" {
		if out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output(); err == nil {
			socketPath := string(out[:len(out)-1])
			os.Setenv("DOCKER_HOST", "unix://"+socketPath)
		}
	}

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{port},
		WaitingFor:   strategy,
	}
	if len(cmd) > 0 {
		req.Cmd = cmd
	}
	if len(envVars) > 0 {
		req.Env = make(map[string]string)
		for _, ev := range envVars {
			parts := splitEnvVar(ev)
			req.Env[parts[0]] = parts[1]
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("failed to start %s container: %v", image, err)
		return ""
	}
	t.Cleanup(func() { container.Terminate(context.Background()) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	mappedPort, err := container.MappedPort(ctx, nat.Port(port))
	if err != nil {
		t.Fatalf("container port: %v", err)
	}

	return fmt.Sprintf("%s:%s", host, mappedPort.Port())
}

func splitEnvVar(ev string) [2]string {
	for i, c := range ev {
		if c == '=' {
			return [2]string{ev[:i], ev[i+1:]}
		}
	}
	return [2]string{ev, ""}
}

// waitForBackendReady verifies the transport is fully operational by publishing
// a probe message and waiting for it to round-trip. This catches cases where the
// container is up but Watermill isn't fully connected (e.g., NATS JetStream provisioning).
func waitForBackendReady(t *testing.T, transport *messaging.Transport) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	probeTopic := "probe-" + fmt.Sprintf("%d", time.Now().UnixNano())
	if transport.TopicSanitizer != nil {
		probeTopic = transport.TopicSanitizer(probeTopic)
	}

	ch, err := transport.Subscriber.Subscribe(ctx, probeTopic)
	if err != nil {
		t.Skipf("backend not ready (subscribe failed): %v", err)
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), []byte(`{"probe":true}`))
	if err := transport.Publisher.Publish(probeTopic, msg); err != nil {
		t.Skipf("backend not ready (publish failed): %v", err)
		return
	}

	select {
	case wmsg, ok := <-ch:
		if ok {
			wmsg.Ack()
		}
	case <-ctx.Done():
		t.Skipf("backend not ready (probe timeout after 30s)")
	}
}

// newTestKernelPair creates two Kernels on the SAME transport with different namespaces.
// Both Kits share one transport instance so cross-namespace messages can route between them.
func newTestKernelPair(t *testing.T, backend string) (sdk.Runtime, sdk.Runtime) {
	t.Helper()
	loadEnv(t)
	cfg := transportConfigForBackend(t, backend)
	transport := mustCreateTransport(t, cfg)
	t.Cleanup(func() { transport.Close() })

	makeKit := func(namespace string) sdk.Runtime {
		tmpDir := t.TempDir()
		k, err := kit.NewKernel(kit.KernelConfig{
			Namespace:    namespace,
			CallerID:     namespace + "-caller",
			WorkspaceDir: tmpDir,
			Transport:    transport,
		})
		if err != nil {
			t.Fatalf("NewKernel(%s, ns=%s): %v", backend, namespace, err)
		}
		t.Cleanup(func() { k.Close() })

		kit.RegisterTool(k, "echo", registry.TypedTool[echoInput]{
			Description: "echoes the input message",
			Execute: func(ctx context.Context, input echoInput) (any, error) {
				return map[string]string{"echoed": input.Message, "from": namespace}, nil
			},
		})
		return k
	}

	return makeKit("kit-a"), makeKit("kit-b")
}

// requiresNetworkTransport skips the test if the backend is memory (in-process only).
// Plugin subprocess tests cannot use GoChannel memory transport.
func requiresNetworkTransport(t *testing.T, backend string) {
	t.Helper()
	if backend == "memory" || backend == "" {
		t.Skip("plugin subprocess tests require network transport (not memory)")
	}
}

// newTestKernelFullWithBackend creates a fully configured Kernel (workspace, storage,
// AI providers, tools) on the given transport backend. This is the backend-parameterized
// version of newTestKernelFull — use it in tests that loop over allBackends().
func newTestKernelFullWithBackend(t *testing.T, backend string) *testKernel {
	t.Helper()
	loadEnv(t)
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

	cfg := transportConfigForBackend(t, backend)
	transport := mustCreateTransport(t, cfg)
	t.Cleanup(func() { transport.Close() })

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-" + backend,
		WorkspaceDir: tmpDir,
		AIProviders:  aiProviders,
		EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{
			"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
		},
		EnvVars:   envVars,
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("NewKernel(%s): %v", backend, err)
	}
	t.Cleanup(func() { k.Close() })

	kit.RegisterTool(k, "echo", registry.TypedTool[echoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	kit.RegisterTool(k, "add", registry.TypedTool[addInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input addInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	return &testKernel{k}
}

// newTestKernelWithStorageAndBackend creates a Kernel with storage + workspace + AI providers
// on the given transport backend. Backend-parameterized version of newTestKernelWithStorage.
func newTestKernelWithStorageAndBackend(t *testing.T, backend string) *testKernel {
	t.Helper()
	loadEnv(t)
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

	cfg := transportConfigForBackend(t, backend)
	transport := mustCreateTransport(t, cfg)
	t.Cleanup(func() { transport.Close() })

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-storage-" + backend,
		WorkspaceDir: tmpDir,
		AIProviders:  storageProviders,
		EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{
			"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
		},
		EnvVars:   storageEnvVars,
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("NewKernel(%s, storage): %v", backend, err)
	}
	t.Cleanup(func() { k.Close() })

	kit.RegisterTool(k, "echo", registry.TypedTool[echoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input echoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	kit.RegisterTool(k, "add", registry.TypedTool[addInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input addInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	return &testKernel{k}
}
