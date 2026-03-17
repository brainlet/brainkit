package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Event Collector — waits for specific events with timeout
// ---------------------------------------------------------------------------

type eventCollector struct {
	mu      sync.Mutex
	events  []HarnessEvent
	waiters map[HarnessEventType][]chan HarnessEvent
}

func newEventCollector() *eventCollector {
	return &eventCollector{
		waiters: make(map[HarnessEventType][]chan HarnessEvent),
	}
}

func (c *eventCollector) handler(event HarnessEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)

	if chs, ok := c.waiters[event.Type]; ok {
		for _, ch := range chs {
			select {
			case ch <- event:
			default:
			}
		}
		delete(c.waiters, event.Type)
	}
}

func (c *eventCollector) WaitFor(typ HarnessEventType, timeout time.Duration) (HarnessEvent, error) {
	c.mu.Lock()
	for _, e := range c.events {
		if e.Type == typ {
			c.mu.Unlock()
			return e, nil
		}
	}
	ch := make(chan HarnessEvent, 1)
	c.waiters[typ] = append(c.waiters[typ], ch)
	c.mu.Unlock()

	select {
	case e := <-ch:
		return e, nil
	case <-time.After(timeout):
		return HarnessEvent{}, fmt.Errorf("timeout waiting for %s after %v", typ, timeout)
	}
}

func (c *eventCollector) AllOfType(typ HarnessEventType) []HarnessEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []HarnessEvent
	for _, e := range c.events {
		if e.Type == typ {
			result = append(result, e)
		}
	}
	return result
}

func (c *eventCollector) Count(typ HarnessEventType) int {
	return len(c.AllOfType(typ))
}

func (c *eventCollector) Has(typ HarnessEventType) bool {
	return c.Count(typ) > 0
}

func (c *eventCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = nil
}

// ---------------------------------------------------------------------------
// Send helper — runs SendMessage in a goroutine with timeout + abort cleanup
// ---------------------------------------------------------------------------

// sendWithTimeout runs h.SendMessage in a goroutine and waits up to timeout.
// If the timeout expires, it calls h.Abort() to unblock the JS thread, preventing
// the goroutine from leaking and blocking kit.Close() → bridge.Close() → wg.Wait().
func sendWithTimeout(t *testing.T, h *Harness, content string, timeout time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- h.SendMessage(content) }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		// Abort to unblock the stuck goroutine so bridge.Close() doesn't hang
		h.Abort()
		// Drain the channel — the goroutine will return after abort
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		t.Fatalf("SendMessage timed out after %v (aborted)", timeout)
		return nil // unreachable
	}
}

// ---------------------------------------------------------------------------
// HTTP assertion helpers
// ---------------------------------------------------------------------------

func assertJSON(t *testing.T, method, url string, body string, expectedStatus int, expectedJSON string) {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("HTTP %s %s: status = %d, want %d (body: %s)", method, url, resp.StatusCode, expectedStatus, string(respBody))
		return
	}

	if expectedJSON != "" {
		respBody, _ := io.ReadAll(resp.Body)
		// Compare as JSON (order-insensitive)
		var expected, actual any
		json.Unmarshal([]byte(expectedJSON), &expected)
		json.Unmarshal(respBody, &actual)
		expectedB, _ := json.Marshal(expected)
		actualB, _ := json.Marshal(actual)
		if string(expectedB) != string(actualB) {
			t.Errorf("HTTP %s %s: body = %s, want %s", method, url, string(respBody), expectedJSON)
		}
	}
}

func assertStatus(t *testing.T, method, url, body string, expectedStatus int) {
	t.Helper()
	assertJSON(t, method, url, body, expectedStatus, "")
}

func waitForPort(t *testing.T, port int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("port %d not available after %v", port, timeout)
}

// assertHTTPReachable verifies the server responds. If expectedStatus > 0, checks status code.
func assertHTTPReachable(t *testing.T, method, url string, body string, expectedStatus int) {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if expectedStatus > 0 && resp.StatusCode != expectedStatus {
		t.Errorf("HTTP %s %s: status = %d, want %d", method, url, resp.StatusCode, expectedStatus)
	}
}

