package harness

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Event type serialization
// ---------------------------------------------------------------------------

func TestHarnessEvent_Unmarshal(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect HarnessEventType
	}{
		{"agent_start", `{"type":"agent_start","threadId":"t1","runId":"r1","content":"hello","modeId":"build","modelId":"openai/gpt-4o"}`, EventAgentStart},
		{"agent_end", `{"type":"agent_end","threadId":"t1","text":"done","finishReason":"stop"}`, EventAgentEnd},
		{"message_update", `{"type":"message_update","messageId":"m1","text":"hello "}`, EventMessageUpdate},
		{"tool_start", `{"type":"tool_start","toolCallId":"tc1","toolName":"write_file","args":{"path":"main.go"}}`, EventToolStart},
		{"tool_approval_required", `{"type":"tool_approval_required","toolCallId":"tc1","toolName":"write_file","category":"edit"}`, EventToolApprovalRequired},
		{"ask_question", `{"type":"ask_question","questionId":"q1","question":"What port?","options":["8080","3000"]}`, EventAskQuestion},
		{"task_updated", `{"type":"task_updated","tasks":[{"title":"Build API","status":"pending"}]}`, EventTaskUpdated},
		{"om_status", `{"type":"om_status","status":"idle","observationThreshold":5}`, EventOMStatus},
		{"error", `{"type":"error","error":"something failed","fatal":true}`, EventError},
		{"subagent_start", `{"type":"subagent_start","toolCallId":"tc1","agentType":"explore","task":"find auth"}`, EventSubagentStart},
		{"shell_output", `{"type":"shell_output","toolCallId":"tc1","stream":"stdout","data":"hello\n"}`, EventShellOutput},
		{"plan_approval_required", `{"type":"plan_approval_required","planId":"p1","plan":"Step 1: ..."}`, EventPlanApprovalRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event HarnessEvent
			err := json.Unmarshal([]byte(tt.input), &event)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if event.Type != tt.expect {
				t.Errorf("type = %q, want %q", event.Type, tt.expect)
			}
		})
	}
}

func TestHarnessEvent_UnknownType(t *testing.T) {
	var event HarnessEvent
	err := json.Unmarshal([]byte(`{"type":"future_event","data":"something"}`), &event)
	if err != nil {
		t.Fatalf("should not error on unknown type: %v", err)
	}
	if event.Type != "future_event" {
		t.Errorf("type = %q, want %q", event.Type, "future_event")
	}
}

func TestHarnessEvent_ToolArgs(t *testing.T) {
	var event HarnessEvent
	err := json.Unmarshal([]byte(`{"type":"tool_start","toolCallId":"tc1","toolName":"write_file","args":{"path":"main.go","content":"package main"}}`), &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Args["path"] != "main.go" {
		t.Errorf("args.path = %v", event.Args["path"])
	}
	if event.Args["content"] != "package main" {
		t.Errorf("args.content = %v", event.Args["content"])
	}
}

func TestHarnessEvent_TaskList(t *testing.T) {
	var event HarnessEvent
	err := json.Unmarshal([]byte(`{"type":"task_updated","tasks":[{"title":"Build API","description":"Create endpoints","status":"pending"},{"title":"Test","status":"completed"}]}`), &event)
	if err != nil {
		t.Fatal(err)
	}
	if len(event.Tasks) != 2 {
		t.Fatalf("tasks len = %d, want 2", len(event.Tasks))
	}
	if event.Tasks[0].Title != "Build API" {
		t.Errorf("tasks[0].title = %q", event.Tasks[0].Title)
	}
	if event.Tasks[1].Status != "completed" {
		t.Errorf("tasks[1].status = %q", event.Tasks[1].Status)
	}
}

func TestTokenUsage_Zero(t *testing.T) {
	var tu TokenUsage
	if tu.TotalTokens != 0 || tu.PromptTokens != 0 || tu.CompletionTokens != 0 {
		t.Error("zero value should have all zeros")
	}
}

// ---------------------------------------------------------------------------
// Config validation
// ---------------------------------------------------------------------------

func TestValidateHarnessConfig_MissingID(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{})
	if err == nil || err.Error() != "harness: ID is required" {
		t.Errorf("expected ID error, got: %v", err)
	}
}

func TestValidateHarnessConfig_NoModes(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{ID: "test"})
	if err == nil {
		t.Error("expected error for no modes")
	}
}

func TestValidateHarnessConfig_NoDefault(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "a", AgentName: "agent1"}},
	})
	if err == nil {
		t.Error("expected error for no default mode")
	}
}

func TestValidateHarnessConfig_MultipleDefaults(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID: "test",
		Modes: []ModeConfig{
			{ID: "a", Default: true, AgentName: "agent1"},
			{ID: "b", Default: true, AgentName: "agent1"},
		},
	})
	if err == nil {
		t.Error("expected error for multiple default modes")
	}
}

func TestValidateHarnessConfig_MissingAgentName(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "a", Default: true}},
	})
	if err == nil {
		t.Error("expected error for missing AgentName")
	}
}

func TestValidateHarnessConfig_Valid(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "build", Name: "Build", Default: true, AgentName: "coder"}},
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateHarnessConfig_SubagentNoTools(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:        "test",
		Modes:     []ModeConfig{{ID: "build", Default: true, AgentName: "coder"}},
		Subagents: []HarnessSubagentConfig{{ID: "explore"}},
	})
	if err == nil {
		t.Error("expected error for subagent with no tools")
	}
}

func TestStateSchemaOf_Defaults(t *testing.T) {
	type TestState struct {
		Name    string   `json:"name" default:"unnamed"`
		Enabled bool     `json:"enabled" default:"true"`
		Count   float64  `json:"count" default:"42"`
		Tags    []string `json:"tags" default:"[]"`
	}

	schema := StateSchemaOf[TestState]()
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties not a map: %+v", schema)
	}

	// Check each field has the correct default
	nameP := props["name"].(map[string]any)
	if nameP["default"] != "unnamed" {
		t.Errorf("name default = %v, want 'unnamed'", nameP["default"])
	}
	if nameP["type"] != "string" {
		t.Errorf("name type = %v, want string", nameP["type"])
	}

	enabledP := props["enabled"].(map[string]any)
	if enabledP["default"] != true {
		t.Errorf("enabled default = %v, want true", enabledP["default"])
	}

	countP := props["count"].(map[string]any)
	if countP["default"] != float64(42) {
		t.Errorf("count default = %v, want 42", countP["default"])
	}

	tagsP := props["tags"].(map[string]any)
	if tagsP["type"] != "array" {
		t.Errorf("tags type = %v, want array", tagsP["type"])
	}
}

func TestTypedPermissionConstants(t *testing.T) {
	// Verify typed constants work in map
	perms := DefaultPermissions()
	if perms[CategoryRead] != PolicyAllow {
		t.Errorf("read = %q, want allow", perms[CategoryRead])
	}
	if perms[CategoryEdit] != PolicyAsk {
		t.Errorf("edit = %q, want ask", perms[CategoryEdit])
	}
	if perms[CategoryExecute] != PolicyAsk {
		t.Errorf("execute = %q, want ask", perms[CategoryExecute])
	}
	if perms[CategoryMCP] != PolicyAsk {
		t.Errorf("mcp = %q, want ask", perms[CategoryMCP])
	}
}

func TestValidateHarnessConfig_ValidWithSubagents(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "build", Default: true, AgentName: "coder"}},
		Subagents: []HarnessSubagentConfig{
			{ID: "explore", AllowedTools: []string{"view", "search"}},
		},
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}
