package harness

import "time"

// DisplayState is the canonical UI state, updated on every event.
type DisplayState struct {
	IsRunning             bool                            `json:"isRunning"`
	CurrentMessage        *CurrentMessage                 `json:"currentMessage"`
	TokenUsage            TokenUsage                      `json:"tokenUsage"`
	ActiveTools           map[string]*ActiveToolState     `json:"activeTools"`
	ToolInputBuffers      map[string]*ToolInputBuffer     `json:"toolInputBuffers"`
	PendingApproval       *PendingApproval                `json:"pendingApproval"`
	PendingQuestion       *PendingQuestion                `json:"pendingQuestion"`
	PendingPlanApproval   *PendingPlanApproval            `json:"pendingPlanApproval"`
	ActiveSubagents       map[string]*ActiveSubagentState `json:"activeSubagents"`
	OMProgress            *OMProgressState                `json:"omProgress"`
	BufferingMessages     bool                            `json:"bufferingMessages"`
	BufferingObservations bool                            `json:"bufferingObservations"`
	ModifiedFiles         map[string]*ModifiedFileState   `json:"modifiedFiles"`
	Tasks                 []HarnessTask                   `json:"tasks"`
	PreviousTasks         []HarnessTask                   `json:"previousTasks"`
}

// NewDisplayState creates a fresh display state with initialized maps.
func NewDisplayState() *DisplayState {
	return &DisplayState{
		ActiveTools:      make(map[string]*ActiveToolState),
		ToolInputBuffers: make(map[string]*ToolInputBuffer),
		ActiveSubagents:  make(map[string]*ActiveSubagentState),
		ModifiedFiles:    make(map[string]*ModifiedFileState),
	}
}

type CurrentMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Text      string `json:"text"`
	Reasoning string `json:"reasoning"`
}

type ActiveToolState struct {
	ToolName    string         `json:"toolName"`
	Args        map[string]any `json:"args"`
	Status      string         `json:"status"` // "running" | "completed" | "error"
	Result      any            `json:"result"`
	IsError     bool           `json:"isError"`
	StartTime   time.Time      `json:"startTime"`
	Duration    int            `json:"duration"`
	ShellOutput []ShellChunk   `json:"shellOutput"`
}

type ShellChunk struct {
	Stream string `json:"stream"` // "stdout" | "stderr"
	Data   string `json:"data"`
}

type ToolInputBuffer struct {
	ToolName string `json:"toolName"`
	Text     string `json:"text"`
}

type PendingApproval struct {
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Args       map[string]any `json:"args"`
	Category   string         `json:"category"`
}

type PendingQuestion struct {
	QuestionID string   `json:"questionId"`
	Question   string   `json:"question"`
	Options    []string `json:"options"`
}

type PendingPlanApproval struct {
	PlanID string `json:"planId"`
	Plan   string `json:"plan"`
	Title  string `json:"title"`
}

type ActiveSubagentState struct {
	AgentType string             `json:"agentType"`
	Task      string             `json:"task"`
	ModelID   string             `json:"modelId"`
	Status    string             `json:"status"`
	TextDelta string             `json:"textDelta"`
	ToolCalls []SubagentToolCall `json:"toolCalls"`
	Result    string             `json:"result"`
	Duration  int                `json:"duration"`
	IsError   bool               `json:"isError"`
}

type SubagentToolCall struct {
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Args       map[string]any `json:"args"`
	Result     any            `json:"result"`
	IsError    bool           `json:"isError"`
	Status     string         `json:"status"`
}

type OMProgressState struct {
	Status                       string `json:"status"`
	MessagesSinceLastObservation int    `json:"messagesSinceLastObservation"`
	MessagesSinceLastReflection  int    `json:"messagesSinceLastReflection"`
	ObservationThreshold         int    `json:"observationThreshold"`
	ReflectionThreshold          int    `json:"reflectionThreshold"`
	TotalObservations            int    `json:"totalObservations"`
	TotalReflections             int    `json:"totalReflections"`
	BufferedCount                int    `json:"bufferedCount"`
	GeneratingObservation        bool   `json:"generatingObservation"`
	GeneratingReflection         bool   `json:"generatingReflection"`
	CurrentCycle                 string `json:"currentCycle"`
}

type ModifiedFileState struct {
	Path          string    `json:"path"`
	Operations    int       `json:"operations"`
	FirstModified time.Time `json:"firstModified"`
}

// clone returns a deep copy of the DisplayState.
func (ds *DisplayState) clone() *DisplayState {
	if ds == nil {
		return nil
	}
	c := *ds
	c.ActiveTools = make(map[string]*ActiveToolState, len(ds.ActiveTools))
	for k, v := range ds.ActiveTools {
		vv := *v
		vv.ShellOutput = append([]ShellChunk(nil), v.ShellOutput...)
		c.ActiveTools[k] = &vv
	}
	c.ToolInputBuffers = make(map[string]*ToolInputBuffer, len(ds.ToolInputBuffers))
	for k, v := range ds.ToolInputBuffers {
		vv := *v
		c.ToolInputBuffers[k] = &vv
	}
	c.ActiveSubagents = make(map[string]*ActiveSubagentState, len(ds.ActiveSubagents))
	for k, v := range ds.ActiveSubagents {
		vv := *v
		vv.ToolCalls = append([]SubagentToolCall(nil), v.ToolCalls...)
		c.ActiveSubagents[k] = &vv
	}
	c.ModifiedFiles = make(map[string]*ModifiedFileState, len(ds.ModifiedFiles))
	for k, v := range ds.ModifiedFiles {
		vv := *v
		c.ModifiedFiles[k] = &vv
	}
	c.Tasks = append([]HarnessTask(nil), ds.Tasks...)
	c.PreviousTasks = append([]HarnessTask(nil), ds.PreviousTasks...)
	if ds.CurrentMessage != nil {
		cm := *ds.CurrentMessage
		c.CurrentMessage = &cm
	}
	if ds.PendingApproval != nil {
		pa := *ds.PendingApproval
		c.PendingApproval = &pa
	}
	if ds.PendingQuestion != nil {
		pq := *ds.PendingQuestion
		c.PendingQuestion = &pq
	}
	if ds.PendingPlanApproval != nil {
		pp := *ds.PendingPlanApproval
		c.PendingPlanApproval = &pp
	}
	if ds.OMProgress != nil {
		om := *ds.OMProgress
		c.OMProgress = &om
	}
	return &c
}
