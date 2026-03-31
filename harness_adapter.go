package brainkit

import (
	"context"

	quickjs "github.com/buke/quickjs-go"

	"github.com/brainlet/brainkit/internal/harness"
)

// --- Re-exported harness types (public API) ---

type Harness = harness.Harness
type HarnessConfig = harness.HarnessConfig
type HarnessEvent = harness.HarnessEvent
type HarnessEventType = harness.HarnessEventType
type DisplayState = harness.DisplayState
type TokenUsage = harness.TokenUsage
type HarnessTask = harness.HarnessTask
type Mode = harness.Mode
type HarnessThread = harness.HarnessThread
type HarnessMessage = harness.HarnessMessage
type AvailableModel = harness.AvailableModel
type HarnessSession = harness.HarnessSession
type PermissionRules = harness.PermissionRules
type SessionGrants = harness.SessionGrants
type ToolApprovalDecision = harness.ToolApprovalDecision
type PlanResponse = harness.PlanResponse
type FileAttachment = harness.FileAttachment
type ModeConfig = harness.ModeConfig
type HarnessSubagentConfig = harness.HarnessSubagentConfig
type HarnessOMConfig = harness.HarnessOMConfig
type WorkspaceHarnessConfig = harness.WorkspaceHarnessConfig
type HeartbeatHandler = harness.HeartbeatHandler
type ThreadLock = harness.ThreadLock
type PermissionPolicy = harness.PermissionPolicy
type ToolCategory = harness.ToolCategory
type CurrentMessage = harness.CurrentMessage
type ActiveToolState = harness.ActiveToolState
type PendingApproval = harness.PendingApproval
type PendingQuestion = harness.PendingQuestion
type PendingPlanApproval = harness.PendingPlanApproval
type ActiveSubagentState = harness.ActiveSubagentState

// Option types
type SendOption = harness.SendOption
type ThreadOption = harness.ThreadOption
type ListThreadsOption = harness.ListThreadsOption
type CloneOption = harness.CloneOption
type ListMessagesOption = harness.ListMessagesOption
type ModelOption = harness.ModelOption

// Re-exported constants
const (
	PolicyAllow = harness.PolicyAllow
	PolicyAsk   = harness.PolicyAsk
	PolicyDeny  = harness.PolicyDeny

	CategoryRead    = harness.CategoryRead
	CategoryEdit    = harness.CategoryEdit
	CategoryExecute = harness.CategoryExecute
	CategoryMCP     = harness.CategoryMCP
)

// Re-exported event constants
const (
	EventAgentStart          = harness.EventAgentStart
	EventAgentEnd            = harness.EventAgentEnd
	EventMessageStart        = harness.EventMessageStart
	EventMessageUpdate       = harness.EventMessageUpdate
	EventMessageEnd          = harness.EventMessageEnd
	EventToolStart           = harness.EventToolStart
	EventToolEnd             = harness.EventToolEnd
	EventToolApprovalRequired = harness.EventToolApprovalRequired
	EventAskQuestion         = harness.EventAskQuestion
	EventTaskUpdated         = harness.EventTaskUpdated
	EventUsageUpdate         = harness.EventUsageUpdate
	EventError               = harness.EventError
	EventSubagentStart       = harness.EventSubagentStart
	EventSubagentEnd         = harness.EventSubagentEnd
	EventShellOutput         = harness.EventShellOutput
	EventPlanApprovalRequired = harness.EventPlanApprovalRequired
	EventOMStatus            = harness.EventOMStatus
	EventThreadChanged       = harness.EventThreadChanged
	EventThreadCreated       = harness.EventThreadCreated
	EventThreadDeleted       = harness.EventThreadDeleted
	EventStateChanged        = harness.EventStateChanged
	EventModeChanged         = harness.EventModeChanged

	ToolApprove             = harness.ToolApprove
	ToolDecline             = harness.ToolDecline
	ToolAlwaysAllowCategory = harness.ToolAlwaysAllowCategory
)

// Re-exported functions
var (
	DefaultPermissions = harness.DefaultPermissions
	NewDisplayState    = harness.NewDisplayState
	WithFiles          = harness.WithFiles
	WithRequestContext = harness.WithRequestContext
	WithThreadTitle    = harness.WithThreadTitle
	ForResource        = harness.ForResource
	CloneFrom          = harness.CloneFrom
	CloneWithTitle     = harness.CloneWithTitle
	CloneForResource   = harness.CloneForResource
	ForThread          = harness.ForThread
	WithMessageLimit   = harness.WithMessageLimit
	ModelScope         = harness.ModelScope
	ModelForMode       = harness.ModelForMode
)

// InitHarness creates and initializes a Harness on this Kernel.
func (k *Kernel) InitHarness(cfg harness.HarnessConfig) (*harness.Harness, error) {
	return harness.Init(k, cfg)
}

// --- harness.Runtime implementation ---

func (k *Kernel) BridgeIsEvalBusy() bool {
	return k.bridge.IsEvalBusy()
}

func (k *Kernel) BridgeEval(filename, code string) (*quickjs.Value, error) {
	return k.bridge.Eval(filename, code)
}

func (k *Kernel) BridgeEvalOnJSThread(filename, code string) (string, error) {
	return k.bridge.EvalOnJSThread(filename, code)
}

func (k *Kernel) BridgeContext() *quickjs.Context {
	return k.bridge.Context()
}

func (k *Kernel) BridgeGoContext() context.Context {
	return k.bridge.GoContext()
}
