package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit"
	provreg "github.com/brainlet/brainkit/registry"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PodmanAvailable returns true if Podman is installed AND the machine is running.
// `podman info` returns exit 0 even when the machine is stopped (it reports client info).
// We need to actually verify the runtime is reachable by running a real container operation.
func PodmanAvailable() bool {
	if _, err := exec.LookPath("podman"); err != nil {
		return false
	}
	// `podman ps` requires a running machine — fails if stopped or missing
	err := exec.Command("podman", "ps", "--noheading").Run()
	return err == nil
}

// AllBackends returns backends available for testing.
// GoChannel and SQLite always included. Others require Podman.
func AllBackends(t *testing.T) []string {
	backends := []string{"memory", "sql-sqlite"}
	if PodmanAvailable() {
		CleanupOrphanedContainers(t)
		backends = append(backends, "nats", "amqp", "redis", "sql-postgres")
	} else {
		t.Log("Podman not available — skipping NATS, AMQP, Redis, Postgres backends")
	}
	return backends
}

// MustCreateTransport creates a transport or fails the test.
func MustCreateTransport(t *testing.T, cfg transport.TransportConfig) *transport.Transport {
	t.Helper()
	transport, err := transport.NewTransportSet(cfg)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	// Don't cleanup transport here — Kernel owns it via ownsTransport=false
	return transport
}

// TransportConfigForBackend returns a TransportConfig for the given backend.
// For Podman-based backends, starts the container and returns the URL.
func TransportConfigForBackend(t *testing.T, backend string) transport.TransportConfig {
	t.Helper()
	switch backend {
	case "memory", "":
		return transport.TransportConfig{Type: "memory"}
	case "sql-sqlite":
		return transport.TransportConfig{
			Type:       "sql-sqlite",
			SQLitePath: filepath.Join(t.TempDir(), "transport.db"),
		}
	case "nats":
		url := StartContainer(t, "nats:latest", "4222/tcp", []string{"-js"},
			wait.ForLog("Server is ready").WithStartupTimeout(30*time.Second))
		return transport.TransportConfig{Type: "nats", NATSURL: url, NATSName: "test"}
	case "amqp":
		url := StartContainer(t, "rabbitmq:management", "5672/tcp", nil,
			wait.ForLog("Ready to start client connection listeners").WithStartupTimeout(60*time.Second))
		return transport.TransportConfig{Type: "amqp", AMQPURL: fmt.Sprintf("amqp://guest:guest@%s/", url)}
	case "redis":
		url := StartContainer(t, "redis:latest", "6379/tcp", nil,
			wait.ForLog("Ready to accept connections").WithStartupTimeout(30*time.Second))
		return transport.TransportConfig{Type: "redis", RedisURL: fmt.Sprintf("redis://%s/0", url)}
	case "sql-postgres":
		url := StartContainer(t, "postgres:16", "5432/tcp", nil,
			wait.ForLog("database system is ready to accept connections").WithStartupTimeout(60*time.Second),
			"POSTGRES_USER=test", "POSTGRES_PASSWORD=test", "POSTGRES_DB=brainkit",
		)
		return transport.TransportConfig{
			Type:        "sql-postgres",
			PostgresURL: fmt.Sprintf("postgres://test:test@%s/brainkit?sslmode=disable", url),
		}
	default:
		t.Fatalf("unknown backend: %s", backend)
		return transport.TransportConfig{}
	}
}

// CleanupOrphanedContainers removes leftover test containers from previous runs.
// Call this before tests that need Podman containers — prevents zombie accumulation
// that overloads the Podman VM and causes container startup timeouts.
func CleanupOrphanedContainers(t *testing.T) {
	t.Helper()
	if !PodmanAvailable() {
		return
	}
	// Kill containers from common test images that are older than 10 minutes
	images := []string{"nats", "rabbitmq", "redis", "postgres", "pgvector"}
	for _, img := range images {
		out, _ := exec.Command("podman", "ps", "-q", "--filter", "ancestor=*"+img+"*").Output()
		ids := strings.TrimSpace(string(out))
		if ids != "" {
			for _, id := range strings.Split(ids, "\n") {
				id = strings.TrimSpace(id)
				if id != "" {
					exec.Command("podman", "rm", "-f", id).Run()
				}
			}
		}
	}
}

