# Harness in Brainkit

The Harness is an orchestration layer that sits between your UI (terminal, web, telegram) and the Agent. It manages agent execution, conversation threads, modes, tool approval, and streams events in real-time.

---

## Quick Start

```go
// Go side — create Kit, agent, then Harness
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Providers: map[string]brainkit.ProviderConfig{
        "openai": {APIKey: os.Getenv("OPENAI_API_KEY")},
    },
    EnvVars: map[string]string{"OPENAI_API_KEY": os.Getenv("OPENAI_API_KEY")},
})
defer kit.Close()

// Create agent in JS
kit.EvalTS(ctx, "setup.ts", `
    const coder = agent({
        model: "openai/gpt-4o-mini",
        name: "coder",
        instructions: "You are a helpful coding assistant.",
    });
`)

// Create Harness
h, _ := kit.InitHarness(brainkit.HarnessConfig{
    ID: "my-harness",
    Modes: []brainkit.ModeConfig{
        {ID: "build", Name: "Build", Default: true, DefaultModelID: "openai/gpt-4o-mini", AgentName: "coder"},
    },
})
defer h.Close()

// Subscribe to events
h.Subscribe(func(e brainkit.HarnessEvent) {
    fmt.Printf("[%s] %s\n", e.Type, e.Text)
})

// Send a message (blocks until agent finishes)
go func() {
    h.SendMessage("Hello!")
}()
```

---

## Modes

Modes let you switch between different agent configurations. Each mode can use a different model.

```go
h, _ := kit.InitHarness(brainkit.HarnessConfig{
    ID: "my-harness",
    Modes: []brainkit.ModeConfig{
        {ID: "build", Name: "Build", Default: true, DefaultModelID: "anthropic/claude-opus-4-6", AgentName: "coder"},
        {ID: "plan",  Name: "Plan",  DefaultModelID: "openai/gpt-4o",              AgentName: "coder"},
        {ID: "fast",  Name: "Fast",  DefaultModelID: "openai/gpt-4o-mini",         AgentName: "coder"},
    },
})

// Switch modes at runtime
h.SwitchMode("fast")
mode := h.GetCurrentMode() // {ID: "fast", Name: "Fast", ...}
```

All modes can use the same agent with different models, or different agents entirely.

---

## Threads

Threads are persistent conversations. Messages are stored, and you can switch between threads.

```go
// Create a thread
threadID, _ := h.CreateThread(brainkit.WithThreadTitle("Calculator Project"))

// Send messages in this thread
h.SendMessage("Build a calculator API")

// List all threads
threads, _ := h.ListThreads()

// Switch to another thread
h.SwitchThread(otherThreadID)

// Delete a thread
h.DeleteThread(threadID)
```

---

## Tool Approval

The Harness can require user approval before executing tools. This is controlled by permission policies per tool category.

### Categories

| Category | Tools | Default Policy |
|----------|-------|---------------|
| `read` | view, search, find_files | `allow` |
| `edit` | write_file, string_replace | `ask` |
| `execute` | execute_command | `ask` |
| `mcp` | all MCP server tools | `ask` |

### Policies

| Policy | Behavior |
|--------|----------|
| `allow` | Auto-approve, no UI interaction |
| `ask` | Pause, emit `tool_approval_required`, wait for response |
| `deny` | Auto-decline |

### YOLO Mode

Set `yolo: true` in the initial state to auto-approve everything:

```go
h, _ := kit.InitHarness(brainkit.HarnessConfig{
    // ...
    InitialState: map[string]any{"yolo": true},
})
```

### Handling Approval

```go
h.Subscribe(func(e brainkit.HarnessEvent) {
    if e.Type == brainkit.EventToolApprovalRequired {
        fmt.Printf("Tool %s wants to run with args: %v\n", e.ToolName, e.Args)
        // Approve, decline, or always-allow the category
        h.RespondToToolApproval(brainkit.ToolApprove)
    }
})
```

---

## Events

The Harness emits 41 event types. Subscribe once to get everything:

```go
h.Subscribe(func(e brainkit.HarnessEvent) {
    switch e.Type {
    case brainkit.EventMessageUpdate:
        fmt.Print(e.Text) // stream text to terminal
    case brainkit.EventToolStart:
        fmt.Printf("\n[tool] %s\n", e.ToolName)
    case brainkit.EventToolEnd:
        fmt.Printf("[done] %s (%dms)\n", e.ToolName, e.Duration)
    case brainkit.EventAgentEnd:
        fmt.Println("\n--- Agent finished ---")
    }
})
```

### Event Categories

| Category | Events | Count |
|----------|--------|-------|
| Agent Lifecycle | agent_start, agent_end | 2 |
| Mode & Model | mode_changed, model_changed | 2 |
| Thread | thread_changed, thread_created, thread_deleted | 3 |
| Message | message_start, message_update, message_end | 3 |
| Tool | tool_start, tool_approval_required, tool_input_*, tool_update, tool_end, shell_output | 8 |
| Interactive | ask_question, plan_approval_required, plan_approved | 3 |
| Subagent | subagent_start, subagent_text_delta, subagent_tool_*, subagent_end | 6 |
| OM | om_status, om_observation_*, om_reflection_*, om_buffering_*, om_activation | 12 |
| Workspace | workspace_status_changed, workspace_ready, workspace_error | 3 |
| System | state_changed, display_state_changed, task_updated, usage_update, info, error | 7 |

See [Events Reference](../api/harness/README.md) for full payload schemas.

---

## Display State

The Harness maintains a canonical display state that reflects the current situation. Updated on every event.

```go
ds := h.GetDisplayState()
fmt.Printf("Running: %v\n", ds.IsRunning)
fmt.Printf("Tokens: %d\n", ds.TokenUsage.TotalTokens)
if ds.PendingApproval != nil {
    fmt.Printf("Waiting for approval: %s\n", ds.PendingApproval.ToolName)
}
```

This is the "single source of truth" for any UI. Instead of tracking individual events, read the display state for the full picture.

---

## Control

```go
// Abort current execution
h.Abort()

// Steer: abort + send new message
h.Steer("Actually, do Y instead")

// Follow-up: queue message after current finishes
h.FollowUp("Also do Z")
```

---

## What's Not Supported Yet

| Feature | Notes |
|---------|-------|
| Dynamic workspace factory | Per-request workspace resolution |
| Observational Memory integration | OM events bridge, but full OM config not wired |
| Custom model catalog | ModelAuthChecker, use count tracking |
| Thread cloning | CloneThread API exists but untested |
