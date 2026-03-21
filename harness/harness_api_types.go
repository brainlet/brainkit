package harness

// HarnessTask represents a task tracked by the Harness.
type HarnessTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // "pending" | "in_progress" | "completed"
}

// Mode represents a Harness mode (e.g., build, plan, fast).
type Mode struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Default        bool   `json:"default"`
	DefaultModelID string `json:"defaultModelId"`
	Color          string `json:"color"`
}

// HarnessThread represents a conversation thread.
type HarnessThread struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	CreatedAt  string         `json:"createdAt"`
	UpdatedAt  string         `json:"updatedAt"`
	ResourceID string         `json:"resourceId"`
	Metadata   map[string]any `json:"metadata"`
}

// HarnessMessage represents a message in a thread.
type HarnessMessage struct {
	ID        string         `json:"id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	CreatedAt string         `json:"createdAt"`
	ThreadID  string         `json:"threadId"`
	Metadata  map[string]any `json:"metadata"`
}

// AvailableModel describes a model the Harness can use.
type AvailableModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	IsAuth   bool   `json:"isAuth"`
	UseCount int    `json:"useCount"`
	IsCustom bool   `json:"isCustom"`
}

// HarnessSession describes the current Harness session.
type HarnessSession struct {
	CurrentThreadID string          `json:"currentThreadId"`
	CurrentModeID   string          `json:"currentModeId"`
	Threads         []HarnessThread `json:"threads"`
}

// PermissionRules tracks persistent permission policies.
type PermissionRules struct {
	Categories map[string]string `json:"categories"` // category -> policy
	Tools      map[string]string `json:"tools"`      // toolName -> policy
}

// SessionGrants tracks temporary session-level permission grants.
type SessionGrants struct {
	Categories []string `json:"categories"`
	Tools      []string `json:"tools"`
}

// ToolApprovalDecision is the user's response to a tool approval request.
type ToolApprovalDecision string

const (
	ToolApprove             ToolApprovalDecision = "approve"
	ToolDecline             ToolApprovalDecision = "decline"
	ToolAlwaysAllowCategory ToolApprovalDecision = "always_allow_category"
)

// PlanResponse is the user's response to a plan approval request.
type PlanResponse struct {
	Action   string `json:"action"`   // "approve" | "reject"
	Feedback string `json:"feedback"` // optional feedback on rejection
}

// FileAttachment represents a file attached to a message.
type FileAttachment struct {
	URI      string `json:"uri,omitempty"`
	Base64   string `json:"base64,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Name     string `json:"name,omitempty"`
}

// ResourceInfo describes a tracked resource in the Kit.
type ResourceInfo struct {
	Type      string `json:"type"`      // "agent", "tool", "workflow", "wasm", "memory", "harness"
	ID        string `json:"id"`        // unique within type
	Name      string `json:"name"`      // display name
	Source    string `json:"source"`    // .ts filename that created it
	CreatedAt int64  `json:"createdAt"` // unix timestamp ms
}
