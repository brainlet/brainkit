// Ported from: packages/core/src/error/index.ts
package mastraerror

import (
	"encoding/json"
)

// ErrorDomain represents the functional domain of an error.
type ErrorDomain string

const (
	ErrorDomainTool                ErrorDomain = "TOOL"
	ErrorDomainAgent               ErrorDomain = "AGENT"
	ErrorDomainMCP                 ErrorDomain = "MCP"
	ErrorDomainAgentNetwork        ErrorDomain = "AGENT_NETWORK"
	ErrorDomainMastraServer        ErrorDomain = "MASTRA_SERVER"
	ErrorDomainMastraObservability ErrorDomain = "MASTRA_OBSERVABILITY"
	ErrorDomainMastraWorkflow      ErrorDomain = "MASTRA_WORKFLOW"
	ErrorDomainMastraVoice         ErrorDomain = "MASTRA_VOICE"
	ErrorDomainMastraVector        ErrorDomain = "MASTRA_VECTOR"
	ErrorDomainMastraMemory        ErrorDomain = "MASTRA_MEMORY"
	ErrorDomainLLM                 ErrorDomain = "LLM"
	ErrorDomainEval                ErrorDomain = "EVAL"
	ErrorDomainScorer              ErrorDomain = "SCORER"
	ErrorDomainA2A                 ErrorDomain = "A2A"
	ErrorDomainMastraInstance      ErrorDomain = "MASTRA_INSTANCE"
	ErrorDomainMastra              ErrorDomain = "MASTRA"
	ErrorDomainDeployer            ErrorDomain = "DEPLOYER"
	ErrorDomainStorage             ErrorDomain = "STORAGE"
	ErrorDomainModelRouter         ErrorDomain = "MODEL_ROUTER"
)

// ErrorCategory represents the broad category of an error.
type ErrorCategory string

const (
	ErrorCategoryUnknown    ErrorCategory = "UNKNOWN"
	ErrorCategoryUser       ErrorCategory = "USER"
	ErrorCategorySystem     ErrorCategory = "SYSTEM"
	ErrorCategoryThirdParty ErrorCategory = "THIRD_PARTY"
)

// ErrorDefinition defines the structure for an error's metadata.
// This is used to create instances of MastraError.
type ErrorDefinition struct {
	// ID is a unique identifier for the error.
	ID string `json:"id"`
	// Text is an optional custom error message that overrides the original error message.
	// If not provided, the original error message will be used, or "Unknown error" if no error is provided.
	Text string `json:"text,omitempty"`
	// Domain is the functional domain of the error (e.g., TOOL, AGENT, MCP).
	Domain ErrorDomain `json:"domain"`
	// Category is the broad category of the error (e.g., USER, SYSTEM, THIRD_PARTY).
	Category ErrorCategory `json:"category"`
	// Details contains optional additional metadata.
	Details map[string]any `json:"details,omitempty"`
}

// MastraErrorJSON is the JSON representation of a MastraError for serialization.
type MastraErrorJSON struct {
	Message  string          `json:"message"`
	Code     string          `json:"code"`
	Category ErrorCategory   `json:"category"`
	Domain   ErrorDomain     `json:"domain"`
	Details  map[string]any  `json:"details,omitempty"`
	Cause    *SerializedError `json:"cause,omitempty"`
}

// MastraErrorJSONDetails is the structured representation returned by ToJSONDetails().
type MastraErrorJSONDetails struct {
	Message  string         `json:"message"`
	Domain   ErrorDomain    `json:"domain"`
	Category ErrorCategory  `json:"category"`
	Details  map[string]any `json:"details,omitempty"`
}

// MastraBaseError is the base error type for the Mastra ecosystem.
// It standardizes error reporting and can be extended for more specific error types.
type MastraBaseError struct {
	id       string
	domain   ErrorDomain
	category ErrorCategory
	details  map[string]any
	message  string
	cause    *SerializableError
}

// NewMastraBaseError creates a new MastraBaseError from an ErrorDefinition and an optional original error.
func NewMastraBaseError(def ErrorDefinition, originalError ...any) *MastraBaseError {
	var sErr *SerializableError

	if len(originalError) > 0 && originalError[0] != nil {
		result := GetErrorFromUnknown(originalError[0], &GetErrorOptions{
			SerializeStack: boolPtr(false),
			FallbackMessage: "Unknown error",
		})
		sErr = result
	}

	message := def.Text
	if message == "" {
		if sErr != nil {
			message = sErr.Message()
		} else {
			message = "Unknown error"
		}
	}

	details := def.Details
	if details == nil {
		details = map[string]any{}
	}

	return &MastraBaseError{
		id:       def.ID,
		domain:   def.Domain,
		category: def.Category,
		details:  details,
		message:  message,
		cause:    sErr,
	}
}

// Error implements the error interface.
func (e *MastraBaseError) Error() string {
	b, err := json.Marshal(e.ToJSON())
	if err != nil {
		return e.message
	}
	return string(b)
}

// String returns the JSON string representation of the error.
func (e *MastraBaseError) String() string {
	return e.Error()
}

// ID returns the error's unique identifier.
func (e *MastraBaseError) ID() string {
	return e.id
}

// Domain returns the error's domain.
func (e *MastraBaseError) Domain() ErrorDomain {
	return e.domain
}

// Category returns the error's category.
func (e *MastraBaseError) Category() ErrorCategory {
	return e.category
}

// Details returns the error's details map.
func (e *MastraBaseError) Details() map[string]any {
	return e.details
}

// Message returns the error message.
func (e *MastraBaseError) Message() string {
	return e.message
}

// Cause returns the cause of the error, if any.
func (e *MastraBaseError) Cause() *SerializableError {
	return e.cause
}

// Unwrap returns the underlying cause for errors.Is/errors.As support.
func (e *MastraBaseError) Unwrap() error {
	if e.cause == nil {
		return nil
	}
	return e.cause
}

// ToJSONDetails returns a structured representation of the error, useful for logging or API responses.
func (e *MastraBaseError) ToJSONDetails() MastraErrorJSONDetails {
	return MastraErrorJSONDetails{
		Message:  e.message,
		Domain:   e.domain,
		Category: e.category,
		Details:  e.details,
	}
}

// ToJSON returns the full JSON representation of the error.
func (e *MastraBaseError) ToJSON() MastraErrorJSON {
	result := MastraErrorJSON{
		Message:  e.message,
		Domain:   e.domain,
		Category: e.category,
		Code:     e.id,
		Details:  e.details,
	}

	if e.cause != nil {
		causeJSON := e.cause.ToJSON()
		result.Cause = &causeJSON
	}

	return result
}

// MarshalJSON implements json.Marshaler for proper JSON serialization.
func (e *MastraBaseError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.ToJSON())
}

// MastraError is a MastraBaseError with concrete ErrorDomain and ErrorCategory types.
// In TypeScript this was: class MastraError extends MastraBaseError<ErrorDomain, ErrorCategory>
// In Go there's no generic distinction needed — MastraBaseError already uses ErrorDomain/ErrorCategory.
type MastraError = MastraBaseError

// NewMastraError creates a new MastraError (alias for NewMastraBaseError).
func NewMastraError(def ErrorDefinition, originalError ...any) *MastraError {
	return NewMastraBaseError(def, originalError...)
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
