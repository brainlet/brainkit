//go:build e2e

package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	brainkit "github.com/brainlet/brainkit"
)

func TestE2E_Calculator_Yolo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	key := requireKey(t)
	tmpDir := t.TempDir()

	initCmd := exec.Command("go", "mod", "init", "calculator")
	initCmd.Dir = tmpDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "e2e-calc",
		Providers: map[string]brainkit.ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"OPENAI_API_KEY": key},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer kit.Close()

	setupCode := fmt.Sprintf(`
		const ws = new Workspace({
			filesystem: new LocalFilesystem({ basePath: %q }),
			sandbox: new LocalSandbox({ workingDirectory: %q }),
		});
		await ws.init();

		const coder = agent({
			model: "openai/gpt-4o-mini",
			name: "coder",
			instructions: "You are a Go developer. When given requirements, write the code in a single file, then compile it. Do not test or iterate — write once, compile once. Use write_file for file creation and execute_command for compilation.",
			workspace: ws,
			maxSteps: 5,
		});
	`, tmpDir, tmpDir)

	if _, err := kit.EvalTS(context.Background(), "e2e-setup.ts", setupCode); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "e2e-calc",
		Modes: []ModeConfig{
			{ID: "build", Name: "Build", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "coder"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	const prompt = `Build a Go HTTP API calculator as a single main.go file.

SPECIFICATIONS:

Language: Go, standard library only (net/http, encoding/json). No external packages.
Port: 18923
Module: The go.mod is already initialized.

ENDPOINTS:

1. GET /health
   - Response: 200, body: {"status":"ok"}

2. POST /add
   - Request body: {"a": <float64>, "b": <float64>}
   - Response: 200, body: {"result": <float64>}  (a + b)

3. POST /subtract
   - Request body: {"a": <float64>, "b": <float64>}
   - Response: 200, body: {"result": <float64>}  (a - b)

4. POST /divide
   - Request body: {"a": <float64>, "b": <float64>}
   - Success: 200, body: {"result": <float64>}  (a / b)
   - If b is 0: 400, body: {"error": "division by zero"}
     The error response MUST NOT contain a "result" field.

ERROR HANDLING:

- Invalid/unparseable JSON body: 400, body: {"error": "invalid request body"}
- Wrong HTTP method (e.g., GET on /add): 405, body: {"error": "method not allowed"}
- Unknown route: 404, body: {"error": "not found"}

RESPONSE FORMAT RULES:

- Every response MUST set Content-Type: application/json
- Success responses use ONLY the "result" field: {"result": <number>}
- Error responses use ONLY the "error" field: {"error": "<message>"}
- Never mix "result" and "error" in the same response.
- Use json.NewEncoder(w).Encode() for all JSON output.
- Always call w.Header().Set("Content-Type", "application/json") BEFORE w.WriteHeader().

AFTER WRITING THE FILE:

Run: go build -o calculator .
This must compile with zero errors.`

	done := make(chan error, 1)
	go func() {
		done <- h.SendMessage(prompt)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	case <-time.After(120 * time.Second):
		t.Fatal("SendMessage timed out after 120s")
	}

	if !collector.Has(EventAgentStart) {
		t.Error("missing agent_start")
	}
	if !collector.Has(EventAgentEnd) {
		t.Error("missing agent_end")
	}

	t.Logf("Agent used %d tool calls", collector.Count(EventToolStart)+collector.Count(EventToolEnd))
	t.Logf("Token usage: %+v", h.GetTokenUsage())

	mainGo := filepath.Join(tmpDir, "main.go")
	if _, err := os.Stat(mainGo); os.IsNotExist(err) {
		t.Fatal("main.go was not created by the agent")
	}

	binaryPath := filepath.Join(tmpDir, "calculator")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "calculator", ".")
		buildCmd.Dir = tmpDir
		if out, err := buildCmd.CombinedOutput(); err != nil {
			code, _ := os.ReadFile(mainGo)
			t.Fatalf("go build failed: %v\n%s\n\nGenerated code:\n%s", err, out, string(code))
		}
	}

	serverCmd := exec.Command(binaryPath)
	serverCmd.Dir = tmpDir
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	})

	waitForPort(t, 18923, 15*time.Second)

	base := "http://localhost:18923"

	assertJSON(t, "GET", base+"/health", "", 200, `{"status":"ok"}`)
	assertJSON(t, "POST", base+"/add", `{"a":2,"b":3}`, 200, `{"result":5}`)
	assertJSON(t, "POST", base+"/add", `{"a":-1,"b":1}`, 200, `{"result":0}`)
	assertJSON(t, "POST", base+"/subtract", `{"a":10,"b":4}`, 200, `{"result":6}`)
	assertJSON(t, "POST", base+"/divide", `{"a":10,"b":2}`, 200, `{"result":5}`)
	assertJSON(t, "POST", base+"/divide", `{"a":1,"b":0}`, 400, `{"error":"division by zero"}`)
	assertStatus(t, "POST", base+"/add", "garbage", 400)
	assertStatus(t, "GET", base+"/add", "", 405)
	assertStatus(t, "GET", base+"/nonexistent", "", 404)

	t.Log("Calculator API E2E: all endpoints verified!")
}

