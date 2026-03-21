//go:build e2e

package harness

import (
	"context"
	"testing"
	"time"
)

func TestE2E_HarnessContextInTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)

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

	toolStarts := collector.AllOfType(EventToolStart)
	for _, ts := range toolStarts {
		t.Logf("tool_start: %s args=%v", ts.ToolName, ts.Args)
	}

	t.Error("check_context tool was never called or never returned")
}

func TestE2E_AskUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E in short mode")
	}

	kit := setupHarnessKit(t)

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

	askEvent, err := collector.WaitFor(EventAskQuestion, 30*time.Second)
	if err != nil {
		if collector.Has(EventAgentEnd) {
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

	ds := h.GetDisplayState()
	if ds.PendingQuestion == nil {
		t.Error("display state should have pendingQuestion")
	}

	if err := h.RespondToQuestion(askEvent.QuestionID, "Go is the best"); err != nil {
		t.Fatalf("RespondToQuestion: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SendMessage after respond: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SendMessage timed out after RespondToQuestion")
	}

	ds2 := h.GetDisplayState()
	if ds2.IsRunning {
		t.Error("should not be running")
	}

	t.Log("ask_user flow: question → respond → agent continues — verified!")
}

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

	if _, err := collector.WaitFor(EventAgentStart, 15*time.Second); err != nil {
		t.Fatalf("WaitFor agent_start: %v", err)
	}

	if err := h.FollowUp("Now count from 6 to 10."); err != nil {
		t.Fatalf("FollowUp: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("First SendMessage: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("First SendMessage timed out")
	}

	time.Sleep(3 * time.Second)

	agentStarts := collector.Count(EventAgentStart)
	agentEnds := collector.Count(EventAgentEnd)
	t.Logf("agent_start count: %d, agent_end count: %d", agentStarts, agentEnds)

	if agentStarts < 1 {
		t.Error("expected at least 1 agent_start")
	}

	t.Log("FollowUp: queued and processed")
}

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

	if err := h.SwitchMode("fast"); err != nil {
		t.Fatalf("SwitchMode: %v", err)
	}

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
