// Ported from: packages/core/src/loop/network/index.test.ts
//
// NOTE: These tests compile but will panic at runtime due to a pre-existing
// bug in run_command_tool.go line 34: regexp.MustCompile(`\\(?![ ])`) uses
// a Perl-style negative lookahead (?!...) which Go's regexp package does
// not support. The MustCompile call panics during package init(), before
// any test code runs. Fix the regex in run_command_tool.go to unblock
// these tests (e.g. replace with `\\[^ ]`).
package network

import (
	"testing"
)

// The TS index.test.ts tests two exported functions:
//   - getLastMessage: extracts the last user message text from various input formats
//   - getRoutingAgent: builds a routing Agent from a parent agent's tools/workflows
//
// Neither function is ported to Go yet. All tests use t.Skip until the
// network index.go is fully implemented with getLastMessage and getRoutingAgent.

// ---------------------------------------------------------------------------
// getLastMessage tests
// ---------------------------------------------------------------------------

// TODO: Once GetLastMessage is ported to Go, remove t.Skip and implement assertions.
// The function should handle: string input, empty input, arrays of strings,
// messages with string content, messages with content arrays, messages with
// parts arrays, multiple messages, single message object, non-text parts.

func TestGetLastMessage(t *testing.T) {
	t.Run("returns string directly", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: expect(getLastMessage('hello')).toBe('hello')
	})

	t.Run("returns empty string for empty input", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: expect(getLastMessage('')).toBe('')
		// TS: expect(getLastMessage([])).toBe('')
	})

	t.Run("extracts from array of strings", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: expect(getLastMessage(['first', 'second', 'last'])).toBe('last')
	})

	t.Run("extracts from message with string content", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: expect(getLastMessage([{ role: 'user', content: 'hello' }])).toBe('hello')
	})

	t.Run("extracts from message with content array", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: messages with content: [{ type: 'text', text: 'first part' }, { type: 'text', text: 'last part' }]
		// TS: expect(getLastMessage(messages)).toBe('last part')
	})

	t.Run("extracts from message with parts array", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: messages with parts: [{ type: 'text', text: 'Tell me about Spirited Away' }]
		// TS: expect(getLastMessage(messages)).toBe('Tell me about Spirited Away')
	})

	t.Run("extracts last part from multiple parts", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: messages with parts: [{ type: 'text', text: 'first' }, { type: 'text', text: 'second' }]
		// TS: expect(getLastMessage(messages)).toBe('second')
	})

	t.Run("returns last message from multiple messages", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: three messages, last has content 'last message'
		// TS: expect(getLastMessage(messages)).toBe('last message')
	})

	t.Run("handles single message object not array", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: expect(getLastMessage({ role: 'user', content: 'single' })).toBe('single')
	})

	t.Run("returns empty string for non-text parts", func(t *testing.T) {
		t.Skip("not yet implemented: GetLastMessage not ported to Go")
		// TS: messages with parts: [{ type: 'image', url: 'http://example.com' }]
		// TS: expect(getLastMessage(messages)).toBe('')
	})
}

// ---------------------------------------------------------------------------
// getRoutingAgent tests
// ---------------------------------------------------------------------------

// TODO: Once GetRoutingAgent is ported to Go, remove t.Skip and implement
// assertions. The function requires Agent, createWorkflow, createTool,
// RequestContext, and Processor types from unported packages.

func TestGetRoutingAgent(t *testing.T) {
	t.Run("should handle workflow with undefined inputSchema without throwing", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent, createWorkflow)")
		// TS: Creates workflow without inputSchema, expects getRoutingAgent to succeed
	})

	t.Run("should handle workflow with explicit inputSchema correctly", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent, createWorkflow)")
		// TS: Creates workflow with inputSchema z.object({ name: z.string() }), expects success
	})

	t.Run("should handle tool with undefined inputSchema without throwing", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent, createTool)")
		// TS: Creates tool without inputSchema, expects getRoutingAgent to succeed
	})

	t.Run("should handle tool with explicit inputSchema correctly", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent, createTool)")
		// TS: Creates tool with inputSchema z.object({ theme: z.string() }), expects success
	})

	t.Run("should handle a mix of tools and workflows with and without inputSchema", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent, createTool, createWorkflow)")
		// TS: Mix of tools/workflows with and without inputSchema, expects success
	})

	t.Run("should pass through configured input processors from the parent agent", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent, Processor)")
		// TS: Creates mock input processor, verifies listConfiguredInputProcessors called
	})

	t.Run("should pass through configured output processors from the parent agent", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent, Processor)")
		// TS: Creates mock output processor, verifies listConfiguredOutputProcessors called
	})

	t.Run("should not call listInputProcessors which includes memory processors", func(t *testing.T) {
		t.Skip("not yet implemented: GetRoutingAgent not ported to Go (requires Agent)")
		// TS: Verifies listInputProcessors is NOT called, only listConfiguredInputProcessors
	})
}
