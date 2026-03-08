// Ported from: packages/core/src/agent/message-list/state/MessageStateManager.ts
package state

// CoreSystemMessage is a stub for the AI SDK V4 CoreSystemMessage type.
// TODO: In TS this comes from @internal/ai-sdk-v4 CoreSystemMessage.
type CoreSystemMessage struct {
	Role                          string           `json:"role"` // always "system"
	Content                       any              `json:"content"`
	ExperimentalProviderMetadata  ProviderMetadata `json:"experimental_providerMetadata,omitempty"`
	ProviderOptions               ProviderMetadata `json:"providerOptions,omitempty"`
}

// SerializedMessageListState is the serialized form of the complete MessageList state.
type SerializedMessageListState struct {
	Messages                     []SerializedMessage            `json:"messages"`
	SystemMessages               []CoreSystemMessage            `json:"systemMessages"`
	TaggedSystemMessages         map[string][]CoreSystemMessage `json:"taggedSystemMessages"`
	MemoryInfo                   *MemoryInfo                    `json:"memoryInfo"`
	AgentNetworkAppend           bool                           `json:"_agentNetworkAppend"`
	MemoryMessages               []string                       `json:"memoryMessages"`
	NewUserMessages              []string                       `json:"newUserMessages"`
	NewResponseMessages          []string                       `json:"newResponseMessages"`
	UserContextMessages          []string                       `json:"userContextMessages"`
	MemoryMessagesPersisted      []string                       `json:"memoryMessagesPersisted"`
	NewUserMessagesPersisted     []string                       `json:"newUserMessagesPersisted"`
	NewResponseMessagesPersisted []string                       `json:"newResponseMessagesPersisted"`
	UserContextMessagesPersisted []string                       `json:"userContextMessagesPersisted"`
}

// MessageStateManager manages the state of messages in a MessageList.
// It tracks messages by their source (memory, input, response, context)
// and which messages have been persisted.
type MessageStateManager struct {
	// Messages tracked by source
	memoryMessages      map[*MastraDBMessage]struct{}
	newUserMessages     map[*MastraDBMessage]struct{}
	newResponseMessages map[*MastraDBMessage]struct{}
	userContextMessages map[*MastraDBMessage]struct{}

	// Persisted message tracking
	memoryMessagesPersisted      map[*MastraDBMessage]struct{}
	newUserMessagesPersisted     map[*MastraDBMessage]struct{}
	newResponseMessagesPersisted map[*MastraDBMessage]struct{}
	userContextMessagesPersisted map[*MastraDBMessage]struct{}
}

// NewMessageStateManager creates a new MessageStateManager.
func NewMessageStateManager() *MessageStateManager {
	return &MessageStateManager{
		memoryMessages:               make(map[*MastraDBMessage]struct{}),
		newUserMessages:              make(map[*MastraDBMessage]struct{}),
		newResponseMessages:          make(map[*MastraDBMessage]struct{}),
		userContextMessages:          make(map[*MastraDBMessage]struct{}),
		memoryMessagesPersisted:      make(map[*MastraDBMessage]struct{}),
		newUserMessagesPersisted:     make(map[*MastraDBMessage]struct{}),
		newResponseMessagesPersisted: make(map[*MastraDBMessage]struct{}),
		userContextMessagesPersisted: make(map[*MastraDBMessage]struct{}),
	}
}

// AddToSource adds a message to the appropriate source set and persisted set.
func (m *MessageStateManager) AddToSource(message *MastraDBMessage, source MessageSource) {
	switch source {
	case MessageSourceMemory:
		m.memoryMessages[message] = struct{}{}
		m.memoryMessagesPersisted[message] = struct{}{}
	case MessageSourceResponse:
		m.newResponseMessages[message] = struct{}{}
		m.newResponseMessagesPersisted[message] = struct{}{}
		// Handle case where a client-side tool response was added as user input
		delete(m.newUserMessages, message)
	case MessageSourceInput, MessageSourceUser:
		m.newUserMessages[message] = struct{}{}
		m.newUserMessagesPersisted[message] = struct{}{}
	case MessageSourceContext:
		m.userContextMessages[message] = struct{}{}
		m.userContextMessagesPersisted[message] = struct{}{}
	default:
		panic("Missing message source for message")
	}
}

// IsMemoryMessage checks if a message belongs to the memory source.
func (m *MessageStateManager) IsMemoryMessage(message *MastraDBMessage) bool {
	_, ok := m.memoryMessages[message]
	return ok
}

// IsUserMessage checks if a message belongs to the input source.
func (m *MessageStateManager) IsUserMessage(message *MastraDBMessage) bool {
	_, ok := m.newUserMessages[message]
	return ok
}

