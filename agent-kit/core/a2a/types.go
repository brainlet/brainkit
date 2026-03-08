// Ported from: packages/core/src/a2a/types.ts
package a2a

// ---------------------------------------------------------------------------
// Stub types from @a2a-js/sdk
// These are local stubs for types that originate in the external @a2a-js/sdk
// package. They are defined here to avoid an external dependency. Each stub
// faithfully mirrors the A2A protocol specification.
// TODO: Replace with a dedicated Go A2A SDK package when one is available.
// ---------------------------------------------------------------------------

// TaskState represents the current state of a task in the A2A protocol.
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateCanceled      TaskState = "canceled"
	TaskStateFailed        TaskState = "failed"
	TaskStateRejected      TaskState = "rejected"
	TaskStateAuthRequired  TaskState = "auth-required"
	TaskStateUnknown       TaskState = "unknown"
)

// MessageRole represents the sender's role in a message.
type MessageRole string

const (
	MessageRoleAgent MessageRole = "agent"
	MessageRoleUser  MessageRole = "user"
)

// PartKind represents the type discriminator for message parts.
type PartKind string

const (
	PartKindText PartKind = "text"
	PartKindFile PartKind = "file"
	PartKindData PartKind = "data"
)

// TextPart represents a text segment within message parts.
type TextPart struct {
	// Part type - text for TextParts.
	Kind PartKind `json:"kind"` // always "text"
	// Optional metadata associated with the part.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Text content.
	Text string `json:"text"`
}

// FileWithBytes defines a file variant where bytes (base64) is present.
type FileWithBytes struct {
	// Base64 encoded content of the file.
	Bytes string `json:"bytes"`
	// Optional mimeType for the file.
	MimeType string `json:"mimeType,omitempty"`
	// Optional name for the file.
	Name string `json:"name,omitempty"`
}

// FileWithURI defines a file variant where uri is present.
type FileWithURI struct {
	// Optional mimeType for the file.
	MimeType string `json:"mimeType,omitempty"`
	// Optional name for the file.
	Name string `json:"name,omitempty"`
	// URL for the File content.
	URI string `json:"uri"`
}

// FileContent represents file content that is either bytes-based or URI-based.
// In Go we represent the TS union (FileWithBytes | FileWithUri) as a struct
// where the caller populates either Bytes or URI.
type FileContent struct {
	// Base64 encoded content of the file (mutually exclusive with URI).
	Bytes string `json:"bytes,omitempty"`
	// URL for the File content (mutually exclusive with Bytes).
	URI string `json:"uri,omitempty"`
	// Optional mimeType for the file.
	MimeType string `json:"mimeType,omitempty"`
	// Optional name for the file.
	Name string `json:"name,omitempty"`
}

