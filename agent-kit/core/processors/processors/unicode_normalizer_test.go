// Ported from: packages/core/src/processors/processors/unicode-normalizer.test.ts
package concreteprocessors

import (
	"strings"
	"testing"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeMessage(role string, parts []processors.MessagePart) processors.MastraDBMessage {
	return processors.MastraDBMessage{
		ID:   "test-msg",
		Role: role,
		Content: processors.MastraMessageContentV2{
			Format: 2,
			Parts:  parts,
		},
	}
}

func makeTextMessage(role, text string) processors.MastraDBMessage {
	return makeMessage(role, []processors.MessagePart{
		{Type: "text", Text: text},
	})
}

func makeContentMessage(role, content string) processors.MastraDBMessage {
	return processors.MastraDBMessage{
		ID:   "test-msg",
		Role: role,
		Content: processors.MastraMessageContentV2{
			Format:  2,
			Content: content,
		},
	}
}

func defaultArgs(messages []processors.MastraDBMessage) processors.ProcessInputArgs {
	return processors.ProcessInputArgs{
		ProcessorMessageContext: processors.ProcessorMessageContext{
			Messages: messages,
		},
		State: map[string]any{},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestUnicodeNormalizer(t *testing.T) {

	t.Run("constructor defaults", func(t *testing.T) {
		t.Run("should use default options when nil is passed", func(t *testing.T) {
			un := NewUnicodeNormalizer(nil)
			if un.options.StripControlChars != false {
				t.Fatal("expected StripControlChars=false")
			}
			if un.options.PreserveEmojis != true {
				t.Fatal("expected PreserveEmojis=true")
			}
			if un.options.CollapseWhitespace != true {
				t.Fatal("expected CollapseWhitespace=true")
			}
			if un.options.Trim != true {
				t.Fatal("expected Trim=true")
			}
		})

		t.Run("should accept custom options", func(t *testing.T) {
			un := NewUnicodeNormalizer(&UnicodeNormalizerOptions{
				StripControlChars:  true,
				PreserveEmojis:     false,
				CollapseWhitespace: false,
				Trim:               false,
			})
			if un.options.StripControlChars != true {
				t.Fatal("expected StripControlChars=true")
			}
			if un.options.PreserveEmojis != false {
				t.Fatal("expected PreserveEmojis=false")
			}
			if un.options.CollapseWhitespace != false {
				t.Fatal("expected CollapseWhitespace=false")
			}
			if un.options.Trim != false {
				t.Fatal("expected Trim=false")
			}
		})

		t.Run("should have correct ID and Name", func(t *testing.T) {
			un := NewUnicodeNormalizer(nil)
			if un.ID() != "unicode-normalizer" {
				t.Fatalf("expected id 'unicode-normalizer', got '%s'", un.ID())
			}
			if un.Name() != "Unicode Normalizer" {
				t.Fatalf("expected name 'Unicode Normalizer', got '%s'", un.Name())
			}
		})
	})

	t.Run("NFKC normalization", func(t *testing.T) {
		un := NewUnicodeNormalizer(nil)

		t.Run("should normalize ligatures", func(t *testing.T) {
			// ﬁ (U+FB01, fi ligature) → fi
			result := un.normalizeText("\uFB01nance")
			if result != "finance" {
				t.Fatalf("expected 'finance', got '%s'", result)
			}
		})

		t.Run("should normalize fullwidth characters", func(t *testing.T) {
			// Ｈｅｌｌｏ (fullwidth) → Hello
			result := un.normalizeText("\uFF28\uFF45\uFF4C\uFF4C\uFF4F")
			if result != "Hello" {
				t.Fatalf("expected 'Hello', got '%s'", result)
			}
		})

		t.Run("should normalize composed characters", func(t *testing.T) {
			// é (U+00E9, precomposed) should stay as é after NFKC
			result := un.normalizeText("\u00E9")
			if result != "\u00E9" {
				t.Fatalf("expected 'é', got '%s'", result)
			}
		})

		t.Run("should normalize decomposed characters", func(t *testing.T) {
			// e + combining acute (U+0065 + U+0301) → é (U+00E9) under NFKC
			result := un.normalizeText("e\u0301")
			if result != "\u00E9" {
				t.Fatalf("expected 'é', got '%s'", result)
			}
		})
	})

	t.Run("whitespace handling", func(t *testing.T) {
		un := NewUnicodeNormalizer(nil)

		t.Run("should collapse multiple spaces", func(t *testing.T) {
			result := un.normalizeText("hello    world")
			if result != "hello world" {
				t.Fatalf("expected 'hello world', got '%s'", result)
			}
		})

		t.Run("should collapse multiple newlines", func(t *testing.T) {
			result := un.normalizeText("hello\n\n\nworld")
			if result != "hello\nworld" {
				t.Fatalf("expected 'hello\\nworld', got '%s'", result)
			}
		})

		t.Run("should normalize mixed line endings", func(t *testing.T) {
			result := un.normalizeText("hello\r\nworld\rfoo")
			if result != "hello\nworld\nfoo" {
				t.Fatalf("expected 'hello\\nworld\\nfoo', got '%s'", result)
			}
		})

		t.Run("should trim leading and trailing whitespace", func(t *testing.T) {
			result := un.normalizeText("  hello world  ")
			if result != "hello world" {
				t.Fatalf("expected 'hello world', got '%s'", result)
			}
		})

		t.Run("should not collapse whitespace when disabled", func(t *testing.T) {
			un2 := NewUnicodeNormalizer(&UnicodeNormalizerOptions{
				CollapseWhitespace: false,
				Trim:               false,
			})
			result := un2.normalizeText("hello    world")
			if result != "hello    world" {
				t.Fatalf("expected 'hello    world', got '%s'", result)
			}
		})

		t.Run("should not trim when disabled", func(t *testing.T) {
			un2 := NewUnicodeNormalizer(&UnicodeNormalizerOptions{
				CollapseWhitespace: false,
				Trim:               false,
			})
			result := un2.normalizeText("  hello  ")
			if result != "  hello  " {
				t.Fatalf("expected '  hello  ', got '%s'", result)
			}
		})
	})

	t.Run("control character handling", func(t *testing.T) {
		t.Run("should preserve control characters by default", func(t *testing.T) {
			// With StripControlChars=false (default), control characters are not stripped.
			// However, CollapseWhitespace=true (default) collapses tabs/spaces into single space.
			// Use explicit options with CollapseWhitespace=false to test pure control char preservation.
			un := NewUnicodeNormalizer(&UnicodeNormalizerOptions{
				StripControlChars:  false,
				PreserveEmojis:     true,
				CollapseWhitespace: false,
				Trim:               false,
			})
			// Tab and newline should be preserved when whitespace collapse is off
			result := un.normalizeText("hello\tworld\nfoo")
			if !strings.Contains(result, "\t") {
				t.Fatal("expected tab to be preserved")
			}
			if !strings.Contains(result, "\n") {
				t.Fatal("expected newline to be preserved")
			}
		})

		t.Run("should strip control characters when enabled with emoji preservation", func(t *testing.T) {
			un := NewUnicodeNormalizer(&UnicodeNormalizerOptions{
				StripControlChars:  true,
				PreserveEmojis:     true,
				CollapseWhitespace: false,
				Trim:               false,
			})
			// NUL should be stripped
			result := un.normalizeText("hello\x00world")
			if strings.Contains(result, "\x00") {
				t.Fatal("expected NUL character to be stripped")
			}
			if result != "helloworld" {
				t.Fatalf("expected 'helloworld', got '%s'", result)
			}
		})

		t.Run("should strip control characters aggressively when emoji preservation disabled", func(t *testing.T) {
			un := NewUnicodeNormalizer(&UnicodeNormalizerOptions{
				StripControlChars:  true,
				PreserveEmojis:     false,
				CollapseWhitespace: false,
				Trim:               false,
			})
			// NUL should be stripped
			result := un.normalizeText("hello\x00world")
			if strings.Contains(result, "\x00") {
				t.Fatal("expected NUL character to be stripped")
			}
		})
	})

	t.Run("emoji handling", func(t *testing.T) {
		un := NewUnicodeNormalizer(&UnicodeNormalizerOptions{
			StripControlChars:  true,
			PreserveEmojis:     true,
			CollapseWhitespace: true,
			Trim:               true,
		})

		t.Run("should preserve simple emojis", func(t *testing.T) {
			result := un.normalizeText("hello \U0001F600 world")
			if !strings.Contains(result, "\U0001F600") {
				t.Fatal("expected emoji to be preserved")
			}
		})

		t.Run("should preserve complex emoji modifiers", func(t *testing.T) {
			// Family emoji with ZWJ sequence
			result := un.normalizeText("test \U0001F468\u200D\U0001F469\u200D\U0001F467 end")
			if !strings.Contains(result, "\U0001F468") {
				t.Fatal("expected complex emoji to be preserved")
			}
		})
	})

	t.Run("message structure", func(t *testing.T) {
		un := NewUnicodeNormalizer(nil)

		t.Run("should normalize text in message parts", func(t *testing.T) {
			messages := []processors.MastraDBMessage{
				makeTextMessage("user", "  hello    world  "),
			}
			result, _, _, err := un.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
			if len(result[0].Content.Parts) != 1 {
				t.Fatalf("expected 1 part, got %d", len(result[0].Content.Parts))
			}
			if result[0].Content.Parts[0].Text != "hello world" {
				t.Fatalf("expected 'hello world', got '%s'", result[0].Content.Parts[0].Text)
			}
		})

		t.Run("should normalize content field", func(t *testing.T) {
			messages := []processors.MastraDBMessage{
				makeContentMessage("user", "  hello    world  "),
			}
			result, _, _, err := un.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
			if result[0].Content.Content != "hello world" {
				t.Fatalf("expected 'hello world', got '%s'", result[0].Content.Content)
			}
		})

		t.Run("should preserve metadata", func(t *testing.T) {
			msg := makeTextMessage("user", "  hello  ")
			msg.Content.Metadata = map[string]any{"key": "value"}
			messages := []processors.MastraDBMessage{msg}

			result, _, _, err := un.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result[0].Content.Metadata["key"] != "value" {
				t.Fatal("expected metadata to be preserved")
			}
		})

		t.Run("should skip non-text parts", func(t *testing.T) {
			messages := []processors.MastraDBMessage{
				makeMessage("user", []processors.MessagePart{
					{Type: "text", Text: "  hello  "},
					{Type: "image", Image: "data:image/png;base64,abc"},
				}),
			}
			result, _, _, err := un.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result[0].Content.Parts[0].Text != "hello" {
				t.Fatalf("expected 'hello', got '%s'", result[0].Content.Parts[0].Text)
			}
			if result[0].Content.Parts[1].Type != "image" {
				t.Fatal("expected image part to be preserved")
			}
		})

		t.Run("should handle multiple messages", func(t *testing.T) {
			messages := []processors.MastraDBMessage{
				makeTextMessage("user", "  hello  "),
				makeTextMessage("assistant", "  world  "),
			}
			result, _, _, err := un.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(result))
			}
			if result[0].Content.Parts[0].Text != "hello" {
				t.Fatalf("expected 'hello', got '%s'", result[0].Content.Parts[0].Text)
			}
			if result[1].Content.Parts[0].Text != "world" {
				t.Fatalf("expected 'world', got '%s'", result[1].Content.Parts[0].Text)
			}
		})
	})

	t.Run("error handling", func(t *testing.T) {
		un := NewUnicodeNormalizer(nil)

		t.Run("should handle empty messages", func(t *testing.T) {
			result, _, _, err := un.ProcessInput(defaultArgs([]processors.MastraDBMessage{}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 0 {
				t.Fatalf("expected 0 messages, got %d", len(result))
			}
		})

		t.Run("should handle message with no parts", func(t *testing.T) {
			messages := []processors.MastraDBMessage{
				makeMessage("user", nil),
			}
			result, _, _, err := un.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		un := NewUnicodeNormalizer(nil)

		t.Run("should handle empty string", func(t *testing.T) {
			result := un.normalizeText("")
			if result != "" {
				t.Fatalf("expected empty string, got '%s'", result)
			}
		})

		t.Run("should handle whitespace-only string", func(t *testing.T) {
			result := un.normalizeText("   ")
			if result != "" {
				t.Fatalf("expected empty string after trim, got '%s'", result)
			}
		})

		t.Run("should handle long strings", func(t *testing.T) {
			long := strings.Repeat("hello ", 10000)
			result := un.normalizeText(long)
			if len(result) == 0 {
				t.Fatal("expected non-empty result for long string")
			}
		})

		t.Run("should handle mixed unicode", func(t *testing.T) {
			// Mix of ASCII, CJK, Arabic
			result := un.normalizeText("Hello \u4E16\u754C \u0645\u0631\u062D\u0628\u0627")
			if !strings.Contains(result, "Hello") {
				t.Fatal("expected ASCII preserved")
			}
			if !strings.Contains(result, "\u4E16\u754C") {
				t.Fatal("expected CJK preserved")
			}
		})
	})
}
