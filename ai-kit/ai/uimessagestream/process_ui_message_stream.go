// Ported from: packages/ai/src/ui/process-ui-message-stream.ts
//
// This file contains the streaming UI message state machine that processes
// UIMessageChunk values and accumulates them into a UIMessage. It is used by
// both handleUIMessageStreamFinish (server side) and readUIMessageStream
// (client side).
package uimessagestream

import (
	"encoding/json"
	"strings"
)

// StreamingUIMessageState holds the mutable state accumulated while processing
// a UI message stream.
type StreamingUIMessageState struct {
	Message              UIMessage
	ActiveTextParts      map[string]*UIMessagePart
	ActiveReasoningParts map[string]*UIMessagePart
	PartialToolCalls     map[string]*partialToolCall
	FinishReason         string
}

type partialToolCall struct {
	Text     string
	Index    int
	ToolName string
	Dynamic  bool
	Title    string
}

// CreateStreamingUIMessageState creates a new StreamingUIMessageState.
func CreateStreamingUIMessageState(messageID string, lastMessage *UIMessage) *StreamingUIMessageState {
	var msg UIMessage
	if lastMessage != nil && lastMessage.Role == "assistant" {
		msg = *lastMessage
	} else {
		msg = UIMessage{
			ID:    messageID,
			Role:  "assistant",
			Parts: []UIMessagePart{},
		}
	}

	return &StreamingUIMessageState{
		Message:              msg,
		ActiveTextParts:      make(map[string]*UIMessagePart),
		ActiveReasoningParts: make(map[string]*UIMessagePart),
		PartialToolCalls:     make(map[string]*partialToolCall),
	}
}

// ProcessUIMessageStreamOptions configures ProcessUIMessageStream.
type ProcessUIMessageStreamOptions struct {
	// Input is the channel of UIMessageChunk values to process.
	Input <-chan UIMessageChunk

	// RunUpdateMessageJob executes a job that modifies the streaming state.
	// The write callback should be called when the state change should be
	// emitted to the output.
	RunUpdateMessageJob func(job func(state *StreamingUIMessageState, write func()) error) error

	// OnError is called when a processing error occurs.
	OnError func(err error)
}

// ProcessUIMessageStream reads chunks from Input, applies them to the streaming
// state via RunUpdateMessageJob, and emits processed chunks to the returned channel.
//
// This is the Go equivalent of the TypeScript processUIMessageStream which returns
// a ReadableStream piped through a TransformStream.
func ProcessUIMessageStream(opts ProcessUIMessageStreamOptions) <-chan UIMessageChunk {
	output := make(chan UIMessageChunk)
	go func() {
		defer close(output)
		for chunk := range opts.Input {
			chunkCopy := chunk
			err := opts.RunUpdateMessageJob(func(state *StreamingUIMessageState, write func()) error {
				return processChunk(state, chunkCopy, write, opts.OnError)
			})
			if err != nil && opts.OnError != nil {
				opts.OnError(err)
			}
			output <- chunkCopy
		}
	}()
	return output
}