// FilePart represents a file segment within message parts.
type FilePart struct {
	// File content either as url or bytes.
	File FileContent `json:"file"`
	// Part type - file for FileParts.
	Kind PartKind `json:"kind"` // always "file"
	// Optional metadata associated with the part.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DataPart represents a structured data segment within message parts.
type DataPart struct {
	// Structured data content.
	Data map[string]any `json:"data"`
	// Part type - data for DataParts.
	Kind PartKind `json:"kind"` // always "data"
	// Optional metadata associated with the part.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Part represents a part of a message, which can be text, a file, or structured data.
// In Go we represent the TS union (TextPart | FilePart | DataPart) as a struct
// with a Kind discriminator. The caller populates the fields matching the Kind.
type Part struct {
	// Discriminator: "text", "file", or "data".
	Kind PartKind `json:"kind"`

	// TextPart fields (populated when Kind == "text").
	Text string `json:"text,omitempty"`

	// FilePart fields (populated when Kind == "file").
	File *FileContent `json:"file,omitempty"`

	// DataPart fields (populated when Kind == "data").
	Data map[string]any `json:"data,omitempty"`

	// Optional metadata associated with the part (all kinds).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Artifact represents an artifact created by the agent during task execution.
type Artifact struct {
	// Unique identifier for the artifact.
	ArtifactID string `json:"artifactId"`
	// Optional description for the artifact.
	Description string `json:"description,omitempty"`
	// The URIs of extensions that are present or contributed to this Artifact.
	Extensions []string `json:"extensions,omitempty"`
	// Extension metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Optional name for the artifact.
	Name string `json:"name,omitempty"`
	// Artifact parts.
	Parts []Part `json:"parts"`
}

// TaskStatus represents the current status of a task.
type TaskStatus struct {
	// Optional message associated with the status.
	Message *Message `json:"message,omitempty"`
	// The state of the task.
	State TaskState `json:"state"`
	// ISO 8601 datetime string when the status was recorded.
	Timestamp string `json:"timestamp,omitempty"`
}

// Message represents a single message exchanged between user and agent.
type Message struct {
	// The context the message is associated with.
	ContextID string `json:"contextId,omitempty"`
	// The URIs of extensions that are present or contributed to this Message.
	Extensions []string `json:"extensions,omitempty"`
	// Event type.
	Kind string `json:"kind"` // always "message"
	// Identifier created by the message creator.
	MessageID string `json:"messageId"`
	// Extension metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Message content.
	Parts []Part `json:"parts"`
	// List of tasks referenced as context by this message.
	ReferenceTaskIDs []string `json:"referenceTaskIds,omitempty"`
	// Message sender's role.
	Role MessageRole `json:"role"`
	// Identifier of task the message is related to.
	TaskID string `json:"taskId,omitempty"`
}

// Task represents an A2A protocol task.
type Task struct {
	// Collection of artifacts created by the agent.
	Artifacts []Artifact `json:"artifacts,omitempty"`
	// Server-generated id for contextual alignment across interactions.
	ContextID string `json:"contextId"`
	// Message history.
	History []Message `json:"history,omitempty"`
	// Unique identifier for the task.
	ID string `json:"id"`
	// Event type.
	Kind string `json:"kind"` // always "task"
	// Extension metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Current status of the task.
	Status TaskStatus `json:"status"`
}

// JSONRPCMessage represents the base JSON-RPC 2.0 message.
type JSONRPCMessage struct {
	// An identifier established by the Client. May be string, number, or null.
	ID any `json:"id,omitempty"`
	// Specifies the version of the JSON-RPC protocol. MUST be exactly "2.0".
	JSONRPC string `json:"jsonrpc"`
}

// ---------------------------------------------------------------------------
// Types defined directly in packages/core/src/a2a/types.ts
// ---------------------------------------------------------------------------

// JSONRPCError represents a JSON-RPC error object.
type JSONRPCError struct {
	// A number indicating the error type that occurred.
	Code int `json:"code"`
	// A string providing a short description of the error.
	Message string `json:"message"`
	// Optional additional data about the error.
	Data any `json:"data,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC response object.
type JSONRPCResponse struct {
	JSONRPCMessage
	// The result of the method invocation. Required on success.
	// Should be nil or omitted if an error occurred.
	Result any `json:"result,omitempty"`
	// An error object if an error occurred during the request. Required on failure.
	// Should be nil or omitted if the request was successful.
	Error *JSONRPCError `json:"error,omitempty"`
}

// TaskContext provides context to a task handler when it is invoked or resumed.
type TaskContext struct {
	// The current state of the task when the handler is invoked or resumed.
	// Note: This is a snapshot. For the absolute latest state during async operations,
	// the handler might need to reload the task via the store.
	Task Task

	// The specific user message that triggered this handler invocation or resumption.
	UserMessage Message

	// IsCancelled checks if cancellation has been requested for this task.
	// Handlers should ideally check this periodically during long-running operations.
	IsCancelled func() bool

	// The message history associated with the task up to the point the handler is invoked.
	// Optional, as history might not always be available or relevant.
	History []Message
}

// === Error Codes (Standard JSON-RPC and A2A-specific) ===

// KnownErrorCode is the type for well-known A2A and standard JSON-RPC error codes.
type KnownErrorCode int

const (
	// ErrorCodeParseError is the error code for JSON Parse Error (-32700).
	// Invalid JSON was received by the server.
	ErrorCodeParseError KnownErrorCode = -32700

	// ErrorCodeInvalidRequest is the error code for Invalid Request (-32600).
	// The JSON sent is not a valid Request object.
	ErrorCodeInvalidRequest KnownErrorCode = -32600

	// ErrorCodeMethodNotFound is the error code for Method Not Found (-32601).
	// The method does not exist / is not available.
	ErrorCodeMethodNotFound KnownErrorCode = -32601

	// ErrorCodeInvalidParams is the error code for Invalid Params (-32602).
	// Invalid method parameter(s).
	ErrorCodeInvalidParams KnownErrorCode = -32602

	// ErrorCodeInternalError is the error code for Internal Error (-32603).
	// Internal JSON-RPC error.
	ErrorCodeInternalError KnownErrorCode = -32603

	// ErrorCodeTaskNotFound is the error code for Task Not Found (-32001).
	// The specified task was not found.
	ErrorCodeTaskNotFound KnownErrorCode = -32001

	// ErrorCodeTaskNotCancelable is the error code for Task Not Cancelable (-32002).
	// The specified task cannot be canceled.
	ErrorCodeTaskNotCancelable KnownErrorCode = -32002

	// ErrorCodePushNotificationNotSupported is the error code for Push Notification
	// Not Supported (-32003). Push Notifications are not supported for this operation
	// or agent.
	ErrorCodePushNotificationNotSupported KnownErrorCode = -32003

	// ErrorCodeUnsupportedOperation is the error code for Unsupported Operation (-32004).
	// The requested operation is not supported by the agent.
	ErrorCodeUnsupportedOperation KnownErrorCode = -32004
)