// StartContainer starts a Podman container and returns "host:port".
func StartContainer(t *testing.T, image, port string, cmd []string, strategy wait.Strategy, envVars ...string) string {
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

// WaitForBackendReady verifies the transport is fully operational by publishing
// a probe message and waiting for it to round-trip. Retries up to 3 times with
// increasing delay for slow backends (SQL table creation, AMQP queue binding).
func WaitForBackendReady(t *testing.T, transport *transport.Transport) {
	t.Helper()

	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		probeTopic := fmt.Sprintf("probe_%d_%d", time.Now().UnixNano(), attempt)
		if transport.TopicSanitizer != nil {
			probeTopic = transport.TopicSanitizer(probeTopic)
		}

		ch, err := transport.Subscriber.Subscribe(ctx, probeTopic)
		if err != nil {
			t.Logf("probe attempt %d: subscribe failed: %v", attempt, err)
			cancel()
			continue
		}

		msg := message.NewMessage(watermill.NewUUID(), []byte(`{"probe":true}`))
		if err := transport.Publisher.Publish(probeTopic, msg); err != nil {
			t.Logf("probe attempt %d: publish failed: %v", attempt, err)
			cancel()
			continue
		}

		ok := false
		select {
		case wmsg, recv := <-ch:
			if recv {
				wmsg.Ack()
				ok = true
			}
		case <-ctx.Done():
		}
		cancel()

		if ok {
			return
		}
	}
	t.Fatalf("backend not ready after 5 probe attempts — transport is broken or container didn't start")
}

// RequiresNetworkTransport skips the test if the backend is memory (in-process only).
func RequiresNetworkTransport(t *testing.T, backend string) {
	t.Helper()
	if backend == "memory" || backend == "" {
		t.Skip("plugin subprocess tests require network transport (not memory)")
	}
}

// NewTestKernelFullWithBackend creates a fully configured Kernel on the given transport backend.
// Probes the transport for readiness before creating the Kernel — prevents hangs on slow
// backends (AMQP queue binding, SQL table creation, etc.).
func NewTestKernelFullWithBackend(t *testing.T, backend string) *TestKernel {
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

	cfg := TransportConfigForBackend(t, backend)
	transport := MustCreateTransport(t, cfg)
	t.Cleanup(func() { transport.Close() })

	// Probe transport readiness — ensures round-trip pub/sub works before
	// creating the Kernel. SQL backends need table creation, AMQP needs
	// queue binding, all of which may take time after container start.
	WaitForBackendReady(t, transport)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:   "test",
		CallerID:    "test-" + backend,
		FSRoot:      tmpDir,
		AIProviders: aiProviders,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
		},
		EnvVars:   envVars,
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("NewKernel(%s): %v", backend, err)
	}
	t.Cleanup(func() { k.Close() })

	brainkit.RegisterTool(k, "echo", registry.TypedTool[EchoInput]{
		Description: "echoes the input message",
		Execute: func(ctx context.Context, input EchoInput) (any, error) {
			return map[string]string{"echoed": input.Message}, nil
		},
	})
	brainkit.RegisterTool(k, "add", registry.TypedTool[AddInput]{
		Description: "adds two numbers",
		Execute: func(ctx context.Context, input AddInput) (any, error) {
			return map[string]int{"sum": input.A + input.B}, nil
		},
	})

	return &TestKernel{k}
}

// BuildTestPlugin compiles the testplugin binary and returns its path.
func BuildTestPlugin(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "testplugin")
	root := projectRoot()
	cmd := exec.Command("go", "build", "-o", binary, "./test/testplugin/")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build test plugin: %v", err)
	}
	return binary
}
