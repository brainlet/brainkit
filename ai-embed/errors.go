package aiembed

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AIError is the base error type for ai-embed errors.
type AIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

func (e *AIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("ai-embed: %s: %s: %s", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("ai-embed: %s: %s", e.Type, e.Message)
}

func (e *AIError) Unwrap() error { return e.Cause }

// APICallError represents an HTTP error from the AI provider.
type APICallError struct {
	StatusCode   int    `json:"statusCode"`
	URL          string `json:"url"`
	RequestBody  string `json:"requestBody,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`
	Message      string `json:"message"`
	IsRetryable  bool   `json:"isRetryable"`
}

func (e *APICallError) Error() string {
	return fmt.Sprintf("ai-embed: API call failed (HTTP %d): %s", e.StatusCode, e.Message)
}

// NoSuchModelError indicates the requested model was not found.
type NoSuchModelError struct {
	ModelID  string `json:"modelId"`
	ModelType string `json:"modelType"`
	Message  string `json:"message"`
}

func (e *NoSuchModelError) Error() string {
	return fmt.Sprintf("ai-embed: model not found: %s (%s)", e.ModelID, e.ModelType)
}

// InvalidResponseError indicates the provider returned an unparseable response.
type InvalidResponseError struct {
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *InvalidResponseError) Error() string {
	return fmt.Sprintf("ai-embed: invalid response: %s", e.Message)
}

// ProviderError indicates an error from the provider configuration.
type ProviderError struct {
	Provider string `json:"provider"`
	Message  string `json:"message"`
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("ai-embed: provider %q: %s", e.Provider, e.Message)
}

// classifyJSError attempts to parse a JS error message into a typed Go error.
// Falls back to a generic AIError if the error can't be classified.
func classifyJSError(operation string, jsErr error) error {
	if jsErr == nil {
		return nil
	}

	msg := jsErr.Error()

	// Try to parse structured error from JS
	var structured struct {
		Type         string `json:"type"`
		Message      string `json:"message"`
		StatusCode   int    `json:"statusCode"`
		URL          string `json:"url"`
		ResponseBody string `json:"responseBody"`
		IsRetryable  bool   `json:"isRetryable"`
		ModelID      string `json:"modelId"`
		ModelType    string `json:"modelType"`
	}

	// Check if the error message contains JSON (common pattern: "Error: {json}")
	if idx := strings.Index(msg, "{"); idx >= 0 {
		jsonPart := msg[idx:]
		if err := json.Unmarshal([]byte(jsonPart), &structured); err == nil {
			switch structured.Type {
			case "APICallError":
				return &APICallError{
					StatusCode:   structured.StatusCode,
					URL:          structured.URL,
					ResponseBody: structured.ResponseBody,
					Message:      structured.Message,
					IsRetryable:  structured.IsRetryable,
				}
			case "NoSuchModelError":
				return &NoSuchModelError{
					ModelID:   structured.ModelID,
					ModelType: structured.ModelType,
					Message:   structured.Message,
				}
			}
		}
	}

	// Check for common error patterns in the message
	if strings.Contains(msg, "401") || strings.Contains(msg, "Unauthorized") || strings.Contains(msg, "Incorrect API key") {
		return &APICallError{
			StatusCode: 401,
			Message:    msg,
		}
	}
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "Rate limit") {
		return &APICallError{
			StatusCode:  429,
			Message:     msg,
			IsRetryable: true,
		}
	}
	if strings.Contains(msg, "404") || strings.Contains(msg, "not found") || strings.Contains(msg, "does not exist") {
		if strings.Contains(msg, "model") {
			return &NoSuchModelError{
				Message: msg,
			}
		}
		return &APICallError{
			StatusCode: 404,
			Message:    msg,
		}
	}
	if strings.Contains(msg, "500") || strings.Contains(msg, "internal server error") || strings.Contains(msg, "Internal Server Error") {
		return &APICallError{
			StatusCode:  500,
			Message:     msg,
			IsRetryable: true,
		}
	}

	// Generic fallback
	return &AIError{
		Type:    "JSError",
		Message: fmt.Sprintf("%s: %s", operation, msg),
		Cause:   jsErr,
	}
}
