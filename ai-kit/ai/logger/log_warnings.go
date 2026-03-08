// Ported from: packages/ai/src/logger/log-warnings.ts
package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// Warning represents a warning from the model provider.
// TODO: import from brainlink/experiments/ai-kit/types once it exists
type Warning struct {
	// Type is the warning type: "unsupported", "compatibility", or "other".
	Type string `json:"type"`
	// Feature is the feature name (for "unsupported" and "compatibility" types).
	Feature string `json:"feature,omitempty"`
	// Details provides additional details (for "unsupported" and "compatibility" types).
	Details string `json:"details,omitempty"`
	// Message is the warning message (for "other" type).
	Message string `json:"message,omitempty"`
}

// LogWarningsOptions contains the options for logging warnings.
type LogWarningsOptions struct {
	// Warnings is the list of warnings returned by the model provider.
	Warnings []Warning
	// Provider is the provider id used for the call.
	Provider string
	// Model is the model id used for the call.
	Model string
}

// LogWarningsFunction is a function for logging warnings.
type LogWarningsFunction func(options LogWarningsOptions)

// LogWarningsConfig controls the behavior of warning logging.
// This replaces the globalThis.AI_SDK_LOG_WARNINGS pattern from TypeScript.
type LogWarningsConfig struct {
	// Disabled suppresses all warning logging when true.
	Disabled bool
	// CustomLogger is called instead of the default logger when set.
	CustomLogger LogWarningsFunction
}

// FirstWarningInfoMessage is the informational message shown on the first warning.
const FirstWarningInfoMessage = "AI SDK Warning System: To turn off warning logging, set the AI_SDK_LOG_WARNINGS global to false."

var (
	mu              sync.Mutex
	hasLoggedBefore bool
	globalConfig    *LogWarningsConfig
)

// SetLogWarningsConfig sets the global warning logging configuration.
func SetLogWarningsConfig(cfg *LogWarningsConfig) {
	mu.Lock()
	defer mu.Unlock()
	globalConfig = cfg
}

// GetLogWarningsConfig returns the current global warning logging configuration.
func GetLogWarningsConfig() *LogWarningsConfig {
	mu.Lock()
	defer mu.Unlock()
	return globalConfig
}

// ResetLogWarningsState resets the internal logging state. Used for testing purposes.
func ResetLogWarningsState() {
	mu.Lock()
	defer mu.Unlock()
	hasLoggedBefore = false
}

// formatWarning formats a warning object into a human-readable string with clear AI SDK branding.
func formatWarning(warning Warning, provider, model string) string {
	prefix := fmt.Sprintf("AI SDK Warning (%s / %s):", provider, model)

	switch warning.Type {
	case "unsupported":
		message := fmt.Sprintf(`%s The feature "%s" is not supported.`, prefix, warning.Feature)
		if warning.Details != "" {
			message += " " + warning.Details
		}
		return message

	case "compatibility":
		message := fmt.Sprintf(`%s The feature "%s" is used in a compatibility mode.`, prefix, warning.Feature)
		if warning.Details != "" {
			message += " " + warning.Details
		}
		return message

	case "other":
		return fmt.Sprintf("%s %s", prefix, warning.Message)

	default:
		// Fallback for any unknown warning types
		data, _ := json.MarshalIndent(warning, "", "  ")
		return fmt.Sprintf("%s %s", prefix, string(data))
	}
}

// LogWarnings logs warnings to the console or uses a custom logger if configured.
//
// The behavior can be customized via SetLogWarningsConfig:
//   - If Disabled is true, warnings are suppressed.
//   - If CustomLogger is set, that function is called with the warnings.
//   - Otherwise, warnings are logged to stderr using log.Println/log.Printf.
func LogWarnings(options LogWarningsOptions) {
	// if the warnings slice is empty, do nothing
	if len(options.Warnings) == 0 {
		return
	}

	mu.Lock()
	cfg := globalConfig
	mu.Unlock()

	// if the logger is disabled, do nothing
	if cfg != nil && cfg.Disabled {
		return
	}

	// use the provided logger if it is a function
	if cfg != nil && cfg.CustomLogger != nil {
		cfg.CustomLogger(options)
		return
	}

	// display information note on first call
	mu.Lock()
	firstCall := !hasLoggedBefore
	if firstCall {
		hasLoggedBefore = true
	}
	mu.Unlock()

	if firstCall {
		log.Println(FirstWarningInfoMessage)
	}

	// default behavior: log warnings to stderr
	for _, warning := range options.Warnings {
		log.Println(formatWarning(warning, options.Provider, options.Model))
	}
}