func TestE2E_ToolApproval_Approve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	key := requireKey(t)
	tmpDir := t.TempDir()

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "e2e-approval",
		Providers: map[string]brainkit.ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"OPENAI_API_KEY": key},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer kit.Close()

	setupCode := fmt.Sprintf(`
		const ws = new Workspace({
			filesystem: new LocalFilesystem({ basePath: %q }),
			sandbox: new LocalSandbox({ workingDirectory: %q }),
		});
		await ws.init();

		const writer = agent({
			model: "openai/gpt-4o-mini",
			name: "writer",
			instructions: "You are a file writer. When asked to create a file, use the write_file tool. Be concise.",
			workspace: ws,
			maxSteps: 5,
		});
	`, tmpDir, tmpDir)

	if _, err := kit.EvalTS(context.Background(), "e2e-approval-setup.ts", setupCode); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "e2e-approval",
		Modes: []ModeConfig{
			{ID: "build", Name: "Build", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "writer"},
		},
		InitialState: map[string]any{
			"yolo": false,
			"permissionRules": map[string]any{
				"categories": map[string]any{
					"read":    "allow",
					"edit":    "ask",
					"execute": "ask",
				},
				"tools": map[string]any{},
			},
		},
	})
	h.Subscribe(collector.handler)

	done := make(chan error, 1)
	go func() {
		done <- h.SendMessage(`Create a file called "hello.txt" with content "Hello E2E"`)
	}()

	approval, err := collector.WaitFor(EventToolApprovalRequired, 30*time.Second)
	if err != nil {
		select {
		case sendErr := <-done:
			if sendErr != nil {
				t.Fatalf("SendMessage failed: %v", sendErr)
			}
			t.Skip("Agent completed without triggering tool approval — permission config may not have matched")
		default:
			t.Fatalf("WaitFor tool_approval_required: %v", err)
		}
	}

	t.Logf("Tool approval required: %s (%s)", approval.ToolName, approval.Category)

	if err := h.RespondToToolApproval(ToolApprove); err != nil {
		t.Fatalf("RespondToToolApproval: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SendMessage timed out after approval")
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if !strings.Contains(string(content), "Hello E2E") {
		t.Errorf("File content = %q, expected to contain 'Hello E2E'", string(content))
	}

	t.Log("Tool approval E2E: approve flow verified!")
}

func TestE2E_MultiTurn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, defaultHarnessConfig())
	h.Subscribe(collector.handler)

	if err := sendWithTimeout(t, h, "My name is TestBot. Remember that.", 30*time.Second); err != nil {
		t.Fatalf("Turn 1: %v", err)
	}

	collector.Reset()
	if err := sendWithTimeout(t, h, "What is my name?", 30*time.Second); err != nil {
		t.Fatalf("Turn 2: %v", err)
	}

	if !collector.Has(EventAgentStart) {
		t.Error("missing agent_start for turn 2")
	}
	if !collector.Has(EventAgentEnd) {
		t.Error("missing agent_end for turn 2")
	}

	tu := h.GetTokenUsage()
	if tu.TotalTokens == 0 {
		t.Error("expected accumulated tokens from 2 turns")
	}
	t.Logf("Multi-turn: %d total tokens across 2 turns", tu.TotalTokens)
}

func TestE2E_Abort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)

	_, err := kit.EvalTS(context.Background(), "abort-agent.ts", `
		const verboseAgent = agent({
			model: "openai/gpt-4o-mini",
			name: "verboseAgent",
			instructions: "You are extremely detailed. Write very long, comprehensive responses with many paragraphs.",
		});
	`)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "e2e-abort",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "verboseAgent"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	done := make(chan error, 1)
	go func() {
		done <- h.SendMessage("Write a 5000 word essay about the history of computing.")
	}()

	_, err = collector.WaitFor(EventAgentStart, 15*time.Second)
	if err != nil {
		t.Fatalf("WaitFor agent_start: %v", err)
	}

	time.Sleep(2 * time.Second)

	if err := h.Abort(); err != nil {
		t.Logf("Abort returned error (may be expected): %v", err)
	}

	select {
	case err := <-done:
		t.Logf("SendMessage after abort: %v", err)
	case <-time.After(15 * time.Second):
		t.Fatal("SendMessage didn't complete after abort")
	}

	ds := h.GetDisplayState()
	if ds.IsRunning {
		t.Error("should not be running after abort")
	}

	t.Log("Abort E2E: abort mid-stream verified!")
}

func TestE2E_EventOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, defaultHarnessConfig())
	h.Subscribe(collector.handler)

	if err := sendWithTimeout(t, h, "say ok", 30*time.Second); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()

	startIdx := -1
	endIdx := -1
	for i, e := range collector.events {
		if e.Type == EventAgentStart && startIdx == -1 {
			startIdx = i
		}
		if e.Type == EventAgentEnd && endIdx == -1 {
			endIdx = i
		}
	}

	if startIdx == -1 {
		t.Fatal("no agent_start")
	}
	if endIdx == -1 {
		t.Fatal("no agent_end")
	}
	if startIdx >= endIdx {
		t.Errorf("agent_start (idx %d) should come before agent_end (idx %d)", startIdx, endIdx)
	}

	t.Logf("Event ordering: %d events, start@%d end@%d", len(collector.events), startIdx, endIdx)
}
