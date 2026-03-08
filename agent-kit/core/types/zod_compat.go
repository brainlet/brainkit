// Ported from: packages/core/src/types/zod-compat.ts
package types

// ZodLikeSchema is a type compatibility layer for Zod v3 and v4.
//
// Zod v3 and v4 have different internal type structures, but they share
// the same public API. This interface uses structural typing to accept schemas
// from both versions by checking for the presence of Parse and SafeParse
// rather than relying on nominal type matching.
//
// In Go, this is expressed as an interface that any schema implementation
// can satisfy. The type parameter T from the TypeScript original is erased
// to any, since Go interfaces cannot carry phantom type parameters.
type ZodLikeSchema interface {
	// Parse validates and parses the input data, returning the parsed value.
	// Returns an error if validation fails.
	Parse(data any) (any, error)

	// SafeParse validates and parses the input data without panicking.
	// Returns a SafeParseResult indicating success or failure.
	SafeParse(data any) SafeParseResult
}

// SafeParseResult represents the result of a SafeParse operation.
// This corresponds to the TypeScript union type:
//
//	{ success: true; data: T } | { success: false; error: { issues: [...]; format(...): any } }
type SafeParseResult struct {
	// Success indicates whether parsing succeeded.
	Success bool `json:"success"`

	// Data holds the parsed value when Success is true.
	// It is nil when Success is false.
	Data any `json:"data,omitempty"`

	// Error holds the parse error details when Success is false.
	// It is nil when Success is true.
	Error *SafeParseError `json:"error,omitempty"`
}

// SafeParseError contains details about a failed SafeParse operation.
type SafeParseError struct {
	// Issues is the list of validation issues encountered during parsing.
	Issues []SafeParseIssue `json:"issues"`
}

// Format returns a formatted representation of the error.
// This corresponds to the format(...args: any[]): any method in the TypeScript original.
func (e *SafeParseError) Format(args ...any) any {
	// Default implementation returns the issues list.
	// Concrete implementations can override this behavior.
	return e.Issues
}

// SafeParseIssue represents a single validation issue from a SafeParse operation.
type SafeParseIssue struct {
	// Path is the location of the issue within the parsed data structure.
	// It may be nil if the issue applies to the root value.
	Path any `json:"path,omitempty"`

	// Message is a human-readable description of the validation issue.
	Message string `json:"message"`
}

// NewSuccessResult creates a SafeParseResult representing a successful parse.
func NewSuccessResult(data any) SafeParseResult {
	return SafeParseResult{
		Success: true,
		Data:    data,
	}
}

// NewFailureResult creates a SafeParseResult representing a failed parse.
func NewFailureResult(issues []SafeParseIssue) SafeParseResult {
	return SafeParseResult{
		Success: false,
		Error: &SafeParseError{
			Issues: issues,
		},
	}
}

// NOTE: The following TypeScript types have no direct Go equivalent and are
// intentionally omitted:
//
//   - InferZodLikeSchema<T>: Type-level computation that extracts the output
//     type from a ZodLikeSchema. Go does not support this kind of type
//     inference at compile time. Users should use type assertions on the
//     any values returned by Parse/SafeParse.
//
//   - InferZodLikeSchemaInput<T>: Type-level computation that extracts the
//     input type from a Zod schema (before transforms). Go does not support
//     conditional type inference. Users should document expected input types
//     in their schema implementations.
