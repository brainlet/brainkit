package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/cmd/brainkit/cmd"
	bkgw "github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCLI invokes the brainkit root command with args, captures
// stdout+stderr, and returns the combined output + any error.
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

// runCLICtx is like runCLI but attaches a context so long-running
// commands can be cancelled by the test's deadline.
func runCLICtx(t *testing.T, ctx context.Context, args ...string) (string, error) {
	t.Helper()
	root := cmd.NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	root.SetContext(ctx)
	err := root.Execute()
	return strings.TrimSpace(buf.String()), err
}

// startTestServer boots a Kit with the gateway module on a random
// port and returns the addr + a cleanup fn. The gateway registers
// /api/bus by default, which is what the new CLI verbs talk to.
func startTestServer(t *testing.T) (addr string) {
	t.Helper()

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	listenAddr := probe.Addr().String()
	_ = probe.Close()

	gw := bkgw.New(bkgw.Config{Listen: listenAddr, Timeout: 5 * time.Second})

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "cli-test",
		Transport: brainkit.Memory(),
		FSRoot:    t.TempDir(),
		Modules:   []brainkit.Module{gw},
	})
	require.NoError(t, err)
	t.Cleanup(func() { kit.Close() })

	// Poll the built-in /healthz until it responds.
	base := "http://" + listenAddr
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("test server did not become healthy at %s", base)
	return base
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

func testNewPackage(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "my-pkg")
	out, err := runCLI(t, "new", "package", "my-pkg", "--dir", pkgDir)
	require.NoError(t, err)
	assert.Contains(t, out, "Created package my-pkg")
	assert.FileExists(t, filepath.Join(pkgDir, "manifest.json"))
	assert.FileExists(t, filepath.Join(pkgDir, "index.ts"))
	assert.FileExists(t, filepath.Join(pkgDir, "tsconfig.json"))
	assert.FileExists(t, filepath.Join(pkgDir, "types", "kit.d.ts"))
	assert.FileExists(t, filepath.Join(pkgDir, "types", "ai.d.ts"))
	assert.FileExists(t, filepath.Join(pkgDir, "types", "agent.d.ts"))

	data, _ := os.ReadFile(filepath.Join(pkgDir, "manifest.json"))
	assert.Contains(t, string(data), `"name": "my-pkg"`)

	kitDts, _ := os.ReadFile(filepath.Join(pkgDir, "types", "kit.d.ts"))
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

// testNewServer asserts that `brainkit new server` drops a main.go,
// brainkit.yaml, and go.mod that reference the session-11 server
// package. Pairs with testdata/example.yaml's shape.
func testNewServer(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	srvDir := filepath.Join(tmpDir, "my-srv")

	out, err := runCLI(t, "new", "server", "my-srv", "--dir", srvDir)
	require.NoError(t, err)
	assert.Contains(t, out, "Created server my-srv")

	assert.FileExists(t, filepath.Join(srvDir, "main.go"))
	assert.FileExists(t, filepath.Join(srvDir, "brainkit.yaml"))
	assert.FileExists(t, filepath.Join(srvDir, "go.mod"))
	assert.FileExists(t, filepath.Join(srvDir, "README.md"))

	mainGo, _ := os.ReadFile(filepath.Join(srvDir, "main.go"))
	assert.Contains(t, string(mainGo), `"github.com/brainlet/brainkit/server"`)
	assert.Contains(t, string(mainGo), "server.LoadConfig")
	assert.Contains(t, string(mainGo), "server.New")

	yaml, _ := os.ReadFile(filepath.Join(srvDir, "brainkit.yaml"))
	assert.Contains(t, string(yaml), "namespace: my-srv")
	assert.Contains(t, string(yaml), "transport:")
	assert.Contains(t, string(yaml), "gateway:")
}

// testInspectHealth boots a test server and asserts that
// `brainkit inspect health --endpoint ...` returns a parseable
// status table.
func testInspectHealth(t *testing.T, _ *suite.TestEnv) {
	addr := startTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := runCLICtx(t, ctx, "inspect", "health", "--endpoint", addr)
	require.NoError(t, err, "output: %s", out)
	assert.Contains(t, out, "STATUS", "inspect health should render a status table: %s", out)
}

// testInspectHealthJSON asserts that `--json` flips the output
// from table form to raw payload JSON.
func testInspectHealthJSON(t *testing.T, _ *suite.TestEnv) {
	addr := startTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := runCLICtx(t, ctx, "--json", "inspect", "health", "--endpoint", addr)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &payload), "output: %s", out)
}

// testCallVerb asserts the generic `brainkit call <topic>` path
// works against a running server — POSTs to /api/bus and returns
// the reply.
func testCallVerb(t *testing.T, _ *suite.TestEnv) {
	addr := startTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := runCLICtx(t, ctx, "call", "kit.health",
		"--endpoint", addr, "--payload", `{}`)
	require.NoError(t, err, "output: %s", out)

	var shape map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &shape), "call output must be JSON: %s", out)
}

// testDeployVerb boots a server, drops a tiny .ts file, and
// deploys it via `brainkit deploy --endpoint ...`.
func testDeployVerb(t *testing.T, _ *suite.TestEnv) {
	addr := startTestServer(t)

	tmp := t.TempDir()
	tsFile := filepath.Join(tmp, "hello-cli.ts")
	require.NoError(t, os.WriteFile(tsFile, []byte(`
		bus.on("ping", (msg) => { msg.reply({ pong: true }); });
	`), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := runCLICtx(t, ctx, "--timeout", "25s", "deploy", tsFile, "--endpoint", addr)
	require.NoError(t, err, "deploy output: %s", out)
	assert.Contains(t, out, "Deployed hello-cli", "output: %s", out)

	// Confirm it shows up in `inspect packages`.
	out, err = runCLICtx(t, ctx, "--timeout", "10s", "inspect", "packages", "--endpoint", addr)
	require.NoError(t, err, "inspect output: %s", out)
	assert.Contains(t, out, "hello-cli", "inspect should list the deployed package: %s", out)
}

// testDeployFullWorkflow stitches deploy + call into one round
// trip so we have coverage of `brainkit call <topic>` against
// a deployed .ts handler.
func testDeployFullWorkflow(t *testing.T, _ *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping CLI full workflow in short mode")
	}

	addr := startTestServer(t)

	tmp := t.TempDir()
	tsFile := filepath.Join(tmp, "echo-cli.ts")
	require.NoError(t, os.WriteFile(tsFile, []byte(`
		bus.on("ping", (msg) => { msg.reply({ pong: msg.payload.value }); });
	`), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := runCLICtx(t, ctx, "--timeout", "25s", "deploy", tsFile, "--endpoint", addr)
	require.NoError(t, err, "deploy: %s", out)

	// Call the deployed handler over the bus through the CLI.
	out, err = runCLICtx(t, ctx, "--timeout", "10s",
		"call", "ts.echo-cli.ping",
		"--endpoint", addr, "--payload", `{"value":"from-cli"}`)
	require.NoError(t, err, "call: %s", out)
	assert.Contains(t, out, "from-cli", "round-trip must carry the payload back: %s", out)
}

// silence unused warnings for helpers referenced by other tests
// in the domain.
var _ = fmt.Sprintf