func processChunk(state *StreamingUIMessageState, chunk UIMessageChunk, write func(), onError func(error)) error {
	switch chunk.Type {
	case "text-start":
		textPart := &UIMessagePart{
			Type:             "text",
			Text:             "",
			ProviderMetadata: chunk.ProviderMetadata,
			State:            "streaming",
		}
		state.ActiveTextParts[chunk.ID] = textPart
		state.Message.Parts = append(state.Message.Parts, *textPart)
		// Keep pointer to the appended element
		state.ActiveTextParts[chunk.ID] = &state.Message.Parts[len(state.Message.Parts)-1]
		write()

	case "text-delta":
		textPart, ok := state.ActiveTextParts[chunk.ID]
		if !ok {
			return nil // In TS this throws UIMessageStreamError; we silently skip
		}
		textPart.Text += chunk.Delta
		if chunk.ProviderMetadata != nil {
			textPart.ProviderMetadata = chunk.ProviderMetadata
		}
		write()

	case "text-end":
		textPart, ok := state.ActiveTextParts[chunk.ID]
		if !ok {
			return nil
		}
		textPart.State = "done"
		if chunk.ProviderMetadata != nil {
			textPart.ProviderMetadata = chunk.ProviderMetadata
		}
		delete(state.ActiveTextParts, chunk.ID)
		write()

	case "reasoning-start":
		reasoningPart := &UIMessagePart{
			Type:             "reasoning",
			Text:             "",
			ProviderMetadata: chunk.ProviderMetadata,
			State:            "streaming",
		}
		state.Message.Parts = append(state.Message.Parts, *reasoningPart)
		state.ActiveReasoningParts[chunk.ID] = &state.Message.Parts[len(state.Message.Parts)-1]
		write()

	case "reasoning-delta":
		rp, ok := state.ActiveReasoningParts[chunk.ID]
		if !ok {
			return nil
		}
		rp.Text += chunk.Delta
		if chunk.ProviderMetadata != nil {
			rp.ProviderMetadata = chunk.ProviderMetadata
		}
		write()

	case "reasoning-end":
		rp, ok := state.ActiveReasoningParts[chunk.ID]
		if !ok {
			return nil
		}
		rp.State = "done"
		if chunk.ProviderMetadata != nil {
			rp.ProviderMetadata = chunk.ProviderMetadata
		}
		delete(state.ActiveReasoningParts, chunk.ID)
		write()

	case "file":
		part := UIMessagePart{
			Type:      "file",
			MediaType: chunk.MediaType,
			URL:       chunk.URL,
		}
		if chunk.ProviderMetadata != nil {
			part.ProviderMetadata = chunk.ProviderMetadata
		}
		state.Message.Parts = append(state.Message.Parts, part)
		write()

	case "source-url":
		state.Message.Parts = append(state.Message.Parts, UIMessagePart{
			Type:             "source-url",
			SourceID:         chunk.SourceID,
			URL:              chunk.URL,
			Title:            chunk.Title,
			ProviderMetadata: chunk.ProviderMetadata,
		})
		write()

	case "source-document":
		state.Message.Parts = append(state.Message.Parts, UIMessagePart{
			Type:             "source-document",
			SourceID:         chunk.SourceID,
			MediaType:        chunk.MediaType,
			Title:            chunk.Title,
			Filename:         chunk.Filename,
			ProviderMetadata: chunk.ProviderMetadata,
		})
		write()

	case "tool-input-start":
		ptc := &partialToolCall{
			Text:     "",
			ToolName: chunk.ToolName,
			Index:    countToolParts(state.Message.Parts),
			Title:    chunk.Title,
		}
		if chunk.Dynamic != nil && *chunk.Dynamic {
			ptc.Dynamic = true
		}
		state.PartialToolCalls[chunk.ToolCallID] = ptc

		part := UIMessagePart{
			ToolCallID:       chunk.ToolCallId(),
			ToolName:         chunk.ToolName,
			State:            "input-streaming",
			ProviderExecuted: chunk.ProviderExecuted,
			Title:            chunk.Title,
		}
		if chunk.ProviderMetadata != nil {
			part.CallProviderMetadata = chunk.ProviderMetadata
		}
		if ptc.Dynamic {
			part.Type = "dynamic-tool"
		} else {
			part.Type = "tool-" + chunk.ToolName
		}
		state.Message.Parts = append(state.Message.Parts, part)
		write()

	case "tool-input-delta":
		ptc, ok := state.PartialToolCalls[chunk.ToolCallID]
		if !ok {
			return nil
		}
		ptc.Text += chunk.InputTextDelta
		// Update the tool part's input with the partial text
		updateToolPartInput(state, chunk.ToolCallID, ptc)
		write()

	case "tool-input-available":
		isDynamic := chunk.Dynamic != nil && *chunk.Dynamic
		part := findOrCreateToolPart(state, chunk.ToolCallID, chunk.ToolName, isDynamic)
		part.State = "input-available"
		part.Input = chunk.Input
		part.ProviderExecuted = chunk.ProviderExecuted
		if chunk.ProviderMetadata != nil {
			part.CallProviderMetadata = chunk.ProviderMetadata
		}
		if chunk.Title != "" {
			part.Title = chunk.Title
		}
		write()

	case "tool-input-error":
		isDynamic := chunk.Dynamic != nil && *chunk.Dynamic
		// Check existing part to determine dynamic status
		for i := range state.Message.Parts {
			p := &state.Message.Parts[i]
			if p.ToolCallID == chunk.ToolCallID {
				isDynamic = p.Type == "dynamic-tool"
				break
			}
		}
		part := findOrCreateToolPart(state, chunk.ToolCallID, chunk.ToolName, isDynamic)
		part.State = "output-error"
		part.ErrorText = chunk.ErrorText
		part.ProviderExecuted = chunk.ProviderExecuted
		if !isDynamic {
			part.RawInput = chunk.Input
		} else {
			part.Input = chunk.Input
		}
		if chunk.ProviderMetadata != nil {
			part.CallProviderMetadata = chunk.ProviderMetadata
		}
		write()

	case "tool-approval-request":
		for i := range state.Message.Parts {
			p := &state.Message.Parts[i]
			if p.ToolCallID == chunk.ToolCallID {
				p.State = "approval-requested"
				p.Approval = &ToolApproval{ID: chunk.ApprovalID}
				break
			}
		}
		write()

	case "tool-output-denied":
		for i := range state.Message.Parts {
			p := &state.Message.Parts[i]
			if p.ToolCallID == chunk.ToolCallID {
				p.State = "output-denied"
				break
			}
		}
		write()

	case "tool-output-available":
		for i := range state.Message.Parts {
			p := &state.Message.Parts[i]
			if p.ToolCallID == chunk.ToolCallID {
				p.State = "output-available"
				p.Output = chunk.Output
				p.ProviderExecuted = chunk.ProviderExecuted
				p.Preliminary = chunk.Preliminary
				break
			}
		}
		write()

	case "tool-output-error":
		for i := range state.Message.Parts {
			p := &state.Message.Parts[i]
			if p.ToolCallID == chunk.ToolCallID {
				p.State = "output-error"
				p.ErrorText = chunk.ErrorText
				p.ProviderExecuted = chunk.ProviderExecuted
				break
			}
		}
		write()

	case "start-step":
		state.Message.Parts = append(state.Message.Parts, UIMessagePart{Type: "step-start"})

	case "finish-step":
		state.ActiveTextParts = make(map[string]*UIMessagePart)
		state.ActiveReasoningParts = make(map[string]*UIMessagePart)

	case "start":
		if chunk.MessageID != "" {
			state.Message.ID = chunk.MessageID
		}
		if chunk.MessageMetadata != nil {
			state.Message.Metadata = mergeMetadata(state.Message.Metadata, chunk.MessageMetadata)
		}
		if chunk.MessageID != "" || chunk.MessageMetadata != nil {
			write()
		}

	case "finish":
		if chunk.FinishReason != "" {
			state.FinishReason = chunk.FinishReason
		}
		if chunk.MessageMetadata != nil {
			state.Message.Metadata = mergeMetadata(state.Message.Metadata, chunk.MessageMetadata)
			write()
		}

	case "message-metadata":
		if chunk.MessageMetadata != nil {
			state.Message.Metadata = mergeMetadata(state.Message.Metadata, chunk.MessageMetadata)
			write()
		}

	case "error":
		if onError != nil {
			onError(newStreamError(chunk.ErrorText))
		}

	default:
		if strings.HasPrefix(chunk.Type, "data-") {
			// Data chunk handling
			if chunk.Transient != nil && *chunk.Transient {
				// Transient parts are not added to the message state
				break
			}
			existingIdx := -1
			if chunk.ID != "" {
				for i, p := range state.Message.Parts {
					if p.Type == chunk.Type && p.ID == chunk.ID {
						existingIdx = i
						break
					}
				}
			}
			if existingIdx >= 0 {
				state.Message.Parts[existingIdx].Data = chunk.Data
			} else {
				state.Message.Parts = append(state.Message.Parts, UIMessagePart{
					Type: chunk.Type,
					ID:   chunk.ID,
					Data: chunk.Data,
				})
			}
			write()
		}
	}

	return nil
}

