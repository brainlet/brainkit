package jsbridge

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	quickjs "github.com/buke/quickjs-go"
)

// DiagnosticContext carries safe context for a JavaScript failure that crossed
// a Go boundary. Keep fields to identifiers and runtime state; do not include
// prompts, payloads, API keys, or full request bodies.
type DiagnosticContext struct {
	RuntimeOwner   string
	Phase          string
	Source         string
	Function       string
	Package        string
	Provider       string
	Model          string
	BridgeSnapshot *DebugSnapshot
}

// DiagnosticError preserves the original error while adding stable context and
// QuickJS stack/cause details for debugging bundled dependency failures.
type DiagnosticError struct {
	Context DiagnosticContext
	Cause   error

	JSName       string
	JSMessage    string
	JSCause      string
	JSStack      string
	JSJSONString string
}

// WrapError attaches JavaScript diagnostic context to err. The returned error
// still unwraps to err, so callers can use errors.Is / errors.As on the
// original QuickJS or SDK error.
func WrapError(err error, ctx DiagnosticContext) error {
	if err == nil {
		return nil
	}
	out := &DiagnosticError{Context: ctx, Cause: err}
	var jsErr *quickjs.Error
	if errors.As(err, &jsErr) {
		out.JSName = strings.TrimSpace(jsErr.Name)
		out.JSMessage = strings.TrimSpace(jsErr.Message)
		out.JSCause = strings.TrimSpace(jsErr.Cause)
		out.JSStack = strings.TrimSpace(jsErr.Stack)
		out.JSJSONString = strings.TrimSpace(jsErr.JSONString)
	} else {
		var nested *DiagnosticError
		if errors.As(err, &nested) {
			out.JSName = nested.JSName
			out.JSMessage = nested.JSMessage
			out.JSCause = nested.JSCause
			out.JSStack = nested.JSStack
			out.JSJSONString = nested.JSJSONString
		}
	}
	return out
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("brainkit js error")
	if fields := e.contextFields(); len(fields) > 0 {
		b.WriteString(" (")
		b.WriteString(strings.Join(fields, ", "))
		b.WriteString(")")
	}
	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}
	if e.JSCause != "" && !strings.Contains(b.String(), e.JSCause) {
		b.WriteString("\nJavaScript cause: ")
		b.WriteString(e.JSCause)
	}
	if snap := e.Context.BridgeSnapshot; snap != nil {
		b.WriteString("\nBridge snapshot: ")
		b.WriteString(formatDiagnosticSnapshot(*snap))
	}
	var nested *DiagnosticError
	if e.JSStack != "" && !errors.As(e.Cause, &nested) {
		b.WriteString("\nJavaScript stack:\n")
		b.WriteString(e.JSStack)
	}
	return b.String()
}

func (e *DiagnosticError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *DiagnosticError) contextFields() []string {
	if e == nil {
		return nil
	}
	var fields []string
	add := func(key, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			fields = append(fields, key+"="+value)
		}
	}
	add("owner", e.Context.RuntimeOwner)
	add("phase", e.Context.Phase)
	add("source", e.Context.Source)
	add("function", e.Context.Function)
	add("package", e.Context.Package)
	add("provider", e.Context.Provider)
	add("model", e.Context.Model)
	return fields
}

func formatDiagnosticSnapshot(s DebugSnapshot) string {
	parts := []string{
		fmt.Sprintf("closing=%t", s.Closing),
		fmt.Sprintf("closed=%t", s.Closed),
		fmt.Sprintf("activeGoroutines=%d", s.ActiveGoroutines),
	}
	if len(s.Resources) > 0 {
		keys := make([]string, 0, len(s.Resources))
		for key := range s.Resources {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		resourceParts := make([]string, 0, len(keys))
		for _, key := range keys {
			resourceParts = append(resourceParts, fmt.Sprintf("%s=%d", key, s.Resources[key]))
		}
		parts = append(parts, "resources={"+strings.Join(resourceParts, ", ")+"}")
	}
	return strings.Join(parts, " ")
}