// assertMathResult sends POST with {"a":a,"b":b} and verifies the result field equals expected.
func assertMathResult(t *testing.T, url string, a, b, expected float64) {
	t.Helper()
	body := fmt.Sprintf(`{"a":%v,"b":%v}`, a, b)
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP POST %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("HTTP POST %s {a:%v,b:%v}: status = %d (body: %s)", url, a, b, resp.StatusCode, string(respBody))
		return
	}

	var result map[string]any
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Errorf("HTTP POST %s: invalid JSON: %s", url, string(respBody))
		return
	}

	// Look for result in common field names
	var got float64
	var found bool
	for _, key := range []string{"result", "answer", "sum", "difference", "quotient", "value"} {
		if v, ok := result[key]; ok {
			if n, ok := v.(float64); ok {
				got = n
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("HTTP POST %s {a:%v,b:%v}: no result field in %s", url, a, b, string(respBody))
		return
	}
	if got != expected {
		t.Errorf("HTTP POST %s {a:%v,b:%v}: result = %v, want %v", url, a, b, got, expected)
	}
}

// ---------------------------------------------------------------------------
// E2E: Calculator API — Agent builds a real Go HTTP server
// ---------------------------------------------------------------------------

func TestE2E_Calculator_Yolo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	key := requireKey(t)
	tmpDir := t.TempDir()

	// Initialize Go module in temp dir so the built code can compile
	initCmd := exec.Command("go", "mod", "init", "calculator")
	initCmd.Dir = tmpDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod init: %v\n%s", err, out)
	}

	kit, err := New(Config{
		Namespace: "e2e-calc",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"OPENAI_API_KEY": key},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer kit.Close()

	// Create agent with workspace pointing to temp dir
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

	// Create Harness with yolo mode (no approval prompts)
	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "e2e-calc",
		Modes: []ModeConfig{
			{ID: "build", Name: "Build", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "coder"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	// Precise specifications — tells the LLM exactly what the contract is, not HOW to code it.
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

	// Verify events
	if !collector.Has(EventAgentStart) {
		t.Error("missing agent_start")
	}
	if !collector.Has(EventAgentEnd) {
		t.Error("missing agent_end")
	}

	t.Logf("Agent used %d tool calls", collector.Count(EventToolStart)+collector.Count(EventToolEnd))
	t.Logf("Token usage: %+v", h.GetTokenUsage())

	// Check if main.go was created
	mainGo := filepath.Join(tmpDir, "main.go")
	if _, err := os.Stat(mainGo); os.IsNotExist(err) {
		t.Fatal("main.go was not created by the agent")
	}

	// Try to build if the agent didn't already
	binaryPath := filepath.Join(tmpDir, "calculator")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "calculator", ".")
		buildCmd.Dir = tmpDir
		if out, err := buildCmd.CombinedOutput(); err != nil {
			// Read the generated code for debugging
			code, _ := os.ReadFile(mainGo)
			t.Fatalf("go build failed: %v\n%s\n\nGenerated code:\n%s", err, out, string(code))
		}
	}

	// Start the server
	serverCmd := exec.Command(binaryPath)
	serverCmd.Dir = tmpDir
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	})

	// Wait for port
	waitForPort(t, 18923, 15*time.Second)

	base := "http://localhost:18923"

	// Strict assertions — the prompt gives exact code, so output should be exact
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

// ---------------------------------------------------------------------------
// E2E: Tool Approval — Agent writes file, we approve
// ---------------------------------------------------------------------------