// ToolCallId returns the ToolCallID field (helper for method-style access).
func (c UIMessageChunk) ToolCallId() string {
	return c.ToolCallID
}

func countToolParts(parts []UIMessagePart) int {
	n := 0
	for _, p := range parts {
		if strings.HasPrefix(p.Type, "tool-") || p.Type == "dynamic-tool" {
			n++
		}
	}
	return n
}

func findOrCreateToolPart(state *StreamingUIMessageState, toolCallID, toolName string, dynamic bool) *UIMessagePart {
	for i := range state.Message.Parts {
		p := &state.Message.Parts[i]
		if p.ToolCallID == toolCallID {
			return p
		}
	}
	// Create new part
	partType := "tool-" + toolName
	if dynamic {
		partType = "dynamic-tool"
	}
	state.Message.Parts = append(state.Message.Parts, UIMessagePart{
		Type:       partType,
		ToolCallID: toolCallID,
		ToolName:   toolName,
	})
	return &state.Message.Parts[len(state.Message.Parts)-1]
}

func updateToolPartInput(state *StreamingUIMessageState, toolCallID string, ptc *partialToolCall) {
	for i := range state.Message.Parts {
		p := &state.Message.Parts[i]
		if p.ToolCallID == toolCallID {
			p.State = "input-streaming"
			// Try to parse the partial JSON into the input field
			var partialInput any
			if err := json.Unmarshal([]byte(ptc.Text), &partialInput); err == nil {
				p.Input = partialInput
			}
			return
		}
	}
}

func mergeMetadata(existing, incoming any) any {
	if existing == nil {
		return incoming
	}
	if incoming == nil {
		return existing
	}
	// Attempt to merge as maps (matching the TS mergeObjects behavior)
	existingMap, ok1 := toMap(existing)
	incomingMap, ok2 := toMap(incoming)
	if ok1 && ok2 {
		for k, v := range incomingMap {
			existingMap[k] = v
		}
		return existingMap
	}
	return incoming
}

func toMap(v any) (map[string]any, bool) {
	switch val := v.(type) {
	case map[string]any:
		return val, true
	default:
		// Try via JSON round-trip
		data, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, false
		}
		return m, true
	}
}

type streamError struct {
	msg string
}

func newStreamError(msg string) *streamError {
	return &streamError{msg: msg}
}

func (e *streamError) Error() string {
	return e.msg
}