// IsResponseMessage checks if a message belongs to the response source.
func (m *MessageStateManager) IsResponseMessage(message *MastraDBMessage) bool {
	_, ok := m.newResponseMessages[message]
	return ok
}

// IsContextMessage checks if a message belongs to the context source.
func (m *MessageStateManager) IsContextMessage(message *MastraDBMessage) bool {
	_, ok := m.userContextMessages[message]
	return ok
}

// GetMemoryMessages returns all memory messages.
func (m *MessageStateManager) GetMemoryMessages() map[*MastraDBMessage]struct{} {
	return m.memoryMessages
}

// GetUserMessages returns all user/input messages.
func (m *MessageStateManager) GetUserMessages() map[*MastraDBMessage]struct{} {
	return m.newUserMessages
}

// GetResponseMessages returns all response messages.
func (m *MessageStateManager) GetResponseMessages() map[*MastraDBMessage]struct{} {
	return m.newResponseMessages
}

// GetContextMessages returns all context messages.
func (m *MessageStateManager) GetContextMessages() map[*MastraDBMessage]struct{} {
	return m.userContextMessages
}

// GetMemoryMessagesPersisted returns persisted memory messages.
func (m *MessageStateManager) GetMemoryMessagesPersisted() map[*MastraDBMessage]struct{} {
	return m.memoryMessagesPersisted
}

// GetUserMessagesPersisted returns persisted user/input messages.
func (m *MessageStateManager) GetUserMessagesPersisted() map[*MastraDBMessage]struct{} {
	return m.newUserMessagesPersisted
}

// GetResponseMessagesPersisted returns persisted response messages.
func (m *MessageStateManager) GetResponseMessagesPersisted() map[*MastraDBMessage]struct{} {
	return m.newResponseMessagesPersisted
}

// GetContextMessagesPersisted returns persisted context messages.
func (m *MessageStateManager) GetContextMessagesPersisted() map[*MastraDBMessage]struct{} {
	return m.userContextMessagesPersisted
}

// RemoveMessage removes a message from all source sets.
func (m *MessageStateManager) RemoveMessage(message *MastraDBMessage) {
	delete(m.memoryMessages, message)
	delete(m.newUserMessages, message)
	delete(m.newResponseMessages, message)
	delete(m.userContextMessages, message)
}

// ClearUserMessages clears all user messages.
func (m *MessageStateManager) ClearUserMessages() {
	m.newUserMessages = make(map[*MastraDBMessage]struct{})
}

// ClearResponseMessages clears all response messages.
func (m *MessageStateManager) ClearResponseMessages() {
	m.newResponseMessages = make(map[*MastraDBMessage]struct{})
}

// ClearContextMessages clears all context messages.
func (m *MessageStateManager) ClearContextMessages() {
	m.userContextMessages = make(map[*MastraDBMessage]struct{})
}

// ClearAll clears all messages from all sources (but not persisted tracking).
func (m *MessageStateManager) ClearAll() {
	m.newUserMessages = make(map[*MastraDBMessage]struct{})
	m.newResponseMessages = make(map[*MastraDBMessage]struct{})
	m.userContextMessages = make(map[*MastraDBMessage]struct{})
}

// SourceChecker provides efficient lookup of message sources by ID.
type SourceChecker struct {
	Memory  map[string]struct{}
	Input   map[string]struct{}
	Output  map[string]struct{}
	Context map[string]struct{}
}

// GetSource returns the source of a message, or empty string if not found.
func (sc *SourceChecker) GetSource(msg *MastraDBMessage) MessageSource {
	if _, ok := sc.Memory[msg.ID]; ok {
		return MessageSourceMemory
	}
	if _, ok := sc.Input[msg.ID]; ok {
		return MessageSourceInput
	}
	if _, ok := sc.Output[msg.ID]; ok {
		return MessageSourceResponse
	}
	if _, ok := sc.Context[msg.ID]; ok {
		return MessageSourceContext
	}
	return ""
}

// CreateSourceChecker creates a lookup function to determine message source.
func (m *MessageStateManager) CreateSourceChecker() *SourceChecker {
	sc := &SourceChecker{
		Memory:  make(map[string]struct{}),
		Input:   make(map[string]struct{}),
		Output:  make(map[string]struct{}),
		Context: make(map[string]struct{}),
	}
	for msg := range m.memoryMessages {
		sc.Memory[msg.ID] = struct{}{}
	}
	for msg := range m.newUserMessages {
		sc.Input[msg.ID] = struct{}{}
	}
	for msg := range m.newResponseMessages {
		sc.Output[msg.ID] = struct{}{}
	}
	for msg := range m.userContextMessages {
		sc.Context[msg.ID] = struct{}{}
	}
	return sc
}

