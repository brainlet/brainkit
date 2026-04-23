package testutil

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	podmanOnce     sync.Once
	podmanSocket   string
	podmanErr      error
)

// podmanConnectionEntry matches the JSON output of `podman system connection list --format json`.
type podmanConnectionEntry struct {
	Name     string `json:"Name"`
	URI      string `json:"URI"`
	Identity string `json:"Identity"`
	Default  bool   `json:"Default"`
}

// ResolvePodmanSocket resolves the Podman socket path for the brainkit machine
// (or an override) and validates it. It does not require a *testing.T and returns
// an error on failure so callers can decide whether to fatal, skip, or propagate.
//
// Resolution priority (first non-empty wins):
//  1. TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE
//  2. DOCKER_HOST (already set — honored unchanged)
//  3. CONTAINER_HOST
//  4. BRAINKIT_PODMAN_MACHINE env var (defaults to "brainkit")
//  5. `podman machine inspect <name> --format '{{.ConnectionInfo.PodmanSocket.Path}}'`
//  6. `podman system connection list --format json` → default connection → inspect its machine
func ResolvePodmanSocket() (string, error) {
	podmanOnce.Do(func() {
		podmanSocket, podmanErr = resolvePodmanSocketInternal()
	})
	if podmanErr != nil {
		return "", podmanErr
	}
	return podmanSocket, nil
}

// EnsurePodmanSocket is the testing-aware variant of ResolvePodmanSocket.
// It calls t.Skipf with an actionable message on failure.
// If t is nil it behaves like ResolvePodmanSocket (returns error instead of skipping).
func EnsurePodmanSocket(t testing.TB) string {
	if t != nil {
		t.Helper()
	}

	socket, err := ResolvePodmanSocket()
	if err == nil {
		return socket
	}

	if t != nil {
		t.Skipf("brainkit podman socket not available: %v; run `make podman-ensure`", err)
	}
	return ""
}

// AssertBrainkitMachineRunning verifies the brainkit machine is in the Running state.
// It fails the test with a clear message if it is not.
func AssertBrainkitMachineRunning(t testing.TB) {
	t.Helper()

	machineName := os.Getenv("BRAINKIT_PODMAN_MACHINE")
	if machineName == "" {
		machineName = "brainkit"
	}

	out, err := exec.Command("podman", "machine", "list", "--format", "{{.Name}} {{.Running}}").Output()
	if err != nil {
		if t != nil {
			t.Fatalf("cannot list podman machines: %v", err)
		}
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimSuffix(fields[0], "*") == machineName {
			if fields[1] == "true" {
				return
			}
			if t != nil {
				t.Fatalf("brainkit podman machine is not Running (state=%s); run `make podman-ensure`", fields[1])
			}
			return
		}
	}

	if t != nil {
		t.Fatalf("brainkit podman machine not found in machine list; run `make podman-ensure`")
	}
}

