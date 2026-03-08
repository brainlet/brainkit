// Ported from: packages/core/src/harness/tools.ts
package harness

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
)

// ---------------------------------------------------------------------------
// Counters for generating unique IDs
// ---------------------------------------------------------------------------

var questionCounter int64
var planCounter int64

// ---------------------------------------------------------------------------
// AskUser tool
// ---------------------------------------------------------------------------

// AskUserInput holds the input parameters for the ask_user tool.
type AskUserInput struct {
	Question string           `json:"question"`
	Options  []QuestionOption `json:"options,omitempty"`
}

// AskUserResult holds the result of the ask_user tool execution.
type AskUserResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}

// AskUser is the built-in harness tool that asks the user a question and waits
// for their response. Supports single-select options and free-text input.
func AskUser(input AskUserInput, harnessCtx *HarnessRequestContext) AskUserResult {
	if harnessCtx == nil || harnessCtx.EmitEvent == nil || harnessCtx.RegisterQuestion == nil {
		optionsStr := ""
		if len(input.Options) > 0 {
			labels := make([]string, len(input.Options))
			for i, o := range input.Options {
				labels[i] = o.Label
			}
			optionsStr = "\nOptions: " + strings.Join(labels, ", ")
		}
		return AskUserResult{
			Content: fmt.Sprintf("[Question for user]: %s%s", input.Question, optionsStr),
			IsError: false,
		}
	}

	counter := atomic.AddInt64(&questionCounter, 1)
	questionID := fmt.Sprintf("q_%d_%d", counter, 0) // timestamp elided in Go

	answerCh := make(chan string, 1)
	harnessCtx.RegisterQuestion(questionID, func(answer string) {
		answerCh <- answer
	})

	harnessCtx.EmitEvent(HarnessEvent{
		Type:       "ask_question",
		QuestionID: questionID,
		Question:   input.Question,
		Options:    input.Options,
	})

	answer := <-answerCh
	return AskUserResult{
		Content: fmt.Sprintf("User answered: %s", answer),
		IsError: false,
	}
}

// ---------------------------------------------------------------------------
// SubmitPlan tool
// ---------------------------------------------------------------------------

// SubmitPlanInput holds the input parameters for the submit_plan tool.
type SubmitPlanInput struct {
	Title string `json:"title,omitempty"`
	Plan  string `json:"plan"`
}

// SubmitPlanResult holds the result of the submit_plan tool execution.
type SubmitPlanResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}

// SubmitPlan is the built-in harness tool that submits a plan for user review.
// The plan renders in the UI with approve/reject options.
func SubmitPlan(input SubmitPlanInput, harnessCtx *HarnessRequestContext) SubmitPlanResult {
	title := input.Title
	if title == "" {
		title = "Implementation Plan"
	}

	if harnessCtx == nil || harnessCtx.EmitEvent == nil || harnessCtx.RegisterPlanApproval == nil {
		return SubmitPlanResult{
			Content: fmt.Sprintf("[Plan submitted for review]\n\nTitle: %s\n\n%s", title, input.Plan),
			IsError: false,
		}
	}

	counter := atomic.AddInt64(&planCounter, 1)
	planID := fmt.Sprintf("plan_%d_%d", counter, 0)

	type planResult struct {
		Action   string
		Feedback string
	}
	resultCh := make(chan planResult, 1)

	harnessCtx.RegisterPlanApproval(planID, func(action, feedback string) {
		resultCh <- planResult{Action: action, Feedback: feedback}
	})

	harnessCtx.EmitEvent(HarnessEvent{
		Type:   "plan_approval_required",
		PlanID: planID,
		Title:  title,
		Plan:   input.Plan,
	})

	result := <-resultCh

	if result.Action == "approved" {
		return SubmitPlanResult{
			Content: "Plan approved. Proceed with implementation following the approved plan.",
			IsError: false,
		}
	}

	feedback := ""
	if result.Feedback != "" {
		feedback = fmt.Sprintf("\n\nUser feedback: %s", result.Feedback)
	}
	return SubmitPlanResult{
		Content: fmt.Sprintf("Plan was not approved. The user wants revisions.%s\n\nPlease revise the plan based on the feedback and submit again with submit_plan.", feedback),
		IsError: false,
	}
}

// ---------------------------------------------------------------------------
// TaskWrite tool
// ---------------------------------------------------------------------------

// TaskWriteInput holds the input parameters for the task_write tool.
type TaskWriteInput struct {
	Tasks []TaskItem `json:"tasks"`
}

// TaskWriteResult holds the result of the task_write tool execution.
type TaskWriteResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}

// TaskWrite is the built-in harness tool for managing a structured task list.
// Full-replacement semantics: each call replaces the entire task list.
func TaskWrite(input TaskWriteInput, harnessCtx *HarnessRequestContext) TaskWriteResult {
	if harnessCtx != nil {
		if harnessCtx.SetState != nil {
			_ = harnessCtx.SetState(map[string]any{"tasks": input.Tasks})
		}
		if harnessCtx.EmitEvent != nil {
			harnessCtx.EmitEvent(HarnessEvent{
				Type:  "task_updated",
				Tasks: input.Tasks,
			})
		}
	}

	completed := 0
	var inProgressTask *TaskItem
	for i := range input.Tasks {
		if input.Tasks[i].Status == "completed" {
			completed++
		}
		if input.Tasks[i].Status == "in_progress" && inProgressTask == nil {
			inProgressTask = &input.Tasks[i]
		}
	}

	summary := fmt.Sprintf("Tasks updated: [%d/%d completed]", completed, len(input.Tasks))
	if inProgressTask != nil {
		summary += fmt.Sprintf("\nCurrently: %s", inProgressTask.ActiveForm)
	}

	return TaskWriteResult{
		Content: summary,
		IsError: false,
	}
}

