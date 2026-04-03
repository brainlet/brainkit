package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/cmd/brainkit/cmd"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCLI creates a fresh command tree per test, sets args, captures output.
func runCLI(t *testing.T, configPath string, args ...string) (string, error) {
	t.Helper()
	root := cmd.NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	fullArgs := []string{}
	if configPath != "" {
		fullArgs = append(fullArgs, "--config", configPath)
	}
	fullArgs = append(fullArgs, args...)
	root.SetArgs(fullArgs)
	err := root.Execute()
	return strings.TrimSpace(buf.String()), err
}

// setupNodeWithControlAPI starts a Node with a control HTTP server (same as brainkit start).
// Returns the config file path pointing to the pidfile.
func setupNodeWithControlAPI(t *testing.T) string {
	t.Helper()
	testutil.LoadEnv(t)
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace: "test-cli",
			CallerID:  "test-cli-node",
			FSRoot:    tmpDir,
			Store:     store,
			SecretKey: "test-secret-key",
		},
		Messaging: brainkit.MessagingConfig{
			Transport: "memory",
		},
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))

	// Start a control API server (same as brainkit start does)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/bus", controlTestHandler(node))
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	// Write pidfile
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)
	pidFile := filepath.Join(dataDir, "brainkit.pid")
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", port)), 0644)

	// Write config that points to the data dir
	configPath := filepath.Join(tmpDir, "brainkit.yaml")
	os.WriteFile(configPath, []byte("namespace: test-cli\n"), 0644)

	// Deploy a simple echo service
	_, err = node.Kernel.Deploy(context.Background(), "echo-svc.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: msg.payload.value });
		});
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		srv.Shutdown(context.Background())
		node.Close()
	})

	// Change working dir so the CLI finds data/brainkit.pid
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	return configPath
}

// controlTestHandler is the same as the one in start.go — bus request-reply over HTTP.
func controlTestHandler(node *brainkit.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		ctx := r.Context()
		correlationID := uuid.NewString()
		replyTo := req.Topic + ".reply." + correlationID

		replyCh := make(chan messages.Message, 1)
		unsub, err := node.Kernel.SubscribeRaw(ctx, replyTo, func(msg messages.Message) {
			select {
			case replyCh <- msg:
			default:
			}
		})
		if err != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer unsub()

		pubCtx := messaging.WithPublishMeta(ctx, correlationID, replyTo)
		node.Kernel.PublishRaw(pubCtx, req.Topic, req.Payload)

		select {
		case msg := <-replyCh:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]json.RawMessage{"payload": msg.Payload})
		case <-ctx.Done():
			w.WriteHeader(504)
			json.NewEncoder(w).Encode(map[string]string{"error": "timeout"})
		}
	}
}

func TestCobra_Version(t *testing.T) {
	out, err := runCLI(t, "", "version")
	require.NoError(t, err)
	assert.Contains(t, out, "brainkit version")
}

func TestCobra_VersionJSON(t *testing.T) {
	out, err := runCLI(t, "", "--json", "version")
	require.NoError(t, err)
	assert.Contains(t, out, `"version"`)
}

func TestCobra_Health(t *testing.T) {
	configPath := setupNodeWithControlAPI(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "health")
	require.NoError(t, err)
	assert.Contains(t, out, "Status: running")
}

func TestCobra_HealthJSON(t *testing.T) {
	configPath := setupNodeWithControlAPI(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "--json", "health")
	require.NoError(t, err)
	assert.Contains(t, out, `"healthy"`)
}

func TestCobra_List(t *testing.T) {
	configPath := setupNodeWithControlAPI(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "echo-svc.ts")
}

func TestCobra_Eval(t *testing.T) {
	configPath := setupNodeWithControlAPI(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "eval", "output(1 + 1)")
	require.NoError(t, err)
	assert.Equal(t, "2", out)
}

func TestCobra_Send_RequestReply(t *testing.T) {
	configPath := setupNodeWithControlAPI(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "send", "echo-svc", "ping", `{"value":"from-test"}`)
	require.NoError(t, err)
	assert.Contains(t, out, "from-test")
}

func TestCobra_DeployAndTeardown(t *testing.T) {
	configPath := setupNodeWithControlAPI(t)
	tmpDir := t.TempDir()

	tsPath := filepath.Join(tmpDir, "test-deploy.ts")
	require.NoError(t, os.WriteFile(tsPath, []byte(`
		import { bus } from "kit";
		bus.on("hello", (msg) => msg.reply({ hi: true }));
	`), 0644))

	out, err := runCLI(t, configPath, "--timeout", "15s", "deploy", tsPath)
	require.NoError(t, err)
	assert.Contains(t, out, "Deployed test-deploy.ts")

	out, err = runCLI(t, configPath, "--timeout", "15s", "teardown", "test-deploy.ts")
	require.NoError(t, err)
	assert.Contains(t, out, "Removed")
}

func TestCobra_Secrets_CRUD(t *testing.T) {
	configPath := setupNodeWithControlAPI(t)

	out, err := runCLI(t, configPath, "--timeout", "15s", "secrets", "set", "TEST_KEY", "test-val")
	require.NoError(t, err)
	assert.Contains(t, out, "TEST_KEY")

	out, err = runCLI(t, configPath, "--timeout", "15s", "secrets", "get", "TEST_KEY")
	require.NoError(t, err)
	assert.Contains(t, out, "test-val")

	out, err = runCLI(t, configPath, "--timeout", "15s", "secrets", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "TEST_KEY")

	out, err = runCLI(t, configPath, "--timeout", "15s", "secrets", "delete", "TEST_KEY")
	require.NoError(t, err)
	assert.Contains(t, out, "deleted")
}

func TestCobra_NewModule(t *testing.T) {
	tmpDir := t.TempDir()
	modDir := filepath.Join(tmpDir, "my-mod")
	out, err := runCLI(t, "", "new", "module", "my-mod", "--dir", modDir)
	require.NoError(t, err)
	assert.Contains(t, out, "Created module my-mod")
	assert.FileExists(t, filepath.Join(modDir, "manifest.json"))
	assert.FileExists(t, filepath.Join(modDir, "types", "kit.d.ts"))
}

func TestCobra_NewPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)
	out, err := runCLI(t, "", "new", "plugin", "my-plug", "--owner", "testorg")
	require.NoError(t, err)
	assert.Contains(t, out, "Created plugin testorg/my-plug")
	assert.FileExists(t, filepath.Join(tmpDir, "my-plug", "main.go"))
}
