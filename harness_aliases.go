package brainkit

import harnesspkg "github.com/brainlet/brainkit/harness"

type Harness = harnesspkg.Harness
type HarnessConfig = harnesspkg.HarnessConfig
type ModeConfig = harnesspkg.ModeConfig
type HarnessSubagentConfig = harnesspkg.HarnessSubagentConfig
type HarnessOMConfig = harnesspkg.HarnessOMConfig
type WorkspaceHarnessConfig = harnesspkg.WorkspaceHarnessConfig
type HeartbeatHandler = harnesspkg.HeartbeatHandler
type ThreadLock = harnesspkg.ThreadLock

type DisplayState = harnesspkg.DisplayState
type CurrentMessage = harnesspkg.CurrentMessage
type ActiveToolState = harnesspkg.ActiveToolState
type ShellChunk = harnesspkg.ShellChunk
type ToolInputBuffer = harnesspkg.ToolInputBuffer
type PendingApproval = harnesspkg.PendingApproval
type PendingQuestion = harnesspkg.PendingQuestion
type PendingPlanApproval = harnesspkg.PendingPlanApproval
type ActiveSubagentState = harnesspkg.ActiveSubagentState
type SubagentToolCall = harnesspkg.SubagentToolCall
type OMProgressState = harnesspkg.OMProgressState
type ModifiedFileState = harnesspkg.ModifiedFileState

type HarnessTask = harnesspkg.HarnessTask
type Mode = harnesspkg.Mode
type HarnessThread = harnesspkg.HarnessThread
type HarnessMessage = harnesspkg.HarnessMessage
type AvailableModel = harnesspkg.AvailableModel
type HarnessSession = harnesspkg.HarnessSession
type PermissionRules = harnesspkg.PermissionRules
type SessionGrants = harnesspkg.SessionGrants
type ToolApprovalDecision = harnesspkg.ToolApprovalDecision
type PlanResponse = harnesspkg.PlanResponse
type FileAttachment = harnesspkg.FileAttachment
type ResourceInfo = harnesspkg.ResourceInfo

type HarnessEventType = harnesspkg.HarnessEventType
type HarnessEvent = harnesspkg.HarnessEvent
type TokenUsage = harnesspkg.TokenUsage

type PermissionPolicy = harnesspkg.PermissionPolicy
type ToolCategory = harnesspkg.ToolCategory

type SendOption = harnesspkg.SendOption
type ThreadOption = harnesspkg.ThreadOption
type ListThreadsOption = harnesspkg.ListThreadsOption
type CloneOption = harnesspkg.CloneOption
type ListMessagesOption = harnesspkg.ListMessagesOption
type ModelOption = harnesspkg.ModelOption