// IsNewMessage checks if a message is a new (unsaved) user or response message by ID.
func (m *MessageStateManager) IsNewMessage(messageOrID interface{}) bool {
	switch v := messageOrID.(type) {
	case *MastraDBMessage:
		if _, ok := m.newUserMessages[v]; ok {
			return true
		}
		if _, ok := m.newResponseMessages[v]; ok {
			return true
		}
		// Check by ID (handles copies)
		for msg := range m.newUserMessages {
			if msg.ID == v.ID {
				return true
			}
		}
		for msg := range m.newResponseMessages {
			if msg.ID == v.ID {
				return true
			}
		}
		return false
	case string:
		for msg := range m.newUserMessages {
			if msg.ID == v {
				return true
			}
		}
		for msg := range m.newResponseMessages {
			if msg.ID == v {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func serializeSet(set map[*MastraDBMessage]struct{}) []string {
	ids := make([]string, 0, len(set))
	for msg := range set {
		ids = append(ids, msg.ID)
	}
	return ids
}

func deserializeSet(ids []string, messages []*MastraDBMessage) map[*MastraDBMessage]struct{} {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	result := make(map[*MastraDBMessage]struct{})
	for _, msg := range messages {
		if _, ok := idSet[msg.ID]; ok {
			result[msg] = struct{}{}
		}
	}
	return result
}

// SerializeAll serializes all MessageList state for workflow suspend/resume.
func (m *MessageStateManager) SerializeAll(data struct {
	Messages             []*MastraDBMessage
	SystemMessages       []CoreSystemMessage
	TaggedSystemMessages map[string][]CoreSystemMessage
	MemoryInfo           *MemoryInfo
	AgentNetworkAppend   bool
}) SerializedMessageListState {
	msgs := make([]MastraDBMessage, len(data.Messages))
	for i, msg := range data.Messages {
		msgs[i] = *msg
	}
	return SerializedMessageListState{
		Messages:                     SerializeMessages(msgs),
		SystemMessages:               data.SystemMessages,
		TaggedSystemMessages:         data.TaggedSystemMessages,
		MemoryInfo:                   data.MemoryInfo,
		AgentNetworkAppend:           data.AgentNetworkAppend,
		MemoryMessages:               serializeSet(m.memoryMessages),
		NewUserMessages:              serializeSet(m.newUserMessages),
		NewResponseMessages:          serializeSet(m.newResponseMessages),
		UserContextMessages:          serializeSet(m.userContextMessages),
		MemoryMessagesPersisted:      serializeSet(m.memoryMessagesPersisted),
		NewUserMessagesPersisted:     serializeSet(m.newUserMessagesPersisted),
		NewResponseMessagesPersisted: serializeSet(m.newResponseMessagesPersisted),
		UserContextMessagesPersisted: serializeSet(m.userContextMessagesPersisted),
	}
}

// DeserializeAllResult holds the result of deserialization.
type DeserializeAllResult struct {
	Messages             []*MastraDBMessage
	SystemMessages       []CoreSystemMessage
	TaggedSystemMessages map[string][]CoreSystemMessage
	MemoryInfo           *MemoryInfo
	AgentNetworkAppend   bool
}

// DeserializeAll deserializes all MessageList state from workflow suspend/resume.
func (m *MessageStateManager) DeserializeAll(s SerializedMessageListState) DeserializeAllResult {
	deserialized := DeserializeMessages(s.Messages)
	messages := make([]*MastraDBMessage, len(deserialized))
	for i := range deserialized {
		messages[i] = &deserialized[i]
	}

	m.memoryMessages = deserializeSet(s.MemoryMessages, messages)
	m.newUserMessages = deserializeSet(s.NewUserMessages, messages)
	m.newResponseMessages = deserializeSet(s.NewResponseMessages, messages)
	m.userContextMessages = deserializeSet(s.UserContextMessages, messages)
	m.memoryMessagesPersisted = deserializeSet(s.MemoryMessagesPersisted, messages)
	m.newUserMessagesPersisted = deserializeSet(s.NewUserMessagesPersisted, messages)
	m.newResponseMessagesPersisted = deserializeSet(s.NewResponseMessagesPersisted, messages)
	m.userContextMessagesPersisted = deserializeSet(s.UserContextMessagesPersisted, messages)

	return DeserializeAllResult{
		Messages:             messages,
		SystemMessages:       s.SystemMessages,
		TaggedSystemMessages: s.TaggedSystemMessages,
		MemoryInfo:           s.MemoryInfo,
		AgentNetworkAppend:   s.AgentNetworkAppend,
	}
}
