// Ported from: packages/core/src/agent/message-list/message-list.ts
package messagelist

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/adapters"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/cache"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/conversion"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/detection"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/merge"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/prompt"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// RecordedEvent represents a recorded mutation event for observability.
type RecordedEvent struct {
	Type    string                 `json:"type"` // "add" | "addSystem" | "removeByIds" | "clear"
	Source  state.MessageSource    `json:"source,omitempty"`
	Count   int                    `json:"count,omitempty"`
	IDs     []string               `json:"ids,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Tag     string                 `json:"tag,omitempty"`
	Message map[string]any         `json:"message,omitempty"`
}

// MessageListOptions contains options for creating a MessageList.
type MessageListOptions struct {
	ThreadID          string
	ResourceID        string
	GenerateMessageID func(context *IdGeneratorContext) string
	// AgentNetworkAppend is an internal flag for agent network messages.
	AgentNetworkAppend bool
}

// MessageList is the central message management class.
// It handles adding, merging, converting, and tracking messages across multiple formats.
type MessageList struct {
	messages       []*state.MastraDBMessage
	systemMessages []state.CoreSystemMessage
	taggedSystemMessages map[string][]state.CoreSystemMessage
	memoryInfo     *state.MemoryInfo
	stateManager   *state.MessageStateManager

	generateMessageID func(context *IdGeneratorContext) string
	agentNetworkAppend bool

	// Event recording for observability
	isRecording    bool
	recordedEvents []RecordedEvent

	// Timestamp tracking
	lastCreatedAt *int64
}

// NewMessageList creates a new MessageList with the given options.
func NewMessageList(opts ...MessageListOptions) *MessageList {
	ml := &MessageList{
		stateManager:         state.NewMessageStateManager(),
		taggedSystemMessages: make(map[string][]state.CoreSystemMessage),
	}

	if len(opts) > 0 {
		opt := opts[0]
		if opt.ThreadID != "" {
			ml.memoryInfo = &state.MemoryInfo{
				ThreadID:   opt.ThreadID,
				ResourceID: opt.ResourceID,
			}
		}
		ml.generateMessageID = opt.GenerateMessageID
		ml.agentNetworkAppend = opt.AgentNetworkAppend
	}

	return ml
}

// StartRecording starts recording mutations for observability/tracing.
func (ml *MessageList) StartRecording() {
	ml.isRecording = true
	ml.recordedEvents = nil
}

// HasRecordedEvents returns whether there are any recorded events.
func (ml *MessageList) HasRecordedEvents() bool {
	return len(ml.recordedEvents) > 0
}

// GetRecordedEvents returns a copy of the recorded events.
func (ml *MessageList) GetRecordedEvents() []RecordedEvent {
	result := make([]RecordedEvent, len(ml.recordedEvents))
	copy(result, ml.recordedEvents)
	return result
}

// StopRecording stops recording and returns the list of recorded events.
func (ml *MessageList) StopRecording() []RecordedEvent {
	ml.isRecording = false
	events := ml.GetRecordedEvents()
	ml.recordedEvents = nil
	return events
}

// Add adds messages to the list from the given source.
func (ml *MessageList) Add(messages any, messageSource state.MessageSource) *MessageList {
	if messageSource == state.MessageSourceUser {
		messageSource = state.MessageSourceInput
	}

	if messages == nil {
		return ml
	}

	// Handle string input
	if s, ok := messages.(string); ok {
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:   "add",
				Source: messageSource,
				Count:  1,
			})
		}
		ml.addOne(map[string]any{
			"role":    "user",
			"content": s,
		}, messageSource)
		return ml
	}

	// Handle string slice
	if strs, ok := messages.([]string); ok {
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:   "add",
				Source: messageSource,
				Count:  len(strs),
			})
		}
		for _, s := range strs {
			ml.addOne(map[string]any{
				"role":    "user",
				"content": s,
			}, messageSource)
		}
		return ml
	}

	// Handle map[string]any (single message)
	if msg, ok := messages.(map[string]any); ok {
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:   "add",
				Source: messageSource,
				Count:  1,
			})
		}
		ml.addOne(msg, messageSource)
		return ml
	}

	// Handle []map[string]any (message array)
	if msgs, ok := messages.([]map[string]any); ok {
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:   "add",
				Source: messageSource,
				Count:  len(msgs),
			})
		}
		for _, msg := range msgs {
			ml.addOne(msg, messageSource)
		}
		return ml
	}

	// Handle typed MastraDBMessage
	if msg, ok := messages.(*state.MastraDBMessage); ok {
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:   "add",
				Source: messageSource,
				Count:  1,
			})
		}
		ctx := ml.createAdapterContext()
		hydrated := conversion.InputToMastraDBMessageTyped(msg, ctx)
		ml.addOneTyped(hydrated, messageSource)
		return ml
	}

	// Handle []*state.MastraDBMessage
	if msgs, ok := messages.([]*state.MastraDBMessage); ok {
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:   "add",
				Source: messageSource,
				Count:  len(msgs),
			})
		}
		for _, msg := range msgs {
			ctx := ml.createAdapterContext()
			hydrated := conversion.InputToMastraDBMessageTyped(msg, ctx)
			ml.addOneTyped(hydrated, messageSource)
		}
		return ml
	}

	// Handle []any
	if arr, ok := messages.([]any); ok {
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:   "add",
				Source: messageSource,
				Count:  len(arr),
			})
		}
		for _, item := range arr {
			switch v := item.(type) {
			case string:
				ml.addOne(map[string]any{
					"role":    "user",
					"content": v,
				}, messageSource)
			case map[string]any:
				ml.addOne(v, messageSource)
			case *state.MastraDBMessage:
				ctx := ml.createAdapterContext()
				hydrated := conversion.InputToMastraDBMessageTyped(v, ctx)
				ml.addOneTyped(hydrated, messageSource)
			}
		}
		return ml
	}

	return ml
}

// Serialize serializes the MessageList state for workflow suspend/resume.
func (ml *MessageList) Serialize() state.SerializedMessageListState {
	return ml.stateManager.SerializeAll(struct {
		Messages             []*state.MastraDBMessage
		SystemMessages       []state.CoreSystemMessage
		TaggedSystemMessages map[string][]state.CoreSystemMessage
		MemoryInfo           *state.MemoryInfo
		AgentNetworkAppend   bool
	}{
		Messages:             ml.messages,
		SystemMessages:       ml.systemMessages,
		TaggedSystemMessages: ml.taggedSystemMessages,
		MemoryInfo:           ml.memoryInfo,
		AgentNetworkAppend:   ml.agentNetworkAppend,
	})
}

// SerializeForSpan returns a clean representation for tracing/observability spans.
func (ml *MessageList) SerializeForSpan() map[string]any {
	var coreMessages []map[string]any
	for _, msg := range ml.messages {
		coreMessages = append(coreMessages, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	var systemMsgs []map[string]any
	for _, m := range ml.systemMessages {
		systemMsgs = append(systemMsgs, map[string]any{
			"role":    m.Role,
			"content": m.Content,
		})
	}
	for tag, msgs := range ml.taggedSystemMessages {
		for _, m := range msgs {
			systemMsgs = append(systemMsgs, map[string]any{
				"role":    m.Role,
				"content": m.Content,
				"tag":     tag,
			})
		}
	}

	return map[string]any{
		"messages":       coreMessages,
		"systemMessages": systemMsgs,
	}
}

// Deserialize deserializes all MessageList state from workflow suspend/resume.
func (ml *MessageList) Deserialize(s state.SerializedMessageListState) *MessageList {
	data := ml.stateManager.DeserializeAll(s)
	ml.messages = data.Messages
	ml.systemMessages = data.SystemMessages
	ml.taggedSystemMessages = data.TaggedSystemMessages
	ml.memoryInfo = data.MemoryInfo
	ml.agentNetworkAppend = data.AgentNetworkAppend
	return ml
}

// MakeMessageSourceChecker creates a source checker for efficient source lookups.
func (ml *MessageList) MakeMessageSourceChecker() *state.SourceChecker {
	return ml.stateManager.CreateSourceChecker()
}

// GetLatestUserContent returns the content string of the last user message.
func (ml *MessageList) GetLatestUserContent() string {
	for i := len(ml.messages) - 1; i >= 0; i-- {
		msg := ml.messages[i]
		if msg.Role == "user" {
			if msg.Content.Content != "" {
				return msg.Content.Content
			}
			for _, part := range msg.Content.Parts {
				if part.Type == "text" {
					return part.Text
				}
			}
		}
	}
	return ""
}

// AllDB returns all messages in MastraDBMessage format.
func (ml *MessageList) AllDB() []*state.MastraDBMessage {
	return ml.messages
}

// AllV1 returns all messages in MastraMessageV1 format.
func (ml *MessageList) AllV1() []state.MastraMessageV1 {
	return prompt.ConvertToV1Messages(derefMessages(ml.messages))
}

// derefMessages converts []*MastraDBMessage to []MastraDBMessage.
func derefMessages(msgs []*state.MastraDBMessage) []state.MastraDBMessage {
	result := make([]state.MastraDBMessage, len(msgs))
	for i, m := range msgs {
		result[i] = *m
	}
	return result
}

// AllAIV5UI returns all messages in AIV5 UIMessage format.
func (ml *MessageList) AllAIV5UI() []*adapters.AIV5UIMessage {
	var result []*adapters.AIV5UIMessage
	for _, msg := range ml.messages {
		result = append(result, adapters.AIV5ToUIMessage(msg))
	}
	return result
}

// AllAIV4UI returns all messages in AIV4 UIMessage format.
func (ml *MessageList) AllAIV4UI() []*state.UIMessageWithMetadata {
	var result []*state.UIMessageWithMetadata
	for _, msg := range ml.messages {
		result = append(result, adapters.AIV4ToUIMessage(msg))
	}
	return result
}

// RememberedDB returns messages from memory source.
func (ml *MessageList) RememberedDB() []*state.MastraDBMessage {
	memMsgs := ml.stateManager.GetMemoryMessages()
	var result []*state.MastraDBMessage
	for _, m := range ml.messages {
		if _, ok := memMsgs[m]; ok {
			result = append(result, m)
		}
	}
	return result
}

// InputDB returns messages from input source.
func (ml *MessageList) InputDB() []*state.MastraDBMessage {
	userMsgs := ml.stateManager.GetUserMessages()
	var result []*state.MastraDBMessage
	for _, m := range ml.messages {
		if _, ok := userMsgs[m]; ok {
			result = append(result, m)
		}
	}
	return result
}

// ResponseDB returns messages from response source.
func (ml *MessageList) ResponseDB() []*state.MastraDBMessage {
	respMsgs := ml.stateManager.GetResponseMessages()
	var result []*state.MastraDBMessage
	for _, m := range ml.messages {
		if _, ok := respMsgs[m]; ok {
			result = append(result, m)
		}
	}
	return result
}

// ClearAllDB clears all messages and returns them.
func (ml *MessageList) ClearAllDB() []*state.MastraDBMessage {
	allMessages := make([]*state.MastraDBMessage, len(ml.messages))
	copy(allMessages, ml.messages)
	ml.messages = nil
	ml.stateManager.ClearAll()
	if ml.isRecording && len(allMessages) > 0 {
		ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
			Type:  "clear",
			Count: len(allMessages),
		})
	}
	return allMessages
}

// ClearInputDB clears input messages and returns them.
func (ml *MessageList) ClearInputDB() []*state.MastraDBMessage {
	userMsgs := ml.stateManager.GetUserMessages()
	var removed []*state.MastraDBMessage
	var remaining []*state.MastraDBMessage
	for _, m := range ml.messages {
		if _, ok := userMsgs[m]; ok {
			removed = append(removed, m)
		} else {
			remaining = append(remaining, m)
		}
	}
	ml.messages = remaining
	ml.stateManager.ClearUserMessages()
	if ml.isRecording && len(removed) > 0 {
		ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
			Type:   "clear",
			Source: state.MessageSourceInput,
			Count:  len(removed),
		})
	}
	return removed
}

// ClearResponseDB clears response messages and returns them.
func (ml *MessageList) ClearResponseDB() []*state.MastraDBMessage {
	respMsgs := ml.stateManager.GetResponseMessages()
	var removed []*state.MastraDBMessage
	var remaining []*state.MastraDBMessage
	for _, m := range ml.messages {
		if _, ok := respMsgs[m]; ok {
			removed = append(removed, m)
		} else {
			remaining = append(remaining, m)
		}
	}
	ml.messages = remaining
	ml.stateManager.ClearResponseMessages()
	if ml.isRecording && len(removed) > 0 {
		ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
			Type:   "clear",
			Source: state.MessageSourceResponse,
			Count:  len(removed),
		})
	}
	return removed
}

// RemoveByIds removes messages by ID and returns the removed messages.
func (ml *MessageList) RemoveByIds(ids []string) []*state.MastraDBMessage {
	idsSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idsSet[id] = struct{}{}
	}
	var removed []*state.MastraDBMessage
	var remaining []*state.MastraDBMessage
	for _, m := range ml.messages {
		if _, ok := idsSet[m.ID]; ok {
			removed = append(removed, m)
			ml.stateManager.RemoveMessage(m)
		} else {
			remaining = append(remaining, m)
		}
	}
	ml.messages = remaining
	if ml.isRecording && len(removed) > 0 {
		ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
			Type:  "removeByIds",
			IDs:   ids,
			Count: len(removed),
		})
	}
	return removed
}

// DrainUnsavedMessages returns and clears unsaved user and response messages.
func (ml *MessageList) DrainUnsavedMessages() []*state.MastraDBMessage {
	userMsgs := ml.stateManager.GetUserMessages()
	respMsgs := ml.stateManager.GetResponseMessages()
	var result []*state.MastraDBMessage
	for _, m := range ml.messages {
		_, isUser := userMsgs[m]
		_, isResp := respMsgs[m]
		if isUser || isResp {
			result = append(result, m)
		}
	}
	ml.stateManager.ClearUserMessages()
	ml.stateManager.ClearResponseMessages()
	return result
}

// GetEarliestUnsavedMessageTimestamp returns the earliest timestamp among unsaved messages.
func (ml *MessageList) GetEarliestUnsavedMessageTimestamp() *int64 {
	userMsgs := ml.stateManager.GetUserMessages()
	respMsgs := ml.stateManager.GetResponseMessages()
	var earliest *int64
	for _, m := range ml.messages {
		_, isUser := userMsgs[m]
		_, isResp := respMsgs[m]
		if isUser || isResp {
			t := m.CreatedAt.UnixMilli()
			if earliest == nil || t < *earliest {
				earliest = &t
			}
		}
	}
	return earliest
}

// IsNewMessage checks if a message is a new user or response message.
func (ml *MessageList) IsNewMessage(messageOrID interface{}) bool {
	return ml.stateManager.IsNewMessage(messageOrID)
}

// GetSystemMessages returns system messages, optionally filtered by tag.
func (ml *MessageList) GetSystemMessages(tag ...string) []state.CoreSystemMessage {
	if len(tag) > 0 && tag[0] != "" {
		msgs, ok := ml.taggedSystemMessages[tag[0]]
		if !ok {
			return nil
		}
		return msgs
	}
	return ml.systemMessages
}

// GetAllSystemMessages returns all system messages (both tagged and untagged).
func (ml *MessageList) GetAllSystemMessages() []state.CoreSystemMessage {
	result := make([]state.CoreSystemMessage, len(ml.systemMessages))
	copy(result, ml.systemMessages)
	for _, msgs := range ml.taggedSystemMessages {
		result = append(result, msgs...)
	}
	return result
}

// ClearSystemMessages clears system messages, optionally for a specific tag.
func (ml *MessageList) ClearSystemMessages(tag ...string) *MessageList {
	if len(tag) > 0 && tag[0] != "" {
		delete(ml.taggedSystemMessages, tag[0])
	} else {
		ml.systemMessages = nil
	}
	return ml
}

// ReplaceAllSystemMessages replaces all system messages with new ones.
func (ml *MessageList) ReplaceAllSystemMessages(messages []state.CoreSystemMessage) *MessageList {
	ml.systemMessages = nil
	ml.taggedSystemMessages = make(map[string][]state.CoreSystemMessage)
	for _, msg := range messages {
		if msg.Role == "system" {
			ml.systemMessages = append(ml.systemMessages, msg)
		}
	}
	return ml
}

// AddSystem adds system messages with an optional tag.
func (ml *MessageList) AddSystem(messages any, tag ...string) *MessageList {
	if messages == nil {
		return ml
	}

	t := ""
	if len(tag) > 0 {
		t = tag[0]
	}

	switch v := messages.(type) {
	case string:
		ml.addOneSystem(map[string]any{"role": "system", "content": v}, t)
	case []string:
		for _, s := range v {
			ml.addOneSystem(map[string]any{"role": "system", "content": s}, t)
		}
	case map[string]any:
		ml.addOneSystem(v, t)
	case []map[string]any:
		for _, m := range v {
			ml.addOneSystem(m, t)
		}
	case state.CoreSystemMessage:
		ml.addOneSystemTyped(v, t)
	case []state.CoreSystemMessage:
		for _, m := range v {
			ml.addOneSystemTyped(m, t)
		}
	case *state.MastraDBMessage:
		result, _ := adapters.AIV4SystemToV4Core(v)
		if result != nil {
			ml.addOneSystem(result, t)
		}
	case []*state.MastraDBMessage:
		for _, m := range v {
			result, _ := adapters.AIV4SystemToV4Core(m)
			if result != nil {
				ml.addOneSystem(result, t)
			}
		}
	}

	return ml
}

func (ml *MessageList) addOneSystem(message map[string]any, tag string) {
	coreMessage := conversion.SystemMessageToAIV4Core(message)
	role, _ := coreMessage["role"].(string)
	if role != "system" {
		return
	}

	csm := state.CoreSystemMessage{
		Role:    "system",
		Content: coreMessage["content"],
	}

	if tag != "" && !ml.isDuplicateSystem(csm, tag) {
		if ml.taggedSystemMessages[tag] == nil {
			ml.taggedSystemMessages[tag] = []state.CoreSystemMessage{}
		}
		ml.taggedSystemMessages[tag] = append(ml.taggedSystemMessages[tag], csm)
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:    "addSystem",
				Tag:     tag,
				Message: coreMessage,
			})
		}
	} else if tag == "" && !ml.isDuplicateSystem(csm, "") {
		ml.systemMessages = append(ml.systemMessages, csm)
		if ml.isRecording {
			ml.recordedEvents = append(ml.recordedEvents, RecordedEvent{
				Type:    "addSystem",
				Message: coreMessage,
			})
		}
	}
}

func (ml *MessageList) addOneSystemTyped(csm state.CoreSystemMessage, tag string) {
	if csm.Role != "system" {
		return
	}
	if tag != "" && !ml.isDuplicateSystem(csm, tag) {
		if ml.taggedSystemMessages[tag] == nil {
			ml.taggedSystemMessages[tag] = []state.CoreSystemMessage{}
		}
		ml.taggedSystemMessages[tag] = append(ml.taggedSystemMessages[tag], csm)
	} else if tag == "" && !ml.isDuplicateSystem(csm, "") {
		ml.systemMessages = append(ml.systemMessages, csm)
	}
}

func (ml *MessageList) isDuplicateSystem(message state.CoreSystemMessage, tag string) bool {
	key := cache.FromAIV4CoreMessageContent(message.Content)
	if tag != "" {
		msgs, ok := ml.taggedSystemMessages[tag]
		if !ok {
			return false
		}
		for _, m := range msgs {
			if cache.FromAIV4CoreMessageContent(m.Content) == key {
				return true
			}
		}
		return false
	}
	for _, m := range ml.systemMessages {
		if cache.FromAIV4CoreMessageContent(m.Content) == key {
			return true
		}
	}
	return false
}

func (ml *MessageList) getMessageByID(id string) *state.MastraDBMessage {
	for _, m := range ml.messages {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func (ml *MessageList) shouldReplaceMessage(message *state.MastraDBMessage) (exists bool, shouldReplace bool, id string) {
	if len(ml.messages) == 0 {
		return false, false, ""
	}
	if message.ID == "" {
		return false, false, ""
	}

	existing := ml.getMessageByID(message.ID)
	if existing == nil {
		return false, false, ""
	}

	return true, !conversion.MessagesAreEqualDB(existing, message), existing.ID
}

func (ml *MessageList) addOne(message map[string]any, messageSource state.MessageSource) *MessageList {
	content := message["content"]
	parts := message["parts"]
	role, _ := message["role"].(string)

	// Validate content or parts exist
	hasContent := content != nil
	if s, ok := content.(string); ok && s == "" {
		hasContent = true // allow empty strings
	}
	hasParts := parts != nil

	if !hasContent && !hasParts {
		return ml
	}

	if role == "system" {
		if messageSource == state.MessageSourceMemory {
			return ml
		}
		if detection.IsAIV4CoreMessage(message) || detection.IsAIV5CoreMessage(message) || detection.IsMastraDBMessage(message) {
			ml.AddSystem(message)
			return ml
		}
		return ml
	}

	ctx := ml.createAdapterContext()
	messageV2, err := conversion.InputToMastraDBMessage(message, messageSource, ctx)
	if err != nil {
		return ml
	}

	ml.addOneTyped(messageV2, messageSource)
	return ml
}

func (ml *MessageList) addOneTyped(messageV2 *state.MastraDBMessage, messageSource state.MessageSource) {
	exists, shouldReplace, existingID := ml.shouldReplaceMessage(messageV2)

	latestMessage := ml.latestMessage()

	if messageSource == state.MessageSourceMemory {
		for _, existingMsg := range ml.messages {
			if conversion.MessagesAreEqualDB(existingMsg, messageV2) {
				return
			}
		}
	}

	// Check if we should merge with the latest assistant message
	isLatestFromMemory := false
	if latestMessage != nil {
		isLatestFromMemory = ml.stateManager.IsMemoryMessage(latestMessage)
	}
	shouldMerge := merge.ShouldMerge(latestMessage, messageV2, string(messageSource), isLatestFromMemory, ml.agentNetworkAppend)

	if shouldMerge && latestMessage != nil {
		merge.Merge(latestMessage, messageV2)
		ml.stateManager.AddToSource(latestMessage, messageSource)
	} else {
		if shouldReplace {
			existingIndex := -1
			for i, m := range ml.messages {
				if m.ID == existingID {
					existingIndex = i
					break
				}
			}
			if existingIndex >= 0 {
				existingMsg := ml.messages[existingIndex]
				if merge.IsSealed(existingMsg) {
					// Handle sealed message replacement
					existingParts := existingMsg.Content.Parts
					sealedPartCount := 0
					for i := len(existingParts) - 1; i >= 0; i-- {
						if existingParts[i].Metadata != nil {
							if mastra, ok := existingParts[i].Metadata["mastra"].(map[string]any); ok {
								if _, ok := mastra["sealedAt"]; ok {
									sealedPartCount = i + 1
									break
								}
							}
						}
					}
					if sealedPartCount == 0 {
						sealedPartCount = len(existingParts)
					}

					incomingParts := messageV2.Content.Parts
					var newParts []state.MastraMessagePart

					if len(incomingParts) <= sealedPartCount {
						if conversion.MessagesAreEqualDB(existingMsg, messageV2) {
							return
						}
						newParts = incomingParts
					} else {
						newParts = incomingParts[sealedPartCount:]
					}

					if len(newParts) > 0 {
						messageV2.ID = ml.newMessageID("")
						messageV2.Content.Parts = newParts
						if !messageV2.CreatedAt.After(existingMsg.CreatedAt) {
							messageV2.CreatedAt = existingMsg.CreatedAt.Add(time.Millisecond)
						}
						ml.messages = append(ml.messages, messageV2)
					}
				} else {
					// Check if we should merge into the existing message
					isExistingFromMemory := ml.stateManager.IsMemoryMessage(existingMsg)
					shouldMergeInto := merge.ShouldMerge(existingMsg, messageV2, string(messageSource), isExistingFromMemory, ml.agentNetworkAppend)
					if shouldMergeInto {
						merge.Merge(existingMsg, messageV2)
						ml.stateManager.AddToSource(existingMsg, messageSource)
						ml.sortMessages()
						return
					}
					ml.messages[existingIndex] = messageV2
				}
			}
		} else if !exists {
			ml.messages = append(ml.messages, messageV2)
		}

		ml.stateManager.AddToSource(messageV2, messageSource)
	}

	ml.sortMessages()
}

func (ml *MessageList) latestMessage() *state.MastraDBMessage {
	if len(ml.messages) == 0 {
		return nil
	}
	return ml.messages[len(ml.messages)-1]
}

func (ml *MessageList) sortMessages() {
	sort.Slice(ml.messages, func(i, j int) bool {
		return ml.messages[i].CreatedAt.Before(ml.messages[j].CreatedAt)
	})
}

func (ml *MessageList) generateCreatedAt(messageSource state.MessageSource, start ...any) time.Time {
	// Normalize timestamp
	var startDate *time.Time
	if len(start) > 0 && start[0] != nil {
		switch v := start[0].(type) {
		case time.Time:
			startDate = &v
		case string:
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				startDate = &t
			}
		case int64:
			t := time.UnixMilli(v)
			startDate = &t
		case float64:
			t := time.UnixMilli(int64(v))
			startDate = &t
		}
	}

	if startDate != nil && ml.lastCreatedAt == nil {
		t := startDate.UnixMilli()
		ml.lastCreatedAt = &t
		return *startDate
	}

	if startDate != nil && messageSource == state.MessageSourceMemory {
		return *startDate
	}

	now := time.Now()
	nowTime := now.UnixMilli()
	if startDate != nil {
		nowTime = startDate.UnixMilli()
	}

	// Find the latest createdAt in all stored messages
	var lastTime int64
	if ml.lastCreatedAt != nil {
		lastTime = *ml.lastCreatedAt
	}
	for _, m := range ml.messages {
		t := m.CreatedAt.UnixMilli()
		if t > lastTime {
			lastTime = t
		}
	}

	if nowTime <= lastTime {
		newTime := lastTime + 1
		ml.lastCreatedAt = &newTime
		return time.UnixMilli(newTime)
	}

	ml.lastCreatedAt = &nowTime
	return now
}

func (ml *MessageList) newMessageID(role string) string {
	if ml.generateMessageID != nil {
		source := aktypes.IdGeneratorSourceAgent
		threadID := ml.threadID()
		resourceID := ml.resourceID()
		return ml.generateMessageID(&IdGeneratorContext{
			IdType:     aktypes.IdTypeMessage,
			Source:     &source,
			ThreadId:   &threadID,
			ResourceId: &resourceID,
			Role:       &role,
		})
	}
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

func (ml *MessageList) threadID() string {
	if ml.memoryInfo != nil {
		return ml.memoryInfo.ThreadID
	}
	return ""
}

func (ml *MessageList) resourceID() string {
	if ml.memoryInfo != nil {
		return ml.memoryInfo.ResourceID
	}
	return ""
}

func (ml *MessageList) createAdapterContext() *conversion.InputConversionContext {
	return &conversion.InputConversionContext{
		MemoryInfo:   ml.memoryInfo,
		NewMessageID: func() string { return ml.newMessageID("") },
		GenerateCreatedAt: func(ms state.MessageSource, start ...any) time.Time {
			return ml.generateCreatedAt(ms, start...)
		},
		DBMessages: ml.messages,
	}
}

// ensure json import is used
var _ = json.Marshal
