package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/cmd/brainkit/cmd"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := cmd.NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return strings.TrimSpace(buf.String()), err
}

func setupWorkDir(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()

	config := `namespace: test-cli
transport: sql-sqlite
sqlite_path: ./data/transport.db
storage:
  default:
    type: sqlite
    path: ./data/brainkit.db
fs_root: ./workspace
store_path: ./data/store.db
secret_key: test-secret-key
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "brainkit.yaml"), []byte(config), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(origDir) })
}

func startInstance(t *testing.T) {
	t.Helper()
	errCh := make(chan error, 1)
	go func() {
		root := cmd.NewRootCmd()
		root.SetArgs([]string{"start"})
		errCh <- root.Execute()
	}()

	pidFile := filepath.Join("data", "brainkit.pid")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(pidFile); err == nil {
			time.Sleep(200 * time.Millisecond)
			t.Cleanup(func() { os.Remove(pidFile) })
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("brainkit start did not create pidfile within 15s")
}

func testVersion(t *testing.T, _ *suite.TestEnv) {
	out, err := runCLI(t, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "brainkit version")
}

func testVersionJSON(t *testing.T, _ *suite.TestEnv) {
	out, err := runCLI(t, "--json", "version")
	require.NoError(t, err)
	assert.Contains(t, out, `"version"`)
}

func testInit(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	out, err := runCLI(t, "init")
	require.NoError(t, err)
	assert.Contains(t, out, "Created brainkit.yaml")
	assert.FileExists(t, filepath.Join(tmpDir, "brainkit.yaml"))
}

func testNewModule(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	modDir := filepath.Join(tmpDir, "my-mod")
	out, err := runCLI(t, "new", "module", "my-mod", "--dir", modDir)
	require.NoError(t, err)
	assert.Contains(t, out, "Created module my-mod")
	assert.FileExists(t, filepath.Join(modDir, "manifest.json"))
	assert.FileExists(t, filepath.Join(modDir, "hello.ts"))
	assert.FileExists(t, filepath.Join(modDir, "tsconfig.json"))
	assert.FileExists(t, filepath.Join(modDir, "types", "kit.d.ts"))

	data, _ := os.ReadFile(filepath.Join(modDir, "manifest.json"))
	assert.Contains(t, string(data), `"name": "my-mod"`)

	kitDts, _ := os.ReadFile(filepath.Join(modDir, "types", "kit.d.ts"))
	assert.Contains(t, string(kitDts), "BusMessage")
}

func testNewPlugin(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	out, err := runCLI(t, "new", "plugin", "my-plug", "--owner", "testorg")
	require.NoError(t, err)
	assert.Contains(t, out, "Created plugin testorg/my-plug")
	assert.FileExists(t, filepath.Join(tmpDir, "my-plug", "main.go"))

	mainGo, _ := os.ReadFile(filepath.Join(tmpDir, "my-plug", "main.go"))
	assert.Contains(t, string(mainGo), `"testorg"`)
}

func testFullWorkflow(t *testing.T, _ *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping full CLI workflow in short mode")
	}

	setupWorkDir(t)
	startInstance(t)

	out, err := runCLI(t, "--timeout", "15s", "health")
	require.NoError(t, err)
	assert.Contains(t, out, "Status: running")

	out, err = runCLI(t, "--timeout", "15s", "--json", "health")
	require.NoError(t, err)
	assert.Contains(t, out, `"healthy"`)

	tsFile := filepath.Join("workspace", "echo.ts")
	os.MkdirAll("workspace", 0755)
	os.WriteFile(tsFile, []byte(`
		import { bus } from "kit";
		bus.on("ping", (msg) => {
			msg.reply({ pong: msg.payload.value });
		});
	`), 0644)

	out, err = runCLI(t, "--timeout", "15s", "deploy", tsFile)
	require.NoError(t, err)
	assert.Contains(t, out, "Deployed echo.ts")

	out, err = runCLI(t, "--timeout", "15s", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "echo.ts")

	out, err = runCLI(t, "--timeout", "15s", "send", "echo", "ping", `{"value":"from-test"}`)
	require.NoError(t, err)
	assert.Contains(t, out, "from-test")

	out, err = runCLI(t, "--timeout", "15s", "eval", "output(1 + 1)")
	require.NoError(t, err)
	assert.Equal(t, "2", out)

	out, err = runCLI(t, "--timeout", "15s", "secrets", "set", "MY_KEY", "my-val")
	require.NoError(t, err)
	assert.Contains(t, out, "MY_KEY")

	out, err = runCLI(t, "--timeout", "15s", "secrets", "get", "MY_KEY")
	require.NoError(t, err)
	assert.Contains(t, out, "my-val")

	out, err = runCLI(t, "--timeout", "15s", "secrets", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "MY_KEY")

	out, err = runCLI(t, "--timeout", "15s", "secrets", "delete", "MY_KEY")
	require.NoError(t, err)
	assert.Contains(t, out, "deleted")

	out, err = runCLI(t, "--timeout", "15s", "teardown", "echo.ts")
	require.NoError(t, err)
	assert.Contains(t, out, "Removed")

	out, err = runCLI(t, "--timeout", "15s", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, "echo.ts")
}

func testSendWithAsyncHandler(t *testing.T, _ *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping async handler test in short mode")
	}

	setupWorkDir(t)
	startInstance(t)

	tsFile := filepath.Join("workspace", "slow.ts")
	os.MkdirAll("workspace", 0755)
	os.WriteFile(tsFile, []byte(`
		import { bus } from "kit";
		bus.on("compute", async (msg) => {
			const result = await new Promise((resolve) => {
				setTimeout(() => resolve(msg.payload.a + msg.payload.b), 500);
			});
			msg.reply({ sum: result });
		});
	`), 0644)

	out, err := runCLI(t, "--timeout", "15s", "deploy", tsFile)
	require.NoError(t, err)
	assert.Contains(t, out, "Deployed slow.ts")

	out, err = runCLI(t, "--timeout", "15s", "send", "slow", "compute", `{"a":17,"b":25}`)
	require.NoError(t, err)
	assert.Contains(t, out, "42")
}
