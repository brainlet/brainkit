package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/cmd/brainkit/cmd"
	"github.com/brainlet/brainkit/internal/testutil"
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

func setupNodeWithConfig(t *testing.T) string {
	t.Helper()
	testutil.LoadEnv(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "transport.db")
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
			Transport:  "sql-sqlite",
			SQLitePath: dbPath,
		},
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))
	t.Cleanup(func() { node.Close() })

	_, err = node.Kernel.Deploy(context.Background(), "echo-svc.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: msg.payload.value });
		});
	`)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, "brainkit.yaml")
	config := "namespace: test-cli\ntransport: sql-sqlite\nsqlite_path: " + dbPath + "\nstore_path: " + storePath + "\nfs_root: " + tmpDir + "\nsecret_key: test-secret-key\n"
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))
	return configPath
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
	configPath := setupNodeWithConfig(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "health")
	require.NoError(t, err)
	assert.Contains(t, out, "Status: running")
}

func TestCobra_HealthJSON(t *testing.T) {
	configPath := setupNodeWithConfig(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "--json", "health")
	require.NoError(t, err)
	var resp struct {
		Health json.RawMessage `json:"health"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &resp))
	assert.NotEmpty(t, resp.Health)
}

func TestCobra_List(t *testing.T) {
	configPath := setupNodeWithConfig(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "echo-svc.ts")
}

func TestCobra_Eval(t *testing.T) {
	configPath := setupNodeWithConfig(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "eval", "output(1 + 1)")
	require.NoError(t, err)
	assert.Equal(t, "2", out)
}

func TestCobra_Send_RequestReply(t *testing.T) {
	configPath := setupNodeWithConfig(t)
	out, err := runCLI(t, configPath, "--timeout", "15s", "send", "echo-svc", "ping", `{"value":"from-test"}`)
	require.NoError(t, err)
	assert.Contains(t, out, "from-test")
}

func TestCobra_DeployAndTeardown(t *testing.T) {
	configPath := setupNodeWithConfig(t)
	tmpDir := t.TempDir()

	tsPath := filepath.Join(tmpDir, "test-deploy.ts")
	require.NoError(t, os.WriteFile(tsPath, []byte(`
		import { bus } from "kit";
		bus.on("hello", (msg) => msg.reply({ hi: true }));
	`), 0644))

	out, err := runCLI(t, configPath, "--timeout", "15s", "deploy", tsPath)
	require.NoError(t, err)
	assert.Contains(t, out, "Deployed test-deploy.ts")

	out, err = runCLI(t, configPath, "--timeout", "15s", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "test-deploy.ts")

	out, err = runCLI(t, configPath, "--timeout", "15s", "teardown", "test-deploy.ts")
	require.NoError(t, err)
	assert.Contains(t, out, "Removed")
}

func TestCobra_Secrets_CRUD(t *testing.T) {
	configPath := setupNodeWithConfig(t)

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
	assert.FileExists(t, filepath.Join(modDir, "hello.ts"))
	assert.FileExists(t, filepath.Join(modDir, "types", "kit.d.ts"))

	data, err := os.ReadFile(filepath.Join(modDir, "manifest.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"name": "my-mod"`)
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