const (
	ToolApprove             ToolApprovalDecision = harnesspkg.ToolApprove
	ToolDecline             ToolApprovalDecision = harnesspkg.ToolDecline
	ToolAlwaysAllowCategory ToolApprovalDecision = harnesspkg.ToolAlwaysAllowCategory

	PolicyAllow PermissionPolicy = harnesspkg.PolicyAllow
	PolicyAsk   PermissionPolicy = harnesspkg.PolicyAsk
	PolicyDeny  PermissionPolicy = harnesspkg.PolicyDeny

	CategoryRead    ToolCategory = harnesspkg.CategoryRead
	CategoryEdit    ToolCategory = harnesspkg.CategoryEdit
	CategoryExecute ToolCategory = harnesspkg.CategoryExecute
	CategoryMCP     ToolCategory = harnesspkg.CategoryMCP

	EventAgentStart            HarnessEventType = harnesspkg.EventAgentStart
	EventAgentEnd              HarnessEventType = harnesspkg.EventAgentEnd
	EventModeChanged           HarnessEventType = harnesspkg.EventModeChanged
	EventModelChanged          HarnessEventType = harnesspkg.EventModelChanged
	EventThreadChanged         HarnessEventType = harnesspkg.EventThreadChanged
	EventThreadCreated         HarnessEventType = harnesspkg.EventThreadCreated
	EventThreadDeleted         HarnessEventType = harnesspkg.EventThreadDeleted
	EventMessageStart          HarnessEventType = harnesspkg.EventMessageStart
	EventMessageUpdate         HarnessEventType = harnesspkg.EventMessageUpdate
	EventMessageEnd            HarnessEventType = harnesspkg.EventMessageEnd
	EventToolStart             HarnessEventType = harnesspkg.EventToolStart
	EventToolApprovalRequired  HarnessEventType = harnesspkg.EventToolApprovalRequired
	EventToolInputStart        HarnessEventType = harnesspkg.EventToolInputStart
	EventToolInputDelta        HarnessEventType = harnesspkg.EventToolInputDelta
	EventToolInputEnd          HarnessEventType = harnesspkg.EventToolInputEnd
	EventToolUpdate            HarnessEventType = harnesspkg.EventToolUpdate
	EventToolEnd               HarnessEventType = harnesspkg.EventToolEnd
	EventShellOutput           HarnessEventType = harnesspkg.EventShellOutput
	EventAskQuestion           HarnessEventType = harnesspkg.EventAskQuestion
	EventPlanApprovalRequired  HarnessEventType = harnesspkg.EventPlanApprovalRequired
	EventPlanApproved          HarnessEventType = harnesspkg.EventPlanApproved
	EventSubagentStart         HarnessEventType = harnesspkg.EventSubagentStart
	EventSubagentTextDelta     HarnessEventType = harnesspkg.EventSubagentTextDelta
	EventSubagentToolStart     HarnessEventType = harnesspkg.EventSubagentToolStart
	EventSubagentToolEnd       HarnessEventType = harnesspkg.EventSubagentToolEnd
	EventSubagentEnd           HarnessEventType = harnesspkg.EventSubagentEnd
	EventSubagentModelChanged  HarnessEventType = harnesspkg.EventSubagentModelChanged
	EventOMStatus              HarnessEventType = harnesspkg.EventOMStatus
	EventOMObservationStart    HarnessEventType = harnesspkg.EventOMObservationStart
	EventOMObservationEnd      HarnessEventType = harnesspkg.EventOMObservationEnd
	EventOMObservationFailed   HarnessEventType = harnesspkg.EventOMObservationFailed
	EventOMReflectionStart     HarnessEventType = harnesspkg.EventOMReflectionStart
	EventOMReflectionEnd       HarnessEventType = harnesspkg.EventOMReflectionEnd
	EventOMReflectionFailed    HarnessEventType = harnesspkg.EventOMReflectionFailed
	EventOMBufferingStart      HarnessEventType = harnesspkg.EventOMBufferingStart
	EventOMBufferingEnd        HarnessEventType = harnesspkg.EventOMBufferingEnd
	EventOMBufferingFailed     HarnessEventType = harnesspkg.EventOMBufferingFailed
	EventOMActivation          HarnessEventType = harnesspkg.EventOMActivation
	EventOMModelChanged        HarnessEventType = harnesspkg.EventOMModelChanged
	EventWorkspaceStatusChanged HarnessEventType = harnesspkg.EventWorkspaceStatusChanged
	EventWorkspaceReady        HarnessEventType = harnesspkg.EventWorkspaceReady
	EventWorkspaceError        HarnessEventType = harnesspkg.EventWorkspaceError
	EventStateChanged          HarnessEventType = harnesspkg.EventStateChanged
	EventDisplayStateChanged   HarnessEventType = harnesspkg.EventDisplayStateChanged
	EventTaskUpdated           HarnessEventType = harnesspkg.EventTaskUpdated
	EventUsageUpdate           HarnessEventType = harnesspkg.EventUsageUpdate
	EventFollowUpQueued        HarnessEventType = harnesspkg.EventFollowUpQueued
	EventInfo                  HarnessEventType = harnesspkg.EventInfo
	EventError                 HarnessEventType = harnesspkg.EventError
)

func NewDisplayState() *DisplayState {
	return harnesspkg.NewDisplayState()
}

func DefaultPermissions() map[ToolCategory]PermissionPolicy {
	return harnesspkg.DefaultPermissions()
}

func StateSchemaOf[T any]() map[string]any {
	return harnesspkg.StateSchemaOf[T]()
}

func WithFiles(files []FileAttachment) SendOption {
	return harnesspkg.WithFiles(files)
}

func WithRequestContext(ctx map[string]any) SendOption {
	return harnesspkg.WithRequestContext(ctx)
}

func WithThreadTitle(title string) ThreadOption {
	return harnesspkg.WithThreadTitle(title)
}

func WithThreadResourceID(id string) ThreadOption {
	return harnesspkg.WithThreadResourceID(id)
}

func ForResource(resourceID string) ListThreadsOption {
	return harnesspkg.ForResource(resourceID)
}

func CloneFrom(id string) CloneOption {
	return harnesspkg.CloneFrom(id)
}

func CloneWithTitle(title string) CloneOption {
	return harnesspkg.CloneWithTitle(title)
}

func CloneForResource(id string) CloneOption {
	return harnesspkg.CloneForResource(id)
}

func ForThread(id string) ListMessagesOption {
	return harnesspkg.ForThread(id)
}

func WithMessageLimit(n int) ListMessagesOption {
	return harnesspkg.WithMessageLimit(n)
}

func ModelScope(scope string) ModelOption {
	return harnesspkg.ModelScope(scope)
}

func ModelForMode(modeID string) ModelOption {
	return harnesspkg.ModelForMode(modeID)
}