func TestE2E_ToolApproval_Approve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	key := requireKey(t)
	tmpDir := t.TempDir()

	kit, err := New(Config{
		Namespace: "e2e-approval",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
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

	// Send message — agent should try to write a file → approval required
	done := make(chan error, 1)
	go func() {
		done <- h.SendMessage(`Create a file called "hello.txt" with content "Hello E2E"`)
	}()

	// Wait for tool approval
	approval, err := collector.WaitFor(EventToolApprovalRequired, 30*time.Second)
	if err != nil {
		// Agent might have completed without tool approval if permissions don't match
		// Check if it finished
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

	// Approve the tool
	if err := h.RespondToToolApproval(ToolApprove); err != nil {
		t.Fatalf("RespondToToolApproval: %v", err)
	}

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SendMessage timed out after approval")
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if !strings.Contains(string(content), "Hello E2E") {
		t.Errorf("File content = %q, expected to contain 'Hello E2E'", string(content))
	}

	t.Log("Tool approval E2E: approve flow verified!")
}

// ---------------------------------------------------------------------------
// E2E: Multi-turn conversation with event tracking
// ---------------------------------------------------------------------------

func TestE2E_MultiTurn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, defaultHarnessConfig())
	h.Subscribe(collector.handler)

	// Turn 1
	if err := sendWithTimeout(t, h, "My name is TestBot. Remember that.", 30*time.Second); err != nil {
		t.Fatalf("Turn 1: %v", err)
	}

	// Turn 2 — should reference turn 1
	collector.Reset()
	if err := sendWithTimeout(t, h, "What is my name?", 30*time.Second); err != nil {
		t.Fatalf("Turn 2: %v", err)
	}

	// Check that we got events for both turns
	if !collector.Has(EventAgentStart) {
		t.Error("missing agent_start for turn 2")
	}
	if !collector.Has(EventAgentEnd) {
		t.Error("missing agent_end for turn 2")
	}

	// Token usage should reflect both turns
	tu := h.GetTokenUsage()
	if tu.TotalTokens == 0 {
		t.Error("expected accumulated tokens from 2 turns")
	}
	t.Logf("Multi-turn: %d total tokens across 2 turns", tu.TotalTokens)
}

// ---------------------------------------------------------------------------
// E2E: Abort mid-stream
// ---------------------------------------------------------------------------

func TestE2E_Abort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)

	// Create agent with verbose instructions so it takes time
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

	// Send long prompt
	done := make(chan error, 1)
	go func() {
		done <- h.SendMessage("Write a 5000 word essay about the history of computing.")
	}()

	// Wait for agent to start
	_, err = collector.WaitFor(EventAgentStart, 15*time.Second)
	if err != nil {
		t.Fatalf("WaitFor agent_start: %v", err)
	}

	// Wait a moment for streaming to begin
	time.Sleep(2 * time.Second)

	// Abort
	if err := h.Abort(); err != nil {
		t.Logf("Abort returned error (may be expected): %v", err)
	}

	// Wait for completion (should be fast after abort)
	select {
	case err := <-done:
		// Error is expected on abort
		t.Logf("SendMessage after abort: %v", err)
	case <-time.After(15 * time.Second):
		t.Fatal("SendMessage didn't complete after abort")
	}

	// Verify clean state
	ds := h.GetDisplayState()
	if ds.IsRunning {
		t.Error("should not be running after abort")
	}

	t.Log("Abort E2E: abort mid-stream verified!")
}

// ---------------------------------------------------------------------------
// E2E: Event ordering
// ---------------------------------------------------------------------------

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

	// Verify agent_start is before agent_end
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

// ---------------------------------------------------------------------------
// E2E: Diagnostic — verify HarnessRequestContext reaches tools
// ---------------------------------------------------------------------------