// ---------------------------------------------------------------------------
// TaskCheck tool
// ---------------------------------------------------------------------------

// TaskCheckResult holds the result of the task_check tool execution.
type TaskCheckResult struct {
	Content string `json:"content"`
	IsError bool   `json:"isError"`
}

// TaskCheck is the built-in harness tool for checking the completion status
// of the current task list.
func TaskCheck(harnessCtx *HarnessRequestContext) TaskCheckResult {
	if harnessCtx == nil {
		return TaskCheckResult{
			Content: "Unable to access task list (no harness context)",
			IsError: true,
		}
	}

	var tasks []TaskItem
	var state map[string]any
	if harnessCtx.GetState != nil {
		state = harnessCtx.GetState()
	} else {
		state = harnessCtx.State
	}

	if rawTasks, ok := state["tasks"]; ok {
		if typed, ok := rawTasks.([]TaskItem); ok {
			tasks = typed
		}
	}

	if len(tasks) == 0 {
		return TaskCheckResult{
			Content: "No tasks found. Consider using task_write to create a task list for complex work.",
			IsError: false,
		}
	}

	var completedTasks, inProgressTasks, pendingTasks []TaskItem
	for _, t := range tasks {
		switch t.Status {
		case "completed":
			completedTasks = append(completedTasks, t)
		case "in_progress":
			inProgressTasks = append(inProgressTasks, t)
		case "pending":
			pendingTasks = append(pendingTasks, t)
		}
	}

	allDone := len(inProgressTasks) == 0 && len(pendingTasks) == 0
	allDoneStr := "NO"
	if allDone {
		allDoneStr = "YES"
	}

	response := fmt.Sprintf("Task Status: [%d/%d completed]\n- Completed: %d\n- In Progress: %d\n- Pending: %d\n\nAll tasks completed: %s",
		len(completedTasks), len(tasks), len(completedTasks), len(inProgressTasks), len(pendingTasks), allDoneStr)

	if !allDone {
		response += "\n\nIncomplete tasks:"
		if len(inProgressTasks) > 0 {
			response += "\n\nIn Progress:"
			for _, t := range inProgressTasks {
				response += fmt.Sprintf("\n- %s", t.Content)
			}
		}
		if len(pendingTasks) > 0 {
			response += "\n\nPending:"
			for _, t := range pendingTasks {
				response += fmt.Sprintf("\n- %s", t.Content)
			}
		}
		response += "\n\nContinue working on these tasks before ending."
	}

	return TaskCheckResult{
		Content: response,
		IsError: false,
	}
}

// ---------------------------------------------------------------------------
// Subagent meta parsing
// ---------------------------------------------------------------------------

// SubagentMeta holds parsed metadata from a subagent result string.
type SubagentMeta struct {
	Text      string
	ModelID   string
	DurationMs int
	ToolCalls  []SubagentToolCall
}

// subagentMetaRegex matches the <subagent-meta .../> tag appended to results.
var subagentMetaRegex = regexp.MustCompile(`\n<subagent-meta modelId="([^"]*)" durationMs="(\d+)" tools="([^"]*)" />$`)

// ParseSubagentMeta parses subagent metadata from a tool result string.
// Returns the metadata and the cleaned result text (without the tag).
func ParseSubagentMeta(content string) SubagentMeta {
	match := subagentMetaRegex.FindStringSubmatchIndex(content)
	if match == nil {
		return SubagentMeta{Text: content}
	}

	text := content[:match[0]]
	modelID := content[match[2]:match[3]]
	durationStr := content[match[4]:match[5]]
	toolsStr := content[match[6]:match[7]]

	durationMs, _ := strconv.Atoi(durationStr)

	var toolCalls []SubagentToolCall
	if toolsStr != "" {
		entries := strings.Split(toolsStr, ",")
		for _, entry := range entries {
			if entry == "" {
				continue
			}
			parts := strings.SplitN(entry, ":", 2)
			name := parts[0]
			isErr := len(parts) > 1 && parts[1] == "err"
			toolCalls = append(toolCalls, SubagentToolCall{
				Name:    name,
				IsError: isErr,
			})
		}
	}

	return SubagentMeta{
		Text:       text,
		ModelID:    modelID,
		DurationMs: durationMs,
		ToolCalls:  toolCalls,
	}
}

// BuildSubagentMeta builds a metadata tag appended to subagent results.
func BuildSubagentMeta(modelID string, durationMs int, toolCalls []SubagentToolCall) string {
	var toolStrs []string
	for _, tc := range toolCalls {
		status := "ok"
		if tc.IsError {
			status = "err"
		}
		toolStrs = append(toolStrs, fmt.Sprintf("%s:%s", tc.Name, status))
	}
	tools := strings.Join(toolStrs, ",")
	return fmt.Sprintf("\n<subagent-meta modelId=\"%s\" durationMs=\"%d\" tools=\"%s\" />", modelID, durationMs, tools)
}
