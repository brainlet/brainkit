// Ported from: packages/provider/src/shared/v3/shared-v3-warning.ts
package shared

// Warning from the model.
//
// For example, that certain features are unsupported or compatibility
// functionality is used (which might lead to suboptimal results).
//
// This is a sealed interface (discriminated union in TS).
// Implementations: UnsupportedWarning, CompatibilityWarning, OtherWarning.
type Warning interface {
	warningType() string
}

// UnsupportedWarning indicates a feature is not supported by the model.
type UnsupportedWarning struct {
	// Feature is the feature that is not supported.
	Feature string

	// Details provides additional details about the warning.
	Details *string
}

func (w UnsupportedWarning) warningType() string { return "unsupported" }

// CompatibilityWarning indicates a compatibility feature is used
// that might lead to suboptimal results.
type CompatibilityWarning struct {
	// Feature is the feature that is used in a compatibility mode.
	Feature string

	// Details provides additional details about the warning.
	Details *string
}

func (w CompatibilityWarning) warningType() string { return "compatibility" }

// OtherWarning represents any other type of warning.
type OtherWarning struct {
	// Message is the warning message.
	Message string
}

func (w OtherWarning) warningType() string { return "other" }
