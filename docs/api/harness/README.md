# Harness API Reference

The Harness wraps Mastra's JS orchestration layer, exposing 48 Go methods for gateway integration.

---

## Go Config

```go
type HarnessConfig struct {
    ID                   string                          // required
    ResourceID           string                          // optional, scopes threads
    Modes                []ModeConfig                    // required, at least 1
    StateSchema          map[string]any                  // optional, JSON Schema
    InitialState         map[string]any                  // optional
    Subagents            []HarnessSubagentConfig         // optional
    Tools                []string                        // optional, extra tool names
    Workspace            *WorkspaceHarnessConfig         // optional
    OMConfig             *HarnessOMConfig                // optional
    HeartbeatHandlers    []HeartbeatHandler              // optional, Go-side timers
    ThreadLock           *ThreadLock                     // optional, defaults to mutex
    DefaultPermissions   map[string]string               // optional, category -> policy
    ToolCategoryResolver func(string) string             // optional
    AlwaysAllowTools     []string                        // optional
    ModelAuthChecker     func(string) bool               // optional
    CustomModels         []AvailableModel                // optional
}

type ModeConfig struct {
    ID             string
    Name           string
    Default        bool
    DefaultModelID string
    Color          string
    AgentName      string  // must match agent({ name: "..." })
}
```

---

## Lifecycle

| Method | Signature | Description |
|--------|-----------|-------------|
| `InitHarness` | `(k *Kit) InitHarness(cfg HarnessConfig) (*Harness, error)` | Create and initialize |
| `Close` | `(h *Harness) Close() error` | Stop heartbeats, cleanup |

---

## Messaging (5 methods)

| Method | Signature | Notes |
|--------|-----------|-------|
| `SendMessage` | `(content string, opts ...SendOption) error` | Blocks until agent finishes |
| `Abort` | `() error` | Cancel current execution |
| `Steer` | `(content string, opts ...SendOption) error` | Abort + new message |
| `FollowUp` | `(content string, opts ...SendOption) error` | Queue after current |
| `IsRunning` | `() bool` | Agent currently processing? |

Options: `WithFiles([]FileAttachment)`, `WithRequestContext(map[string]any)`

---

## Threads (8 methods)

| Method | Signature |
|--------|-----------|
| `CreateThread` | `(opts ...ThreadOption) (string, error)` |
| `SwitchThread` | `(threadID string) error` |
| `DeleteThread` | `(threadID string) error` |
| `ListThreads` | `(opts ...ListThreadsOption) ([]HarnessThread, error)` |
| `RenameThread` | `(title string) error` |
| `CloneThread` | `(opts ...CloneOption) (string, error)` |
| `GetCurrentThreadID` | `() string` |
| `ListMessages` | `(opts ...ListMessagesOption) ([]HarnessMessage, error)` |

---

## Modes (4 methods)

| Method | Signature |
|--------|-----------|
| `SwitchMode` | `(modeID string) error` |
| `ListModes` | `() []Mode` |
| `GetCurrentMode` | `() Mode` |
| `GetCurrentModeID` | `() string` |

---

## Models (4 methods)

| Method | Signature |
|--------|-----------|
| `SwitchModel` | `(modelID string, opts ...ModelOption) error` |
| `ListAvailableModels` | `() ([]AvailableModel, error)` |
| `GetCurrentModelID` | `() string` |
| `HasModelSelected` | `() bool` |

Options: `ModelScope("global"|"mode"|"thread")`, `ModelForMode(modeID)`

---

## Permissions (6 methods)

| Method | Signature |
|--------|-----------|
| `RespondToToolApproval` | `(decision ToolApprovalDecision) error` |
| `SetPermissionForCategory` | `(category, policy string) error` |
| `SetPermissionForTool` | `(toolName, policy string) error` |
| `GetPermissionRules` | `() PermissionRules` |
| `GrantSessionCategory` | `(category string) error` |
| `GrantSessionTool` | `(toolName string) error` |

Decisions: `ToolApprove`, `ToolDecline`, `ToolAlwaysAllowCategory`

---

## Interactive (2 methods)

| Method | Signature |
|--------|-----------|
| `RespondToQuestion` | `(questionID, answer string) error` |
| `RespondToPlanApproval` | `(planID string, resp PlanResponse) error` |

---

## State (2 methods)

| Method | Signature |
|--------|-----------|
| `GetState` | `() map[string]any` |
| `SetState` | `(updates map[string]any) error` |

---

## Display State (1 method)

| Method | Signature |
|--------|-----------|
| `GetDisplayState` | `() *DisplayState` |

Returns a deep copy. See guide for DisplayState fields.

---

## Events (1 method)

| Method | Signature |
|--------|-----------|
| `Subscribe` | `(fn func(HarnessEvent)) func()` |

Returns unsubscribe function. 41 event types — see guide for full list.

---

## OM, Subagents, Workspace, Session, Resource

| Category | Methods |
|----------|---------|
| OM (4) | `SwitchObserverModel`, `SwitchReflectorModel`, `GetObserverModelID`, `GetReflectorModelID` |
| Subagents (2) | `SetSubagentModelID(modelID, agentType)`, `GetSubagentModelID(agentType)` |
| Workspace (3) | `HasWorkspace`, `IsWorkspaceReady`, `DestroyWorkspace` |
| Session (2) | `GetSession`, `GetTokenUsage` |
| Resource (3) | `SetResourceID`, `GetResourceID`, `GetKnownResourceIDs` |

---

## Event Types (41)

See `brainkit-maps/references/harness/EVENTS.md` for complete payload schemas.

| Type | Key Fields |
|------|-----------|
| `agent_start` | threadId, runId, content, modeId, modelId |
| `agent_end` | threadId, text, finishReason, usage |
| `message_start` | messageId |
| `message_update` | text, reasoning |
| `message_end` | text, usage |
| `tool_start` | toolCallId, toolName, args |
| `tool_approval_required` | toolCallId, toolName, args, category |
| `tool_end` | toolCallId, toolName, result, isError, duration |
| `ask_question` | questionId, question, options |
| `plan_approval_required` | planId, plan |
| `task_updated` | tasks[] |
| `display_state_changed` | displayState |