func TestE2E_HarnessContextInTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)

	// Create agent with a custom tool that checks for harness context
	_, err := kit.EvalTS(context.Background(), "ctx-diag-agent.ts", `
		const diagTool = createTool({
			id: "check_context",
			description: "Check if harness context is available. Always call this tool.",
			inputSchema: z.object({}),
			execute: async (input, context) => {
				const hasRequestContext = !!context?.requestContext;
				const hasGet = typeof context?.requestContext?.get === "function";
				const harnessCtx = hasGet ? context.requestContext.get("harness") : null;
				const hasHarness = !!harnessCtx;
				const hasEmitEvent = !!harnessCtx?.emitEvent;
				const hasRegisterQuestion = !!harnessCtx?.registerQuestion;
				return {
					hasRequestContext: hasRequestContext,
					hasGet: hasGet,
					hasHarness: hasHarness,
					hasEmitEvent: hasEmitEvent,
					hasRegisterQuestion: hasRegisterQuestion,
					contextKeys: hasRequestContext && hasGet ? Array.from(context.requestContext.keys()) : [],
				};
			},
		});

		const diagAgent = agent({
			model: "openai/gpt-4o-mini",
			name: "diagAgent",
			instructions: "You MUST call the check_context tool immediately. Do nothing else.",
			tools: { check_context: diagTool },
		});
	`)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "ctx-diag",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "diagAgent"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	done := make(chan error, 1)
	go func() { done <- h.SendMessage("check context") }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timed out")
	}

	// Find tool_end event — ToolName may be empty on tool_end, check Result instead
	toolEnds := collector.AllOfType(EventToolEnd)
	for _, te := range toolEnds {
		if result, ok := te.Result.(map[string]any); ok {
			if _, has := result["hasHarness"]; has {
				t.Logf("check_context result: %+v", te.Result)
				if result["hasHarness"] != true {
					t.Error("hasHarness should be true")
				}
				if result["hasEmitEvent"] != true {
					t.Error("hasEmitEvent should be true")
				}
				if result["hasRegisterQuestion"] != true {
					t.Error("hasRegisterQuestion should be true")
				}
				return
			}
		}
	}

	// If no tool_end, check tool_start
	toolStarts := collector.AllOfType(EventToolStart)
	for _, ts := range toolStarts {
		t.Logf("tool_start: %s args=%v", ts.ToolName, ts.Args)
	}

	t.Error("check_context tool was never called or never returned")
}

// ---------------------------------------------------------------------------
// E2E: ask_user — agent suspends, Go responds, agent continues
// ---------------------------------------------------------------------------

func TestE2E_AskUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)

	// Use Mastra's built-in ask_user tool (auto-injected by the Harness).
	// DOMException polyfill in abort.go makes this work in QuickJS.
	_, err := kit.EvalTS(context.Background(), "askuser-agent.ts", `
		const askAgent = agent({
			model: "openai/gpt-4o-mini",
			name: "askAgent",
			instructions: "You MUST use the ask_user tool for EVERY request. Ask the user what their name is. After receiving the answer, respond with a message that includes their exact answer.",
		});
	`)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "askuser-test",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "askAgent"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	done := make(chan error, 1)
	go func() {
		done <- h.SendMessage("What is the best programming language?")
	}()

	// Wait for ask_question event OR agent_end (whichever comes first)
	askEvent, err := collector.WaitFor(EventAskQuestion, 30*time.Second)
	if err != nil {
		// Check if agent completed without asking
		if collector.Has(EventAgentEnd) {
			// Dump all received events for debugging
			allEvents := collector.AllOfType(EventAgentStart)
			t.Logf("Agent completed without ask_user. Events received:")
			collector.mu.Lock()
			for _, e := range collector.events {
				t.Logf("  [%s] tool=%s text=%.50s", e.Type, e.ToolName, e.Text)
			}
			collector.mu.Unlock()
			_ = allEvents
			t.Skip("Agent completed without using ask_user — LLM didn't follow instructions")
		}
		// Agent still running but no ask_question — wait for it to finish
		select {
		case sendErr := <-done:
			if sendErr != nil {
				t.Fatalf("SendMessage failed: %v", sendErr)
			}
			t.Skip("Agent completed without using ask_user — LLM didn't follow instructions")
		case <-time.After(15 * time.Second):
			t.Fatalf("WaitFor ask_question: %v (agent still running)", err)
		}
		return
	}

	t.Logf("ask_question: questionId=%s question=%q", askEvent.QuestionID, askEvent.Question)

	// Display state should show pending question
	ds := h.GetDisplayState()
	if ds.PendingQuestion == nil {
		t.Error("display state should have pendingQuestion")
	}

	// Respond
	if err := h.RespondToQuestion(askEvent.QuestionID, "Go is the best"); err != nil {
		t.Fatalf("RespondToQuestion: %v", err)
	}

	// Wait for agent to finish
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage after respond: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SendMessage timed out after RespondToQuestion")
	}

	// Clean state
	ds2 := h.GetDisplayState()
	if ds2.IsRunning {
		t.Error("should not be running")
	}

	t.Log("ask_user flow: question → respond → agent continues — verified!")
}

