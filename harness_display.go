package brainkit

import "time"

// updateDisplayState applies event-driven state changes to the DisplayState.
// Called synchronously before dispatching events to subscribers.
func updateDisplayState(ds *DisplayState, event HarnessEvent) {
	switch event.Type {
	case EventAgentStart:
		ds.IsRunning = true

	case EventAgentEnd:
		ds.IsRunning = false
		ds.ActiveTools = make(map[string]*ActiveToolState)
		ds.ToolInputBuffers = make(map[string]*ToolInputBuffer)
		ds.PendingApproval = nil
		ds.PendingQuestion = nil
		ds.PendingPlanApproval = nil
		ds.ActiveSubagents = make(map[string]*ActiveSubagentState)
		ds.CurrentMessage = nil

	case EventMessageStart:
		ds.CurrentMessage = &CurrentMessage{
			ID:   event.MessageID,
			Role: "assistant",
		}

	case EventMessageUpdate:
		if ds.CurrentMessage != nil {
			ds.CurrentMessage.Text += event.Text
			ds.CurrentMessage.Reasoning += event.Reasoning
		}

	case EventMessageEnd:
		if ds.CurrentMessage != nil {
			ds.CurrentMessage.Text = event.Text
		}

	case EventToolInputStart:
		ds.ToolInputBuffers[event.ToolCallID] = &ToolInputBuffer{
			ToolName: event.ToolName,
		}

	case EventToolInputDelta:
		if buf, ok := ds.ToolInputBuffers[event.ToolCallID]; ok {
			buf.Text += event.Delta
		}

	case EventToolInputEnd:
		delete(ds.ToolInputBuffers, event.ToolCallID)
		ds.ActiveTools[event.ToolCallID] = &ActiveToolState{
			ToolName:  event.ToolName,
			Args:      event.Args,
			Status:    "running",
			StartTime: time.Now(),
		}

	case EventToolStart:
		ds.ActiveTools[event.ToolCallID] = &ActiveToolState{
			ToolName:  event.ToolName,
			Args:      event.Args,
			Status:    "running",
			StartTime: time.Now(),
		}

	case EventToolApprovalRequired:
		ds.PendingApproval = &PendingApproval{
			ToolCallID: event.ToolCallID,
			ToolName:   event.ToolName,
			Args:       event.Args,
			Category:   event.Category,
		}

	case EventToolUpdate:
		if tool, ok := ds.ActiveTools[event.ToolCallID]; ok {
			tool.Result = event.Result
		}

	case EventToolEnd:
		if tool, ok := ds.ActiveTools[event.ToolCallID]; ok {
			if event.IsError {
				tool.Status = "error"
			} else {
				tool.Status = "completed"
			}
			tool.Result = event.Result
			tool.IsError = event.IsError
			tool.Duration = event.Duration
		}
		if isFileModifyTool(event.ToolName) {
			// Args may be on the event or on the active tool state
			args := event.Args
			if args == nil {
				if tool, ok := ds.ActiveTools[event.ToolCallID]; ok {
					args = tool.Args
				}
			}
			if path := extractPathFromArgs(args); path != "" {
				if mf, ok := ds.ModifiedFiles[path]; ok {
					mf.Operations++
				} else {
					ds.ModifiedFiles[path] = &ModifiedFileState{
						Path:          path,
						Operations:    1,
						FirstModified: time.Now(),
					}
				}
			}
		}

	case EventShellOutput:
		if tool, ok := ds.ActiveTools[event.ToolCallID]; ok {
			tool.ShellOutput = append(tool.ShellOutput, ShellChunk{
				Stream: event.Stream,
				Data:   event.Data,
			})
		}

	case EventAskQuestion:
		ds.PendingQuestion = &PendingQuestion{
			QuestionID: event.QuestionID,
			Question:   event.Question,
			Options:    event.Options,
		}

	case EventPlanApprovalRequired:
		ds.PendingPlanApproval = &PendingPlanApproval{
			PlanID: event.PlanID,
			Plan:   event.Plan,
			Title:  event.Title,
		}

	case EventPlanApproved:
		ds.PendingPlanApproval = nil

	case EventSubagentStart:
		ds.ActiveSubagents[event.ToolCallID] = &ActiveSubagentState{
			AgentType: event.AgentType,
			Task:      event.Task,
			ModelID:   event.ModelID,
			Status:    "running",
		}

	case EventSubagentTextDelta:
		if sa, ok := ds.ActiveSubagents[event.ToolCallID]; ok {
			sa.TextDelta += event.Text
		}

	case EventSubagentToolStart:
		if sa, ok := ds.ActiveSubagents[event.ToolCallID]; ok {
			sa.ToolCalls = append(sa.ToolCalls, SubagentToolCall{
				ToolCallID: event.SubToolCallID,
				ToolName:   event.ToolName,
				Args:       event.Args,
				Status:     "running",
			})
		}

	case EventSubagentToolEnd:
		if sa, ok := ds.ActiveSubagents[event.ToolCallID]; ok {
			for i := range sa.ToolCalls {
				if sa.ToolCalls[i].ToolCallID == event.SubToolCallID {
					if event.IsError {
						sa.ToolCalls[i].Status = "error"
					} else {
						sa.ToolCalls[i].Status = "completed"
					}
					sa.ToolCalls[i].Result = event.Result
					sa.ToolCalls[i].IsError = event.IsError
					break
				}
			}
		}

	case EventSubagentEnd:
		if sa, ok := ds.ActiveSubagents[event.ToolCallID]; ok {
			if event.IsError {
				sa.Status = "error"
			} else {
				sa.Status = "completed"
			}
			sa.Result = event.Text
			sa.Duration = event.Duration
			sa.IsError = event.IsError
		}

	case EventOMStatus:
		ds.OMProgress = &OMProgressState{
			Status:                       event.Status,
			MessagesSinceLastObservation: event.MessagesSinceLastObservation,
			MessagesSinceLastReflection:  event.MessagesSinceLastReflection,
			ObservationThreshold:         event.ObservationThreshold,
			ReflectionThreshold:          event.ReflectionThreshold,
			TotalObservations:            event.TotalObservations,
			TotalReflections:             event.TotalReflections,
		}

	case EventOMObservationStart:
		if ds.OMProgress != nil {
			ds.OMProgress.GeneratingObservation = true
		}

	case EventOMObservationEnd:
		if ds.OMProgress != nil {
			ds.OMProgress.GeneratingObservation = false
			ds.OMProgress.TotalObservations++
		}

	case EventOMObservationFailed:
		if ds.OMProgress != nil {
			ds.OMProgress.GeneratingObservation = false
		}

	case EventOMReflectionStart:
		if ds.OMProgress != nil {
			ds.OMProgress.GeneratingReflection = true
		}

	case EventOMReflectionEnd:
		if ds.OMProgress != nil {
			ds.OMProgress.GeneratingReflection = false
			ds.OMProgress.TotalReflections++
		}

	case EventOMReflectionFailed:
		if ds.OMProgress != nil {
			ds.OMProgress.GeneratingReflection = false
		}

	case EventOMBufferingStart:
		if event.Target == "messages" {
			ds.BufferingMessages = true
		} else if event.Target == "observations" {
			ds.BufferingObservations = true
		}

	case EventOMBufferingEnd, EventOMBufferingFailed:
		if event.Target == "messages" {
			ds.BufferingMessages = false
		} else if event.Target == "observations" {
			ds.BufferingObservations = false
		}

	case EventTaskUpdated:
		ds.PreviousTasks = ds.Tasks
		ds.Tasks = event.Tasks

	case EventUsageUpdate:
		if event.Usage != nil {
			ds.TokenUsage.PromptTokens += event.Usage.PromptTokens
			ds.TokenUsage.CompletionTokens += event.Usage.CompletionTokens
			ds.TokenUsage.TotalTokens += event.Usage.TotalTokens
		}

	case EventThreadChanged, EventThreadCreated:
		resetThreadDisplayState(ds)

	case EventStateChanged:
		for _, key := range event.ChangedKeys {
			if key == "observationThreshold" || key == "reflectionThreshold" {
				if ds.OMProgress != nil {
					if v, ok := event.State["observationThreshold"]; ok {
						if n, ok := v.(float64); ok {
							ds.OMProgress.ObservationThreshold = int(n)
						}
					}
					if v, ok := event.State["reflectionThreshold"]; ok {
						if n, ok := v.(float64); ok {
							ds.OMProgress.ReflectionThreshold = int(n)
						}
					}
				}
				break
			}
		}
	}
}

// resetThreadDisplayState clears all thread-scoped display state.
func resetThreadDisplayState(ds *DisplayState) {
	ds.ActiveTools = make(map[string]*ActiveToolState)
	ds.ToolInputBuffers = make(map[string]*ToolInputBuffer)
	ds.PendingApproval = nil
	ds.PendingQuestion = nil
	ds.PendingPlanApproval = nil
	ds.ActiveSubagents = make(map[string]*ActiveSubagentState)
	ds.CurrentMessage = nil
	ds.ModifiedFiles = make(map[string]*ModifiedFileState)
	ds.Tasks = nil
	ds.PreviousTasks = nil
	ds.OMProgress = nil
	ds.BufferingMessages = false
	ds.BufferingObservations = false
}

func isFileModifyTool(name string) bool {
	return name == "string_replace_lsp" || name == "write_file" || name == "ast_smart_edit"
}

func extractPathFromArgs(args map[string]any) string {
	if args == nil {
		return ""
	}
	if p, ok := args["path"].(string); ok {
		return p
	}
	if p, ok := args["file_path"].(string); ok {
		return p
	}
	return ""
}