func resolvePodmanSocketInternal() (string, error) {
	// 1. TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE
	if override := os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE"); override != "" {
		if err := validateSocket(override); err != nil {
			return "", fmt.Errorf("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE points to invalid socket %q: %w", override, err)
		}
		_ = os.Setenv("DOCKER_HOST", "unix://"+override)
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		return override, nil
	}

	// 2. DOCKER_HOST already set — honor it unchanged.
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		return strings.TrimPrefix(dockerHost, "unix://"), nil
	}

	// 3. CONTAINER_HOST
	if containerHost := os.Getenv("CONTAINER_HOST"); containerHost != "" {
		socketPath := strings.TrimPrefix(containerHost, "unix://")
		if err := validateSocket(socketPath); err != nil {
			return "", fmt.Errorf("CONTAINER_HOST points to invalid socket %q: %w", containerHost, err)
		}
		_ = os.Setenv("DOCKER_HOST", "unix://"+socketPath)
		_ = os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", socketPath)
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		return socketPath, nil
	}

	// 4. Resolve machine name.
	machineName := os.Getenv("BRAINKIT_PODMAN_MACHINE")
	if machineName == "" {
		machineName = "brainkit"
	}

	// Before attempting to dial, sanity-check the machine is Running.
	// We do this here (not exported) so ResolvePodmanSocket can fail fast
	// with a clear message.
	if err := assertMachineRunning(machineName); err != nil {
		return "", err
	}

	// 5. Inspect the named machine.
	socketPath, err := inspectMachineSocket(machineName)
	if err == nil {
		if validateErr := validateSocket(socketPath); validateErr != nil {
			return "", fmt.Errorf("brainkit podman machine %q socket %q exists but is not reachable: %w; run `make podman-ensure`", machineName, socketPath, validateErr)
		}
		if err := verifySocketBelongsToMachine(socketPath, machineName); err != nil {
			return "", err
		}
		_ = os.Setenv("DOCKER_HOST", "unix://"+socketPath)
		_ = os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", socketPath)
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
		return socketPath, nil
	}

	// 6. Fallback: query default connection from `podman system connection list`.
	defaultName, fallbackErr := defaultConnectionName()
	if fallbackErr != nil {
		return "", fmt.Errorf("cannot resolve podman socket for machine %q (inspect error: %v) and fallback failed: %w", machineName, err, fallbackErr)
	}

	// Re-inspect using the default connection name.
	socketPath, inspectErr := inspectMachineSocket(defaultName)
	if inspectErr != nil {
		return "", fmt.Errorf("cannot resolve podman socket for machine %q (default connection %q also failed: %v); run `make podman-ensure`", machineName, defaultName, inspectErr)
	}

	if validateErr := validateSocket(socketPath); validateErr != nil {
		return "", fmt.Errorf("default connection %q socket %q is not reachable: %w; run `make podman-ensure`", defaultName, socketPath, validateErr)
	}

	// Verify the fallback socket actually matches the expected machine.
	if verifyErr := verifySocketBelongsToMachine(socketPath, machineName); verifyErr != nil {
		return "", verifyErr
	}

	_ = os.Setenv("DOCKER_HOST", "unix://"+socketPath)
	_ = os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", socketPath)
	_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	return socketPath, nil
}

func inspectMachineSocket(name string) (string, error) {
	out, err := exec.Command("podman", "machine", "inspect", name, "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output()
	if err != nil {
		return "", fmt.Errorf("podman machine inspect %q failed: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func defaultConnectionName() (string, error) {
	out, err := exec.Command("podman", "system", "connection", "list", "--format", "json").Output()
	if err != nil {
		return "", fmt.Errorf("podman system connection list failed: %w", err)
	}

	var entries []podmanConnectionEntry
	if jsonErr := json.Unmarshal(out, &entries); jsonErr != nil {
		return "", fmt.Errorf("parse podman connection list JSON: %w", jsonErr)
	}

	for _, e := range entries {
		if e.Default {
			return e.Name, nil
		}
	}

	return "", fmt.Errorf("no default connection found in podman system connection list")
}

func validateSocket(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("socket stat failed: %w", err)
	}

	conn, err := net.DialTimeout("unix", path, 2*time.Second)
	if err != nil {
		return fmt.Errorf("socket dial failed: %w", err)
	}
	_ = conn.Close()
	return nil
}

func verifySocketBelongsToMachine(socketPath, expectedMachine string) error {
	expected, err := inspectMachineSocket(expectedMachine)
	if err != nil {
		return fmt.Errorf("cannot verify socket belongs to machine %q: %w", expectedMachine, err)
	}
	if strings.TrimSpace(expected) != strings.TrimSpace(socketPath) {
		return fmt.Errorf("resolved socket %q does NOT match machine %q expected socket %q; run `make podman-ensure`", socketPath, expectedMachine, expected)
	}
	return nil
}

func assertMachineRunning(name string) error {
	out, err := exec.Command("podman", "machine", "list", "--format", "{{.Name}} {{.Running}}").Output()
	if err != nil {
		return fmt.Errorf("cannot list podman machines: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimSuffix(fields[0], "*") == name {
			if fields[1] == "true" {
				return nil
			}
			return fmt.Errorf("podman machine %q is not Running (state=%s); run `make podman-ensure`", name, fields[1])
		}
	}
	return fmt.Errorf("podman machine %q not found in machine list; run `make podman-ensure`", name)
}