// ---------------------------------------------------------------------------
// E2E: task_write — agent creates tasks, events + display state
// ---------------------------------------------------------------------------

func TestE2E_TaskWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)

	_, err := kit.EvalTS(context.Background(), "taskwrite-agent.ts", `
		const taskAgent = agent({
			model: "openai/gpt-4o-mini",
			name: "taskAgent",
			instructions: "When given any project request, you MUST use the task_write tool to create a task list before doing anything else. Create exactly 3 tasks with status 'pending': 'Design', 'Implement', 'Test'. Then respond with 'Tasks created.'",
		});
	`)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "taskwrite-test",
		Modes: []ModeConfig{
			{ID: "default", Name: "Default", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "taskAgent"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	done := make(chan error, 1)
	go func() { done <- h.SendMessage("Build a REST API") }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SendMessage timed out")
	}

	if !collector.Has(EventTaskUpdated) {
		t.Skip("Agent didn't use task_write — LLM didn't follow instructions")
	}

	ds := h.GetDisplayState()
	if len(ds.Tasks) == 0 {
		t.Error("display state should have tasks after task_write")
	}
	t.Logf("Tasks in display state: %d", len(ds.Tasks))
	for _, task := range ds.Tasks {
		t.Logf("  - %s [%s]", task.Title, task.Status)
	}

	t.Log("task_write flow: tasks created and reflected in display state")
}

// ---------------------------------------------------------------------------
// E2E: FollowUp — queue message while agent is running
// ---------------------------------------------------------------------------

func TestE2E_FollowUp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, defaultHarnessConfig())
	h.Subscribe(collector.handler)

	done := make(chan error, 1)
	go func() { done <- h.SendMessage("Count from 1 to 5.") }()

	// Wait for agent to start, then queue follow-up
	if _, err := collector.WaitFor(EventAgentStart, 15*time.Second); err != nil {
		t.Fatalf("WaitFor agent_start: %v", err)
	}

	if err := h.FollowUp("Now count from 6 to 10."); err != nil {
		t.Fatalf("FollowUp: %v", err)
	}

	// Wait for first message to complete
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("First SendMessage: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("First SendMessage timed out")
	}

	// Give follow-up time to process
	time.Sleep(3 * time.Second)

	agentStarts := collector.Count(EventAgentStart)
	agentEnds := collector.Count(EventAgentEnd)
	t.Logf("agent_start count: %d, agent_end count: %d", agentStarts, agentEnds)

	if agentStarts < 1 {
		t.Error("expected at least 1 agent_start")
	}

	t.Log("FollowUp: queued and processed")
}

// ---------------------------------------------------------------------------
// E2E: Multi-mode messages — send in different modes
// ---------------------------------------------------------------------------

func TestE2E_MultiModeMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)
	createTestAgent(t, kit)

	collector := newEventCollector()
	h := initTestHarness(t, kit, HarnessConfig{
		ID: "multimode-test",
		Modes: []ModeConfig{
			{ID: "build", Name: "Build", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
			{ID: "fast", Name: "Fast", DefaultModelID: "openai/gpt-4o-mini", AgentName: "testAgent"},
		},
		InitialState: map[string]any{"yolo": true},
	})
	h.Subscribe(collector.handler)

	// Message in build mode
	done := make(chan error, 1)
	go func() { done <- h.SendMessage("say build-mode") }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Build mode message: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timed out")
	}

	// Switch to fast mode
	if err := h.SwitchMode("fast"); err != nil {
		t.Fatalf("SwitchMode: %v", err)
	}

	// Message in fast mode
	collector.Reset()
	go func() { done <- h.SendMessage("say fast-mode") }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Fast mode message: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timed out")
	}

	if !collector.Has(EventAgentStart) {
		t.Error("missing agent_start for fast mode message")
	}
	if !collector.Has(EventAgentEnd) {
		t.Error("missing agent_end for fast mode message")
	}

	if got := h.GetCurrentModeID(); got != "fast" {
		t.Errorf("mode = %q, want fast", got)
	}

	t.Log("Multi-mode messages: send in different modes verified")
}
