// Ported from: packages/core/src/tools/toolchecks.ts
package tools

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraToolMarker is the Go equivalent of MASTRA_TOOL_MARKER.
// In TypeScript this is Symbol.for('mastra.core.tool.Tool') used for
// cross-module identity checks. In Go we use a sentinel string stored
// in a map field convention.
// In Go, marker-based identity checks use MastraToolChecker interface below.
const MastraToolMarker = "mastra.core.tool.Tool"

// ToolToConvert represents the union type of convertible tools:
//   VercelTool | ToolAction | VercelToolV5 | ProviderDefinedTool
// In Go we represent this as any since these are all interface types.
// Union type: uses any since Go lacks discriminated unions.
type ToolToConvert = any

// ProviderDefinedTool is a stub for AI SDK ProviderDefinedTool.
// ai-kit only ported V3 (@ai-sdk/provider-v6). These types remain local stubs.
type ProviderDefinedTool = any

// ---------------------------------------------------------------------------
// Marker interface for Mastra tools
// ---------------------------------------------------------------------------

// MastraToolChecker is an interface that Mastra tools implement
// to identify themselves. This is the Go equivalent of the
// MASTRA_TOOL_MARKER Symbol check in TypeScript.
type MastraToolChecker interface {
	IsMastraTool() bool
}

// ---------------------------------------------------------------------------
// Tool check functions
// ---------------------------------------------------------------------------

// IsMastraTool checks if a tool is a Mastra Tool, using the MastraToolChecker
// interface. This is the Go equivalent of the TypeScript instanceof + marker check.
//
// The marker fallback handles environments like Vite SSR where the same
// module may be loaded multiple times, causing instanceof to fail.
// In Go, we use the interface check which is structurally equivalent.
func IsMastraTool(tool any) bool {
	if tool == nil {
		return false
	}
	if checker, ok := tool.(MastraToolChecker); ok {
		return checker.IsMastraTool()
	}
	// Fallback: check for marker field in map-based tools.
	if m, ok := tool.(map[string]any); ok {
		if marker, exists := m[MastraToolMarker]; exists {
			if b, ok := marker.(bool); ok && b {
				return true
			}
		}
	}
	return false
}

// IsVercelTool checks if a tool is a Vercel Tool (AI SDK tool).
//
// AI SDK tools must have an execute function and either:
//   - "parameters" (v4) or "inputSchema" (v5/v6)
//
// This prevents plain objects with inputSchema (like client tools) from
// being treated as VercelTools.
func IsVercelTool(tool any) bool {
	if tool == nil {
		return false
	}
	if IsMastraTool(tool) {
		return false
	}

	m, ok := tool.(map[string]any)
	if !ok {
		return false
	}

	// Check for "parameters" field (v4 format).
	if _, hasParams := m["parameters"]; hasParams {
		return true
	}

	// Check for "execute" function + "inputSchema" (v5/v6 format).
	exec, hasExec := m["execute"]
	_, hasInput := m["inputSchema"]
	if hasExec && hasInput {
		// Verify execute is callable (function-like).
		if exec != nil {
			return true
		}
	}

	return false
}

// IsProviderDefinedTool checks if a tool is a provider-defined tool from the AI SDK.
//
// Provider tools (like google.tools.googleSearch(), openai.tools.webSearch()) have:
//   - type: "provider-defined" (AI SDK v5) or "provider" (AI SDK v6)
//   - id: in format "provider.tool_name" (e.g., "google.google_search")
//
// These tools have a lazy inputSchema function that returns an AI SDK Schema
// (not a Zod schema), so they require special handling during serialization.
func IsProviderDefinedTool(tool any) bool {
	if tool == nil {
		return false
	}

	m, ok := tool.(map[string]any)
	if !ok {
		return false
	}

	toolType, hasType := m["type"]
	if !hasType {
		return false
	}

	typeStr, isStr := toolType.(string)
	if !isStr {
		return false
	}

	isProviderType := typeStr == "provider-defined" || typeStr == "provider"
	if !isProviderType {
		return false
	}

	id, hasID := m["id"]
	if !hasID {
		return false
	}
	_, idIsStr := id.(string)
	return idIsStr
}
