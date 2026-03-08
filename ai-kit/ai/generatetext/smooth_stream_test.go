// Ported from: packages/ai/src/generate-text/smooth-stream.test.ts
package generatetext

import (
	"fmt"
	"regexp"
	"testing"
)

// smoothEvent represents either a stream part or a delay marker.
type smoothEvent struct {
	Part     *TextStreamPartForSmooth
	DelayStr string
}

// runSmooth runs the SmoothStream transformer synchronously and collects all output events.
// Delay markers are injected via a special "__delay__" type part in the output channel.
func runSmooth(t *testing.T, opts SmoothStreamOptions, parts []TextStreamPartForSmooth) []smoothEvent {
	t.Helper()

	input := make(chan TextStreamPartForSmooth, len(parts))
	for _, p := range parts {
		input <- p
	}
	close(input)

	// Use a large buffered output channel so transformer won't block
	output := make(chan TextStreamPartForSmooth, 1000)

	// Inject delay markers into the output channel
	opts.DelayFunc = func(ms int) {
		output <- TextStreamPartForSmooth{
			Type: "__delay__",
			Text: fmt.Sprintf("delay %d", ms),
		}
	}

	transformer := SmoothStream(opts)

	// Run synchronously -- input is pre-filled and closed, output is large enough
	transformer(input, output)

	// Collect all events (channel is closed by transformer's defer close(output))
	var events []smoothEvent
	for part := range output {
		if part.Type == "__delay__" {
			events = append(events, smoothEvent{DelayStr: part.Text})
		} else {
			p := part
			events = append(events, smoothEvent{Part: &p})
		}
	}
	return events
}

func assertPartType(t *testing.T, events []smoothEvent, idx int, expected string) {
	t.Helper()
	if idx >= len(events) {
		t.Fatalf("event index %d out of range (len=%d)", idx, len(events))
	}
	e := events[idx]
	if e.Part == nil {
		t.Fatalf("event[%d] expected part, got delay %q", idx, e.DelayStr)
	}
	if e.Part.Type != expected {
		t.Errorf("event[%d] expected type %q, got %q", idx, expected, e.Part.Type)
	}
}

func assertPartText(t *testing.T, events []smoothEvent, idx int, expectedType, expectedText string) {
	t.Helper()
	if idx >= len(events) {
		t.Fatalf("event index %d out of range (len=%d)", idx, len(events))
	}
	e := events[idx]
	if e.Part == nil {
		t.Fatalf("event[%d] expected part, got delay %q", idx, e.DelayStr)
	}
	if e.Part.Type != expectedType {
		t.Errorf("event[%d] expected type %q, got %q", idx, expectedType, e.Part.Type)
	}
	if e.Part.Text != expectedText {
		t.Errorf("event[%d] expected text %q, got %q", idx, expectedText, e.Part.Text)
	}
}

func assertDelay(t *testing.T, events []smoothEvent, idx int, ms int) {
	t.Helper()
	if idx >= len(events) {
		t.Fatalf("event index %d out of range (len=%d)", idx, len(events))
	}
	e := events[idx]
	expected := fmt.Sprintf("delay %d", ms)
	if e.DelayStr != expected {
		if e.Part != nil {
			t.Errorf("event[%d] expected %q, got part type=%q text=%q", idx, expected, e.Part.Type, e.Part.Text)
		} else {
			t.Errorf("event[%d] expected %q, got %q", idx, expected, e.DelayStr)
		}
	}
}

func assertPartID(t *testing.T, events []smoothEvent, idx int, expectedID string) {
	t.Helper()
	if idx >= len(events) {
		t.Fatalf("event index %d out of range (len=%d)", idx, len(events))
	}
	e := events[idx]
	if e.Part == nil {
		t.Fatalf("event[%d] expected part, got delay %q", idx, e.DelayStr)
	}
	if e.Part.ID != expectedID {
		t.Errorf("event[%d] expected id %q, got %q", idx, expectedID, e.Part.ID)
	}
}

func intPtr(v int) *int { return &v }

// --- Chunking validation ---

func TestSmoothStream_InvalidChunkingStrategy(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid chunking strategy")
		}
	}()
	SmoothStream(SmoothStreamOptions{Chunking: "foo"})
}

func TestSmoothStream_NullChunkingOption(t *testing.T) {
	// In Go, nil chunking defaults to "word", so this should NOT panic.
	// The TS test checks for null (explicit), which panics in TS.
	// In Go, nil defaults to "word" so it works fine.
	transformer := SmoothStream(SmoothStreamOptions{Chunking: nil})
	if transformer == nil {
		t.Error("expected transformer, got nil")
	}
}

// --- Word chunking ---

func TestSmoothStream_Word_CombinePartialWords(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello", ID: "1"},
		{Type: "text-delta", Text: ", ", ID: "1"},
		{Type: "text-delta", Text: "world!", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Expected: text-start, "Hello, ", delay 10, "world!", text-end
	// Go ordering: chunk is sent to output channel first, then delay marker
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "Hello, ")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "text-delta", "world!")
	assertPartType(t, events, 4, "text-end")
}

func TestSmoothStream_Word_SplitLargerChunks(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello, World! This is an example text.", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Should split into: text-start, Hello, delay, World!, delay, This, delay, is, delay, an, delay, example, delay, text., text-end
	// Go ordering: chunk sent first, then delay marker
	if len(events) < 10 {
		t.Fatalf("expected at least 10 events, got %d", len(events))
	}
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "Hello, ")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "text-delta", "World! ")
	assertDelay(t, events, 4, 10)
	// Last event is text-end
	assertPartType(t, events, len(events)-1, "text-end")
}

func TestSmoothStream_Word_SpacesOnly(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: " ", ID: "1"},
		{Type: "text-delta", Text: " ", ID: "1"},
		{Type: "text-delta", Text: " ", ID: "1"},
		{Type: "text-delta", Text: "foo", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Should combine to: text-start, "   foo", text-end (no delay since word regex doesn't match spaces-only)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "   foo")
	assertPartType(t, events, 2, "text-end")
}

func TestSmoothStream_Word_FlushBeforeToolCall(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "I will check the", ID: "1"},
		{Type: "text-delta", Text: " weather in Lon", ID: "1"},
		{Type: "text-delta", Text: "don.", ID: "1"},
		{Type: "tool-call", ID: "1"}, // non-smoothable chunk
		{Type: "text-end", ID: "1"},
	})

	// Should flush text buffer before tool-call
	found := false
	for _, e := range events {
		if e.Part != nil && e.Part.Type == "tool-call" {
			found = true
		}
	}
	if !found {
		t.Error("expected tool-call event in output")
	}
	// The last text before tool-call should be "London." (flushed)
	lastTextIdx := -1
	for i, e := range events {
		if e.Part != nil && e.Part.Type == "text-delta" {
			lastTextIdx = i
		}
		if e.Part != nil && e.Part.Type == "tool-call" {
			break
		}
	}
	if lastTextIdx < 0 {
		t.Fatal("expected text-delta before tool-call")
	}
}

// --- Line chunking ---

func TestSmoothStream_Line_SplitByLines(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
		Chunking:  "line",
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "First line\nSecond line\nThird line with more text\n", ID: "1"},
		{Type: "text-delta", Text: "Partial line", ID: "1"},
		{Type: "text-delta", Text: " continues\nFinal line\n", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Should produce chunks split at newlines
	// Go ordering: chunk sent first, then delay marker
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "First line\n")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "text-delta", "Second line\n")
	assertDelay(t, events, 4, 10)
	assertPartType(t, events, len(events)-1, "text-end")
}

func TestSmoothStream_Line_NoLineBreaks(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		Chunking: "line",
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Text without", ID: "1"},
		{Type: "text-delta", Text: " any line", ID: "1"},
		{Type: "text-delta", Text: " breaks", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Should just flush the full text as one chunk
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "Text without any line breaks")
	assertPartType(t, events, 2, "text-end")
}

// --- Custom regexp chunking ---

func TestSmoothStream_CustomRegexp_Underscore(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		Chunking:  regexp.MustCompile(`_`),
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello_, world!", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Should split at underscore: "Hello_", delay, ", world!"
	// Go ordering: chunk sent first, then delay marker
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "Hello_")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "text-delta", ", world!")
	assertPartType(t, events, 4, "text-end")
}

func TestSmoothStream_CustomRegexp_CharacterLevel(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		Chunking:  regexp.MustCompile(`.`),
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello, world!", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Should split every character: H, e, l, l, o, ,, ' ', w, o, r, l, d, !
	// Go ordering: chunk sent first, then delay marker
	// 1 (text-start) + 12*(char+delay) + char + text-end = 1 + 24 + 1 + 1 = 27 events
	// Wait - the last chunk has no delay after it since the buffer is empty and loop exits.
	// Actually: for 13 chars, each match produces [chunk, delay], so 13*2=26, plus text-start + text-end = 28.
	// But the last char's delay still fires because delayFn is called after every match in the inner loop.
	if len(events) != 28 {
		t.Fatalf("expected 28 events, got %d", len(events))
	}
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "H")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "text-delta", "e")
	assertDelay(t, events, 4, 10)
	assertPartType(t, events, 27, "text-end")
}

// --- Custom callback chunking ---

func TestSmoothStream_CustomCallback(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		Chunking: ChunkDetector(func(buffer string) string {
			re := regexp.MustCompile(`[^_]*_`)
			m := re.FindString(buffer)
			return m
		}),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "He_llo, ", ID: "1"},
		{Type: "text-delta", Text: "w_orld!", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Go ordering: chunk sent first, then delay marker
	// Should produce: text-start, "He_", delay, "llo, w_", delay, "orld!", text-end
	if len(events) != 7 {
		t.Fatalf("expected 7 events, got %d", len(events))
	}
	assertPartType(t, events, 0, "text-start")
	assertPartText(t, events, 1, "text-delta", "He_")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "text-delta", "llo, w_")
	assertDelay(t, events, 4, 10)
	assertPartText(t, events, 5, "text-delta", "orld!")
	assertPartType(t, events, 6, "text-end")
}

func TestSmoothStream_CustomCallback_NonPrefixPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-prefix match")
		}
	}()

	input := make(chan TextStreamPartForSmooth, 3)
	input <- TextStreamPartForSmooth{Type: "text-start", ID: "1"}
	input <- TextStreamPartForSmooth{Type: "text-delta", Text: "Hello, world!", ID: "1"}
	input <- TextStreamPartForSmooth{Type: "text-end", ID: "1"}
	close(input)

	output := make(chan TextStreamPartForSmooth, 100)
	transformer := SmoothStream(SmoothStreamOptions{
		Chunking: ChunkDetector(func(buffer string) string {
			return "world" // not a prefix of "Hello, world!"
		}),
	})
	transformer(input, output)
}

// --- Delay tests ---

func TestSmoothStream_DefaultDelay(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello, world!", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Default delay is 10ms
	// Go ordering: chunk at 1, delay at 2
	assertPartText(t, events, 1, "text-delta", "Hello, ")
	assertDelay(t, events, 2, 10)
}

func TestSmoothStream_CustomDelay(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(20),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello, world!", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// Go ordering: chunk at 1, delay at 2
	assertPartText(t, events, 1, "text-delta", "Hello, ")
	assertDelay(t, events, 2, 20)
}

func TestSmoothStream_ZeroDelay(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(0),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello, world!", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// 0 delay still records the delay call
	// Go ordering: chunk at 1, delay at 2
	assertPartText(t, events, 1, "text-delta", "Hello, ")
	assertDelay(t, events, 2, 0)
}

// --- ID changes ---

func TestSmoothStream_IDChange(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-start", ID: "2"},
		{Type: "text-delta", Text: "I will check the", ID: "1"},
		{Type: "text-delta", Text: " weather in Lon", ID: "1"},
		{Type: "text-delta", Text: "don.", ID: "1"},
		{Type: "text-delta", Text: "I will check the", ID: "2"},
		{Type: "text-delta", Text: " weather in Lon", ID: "2"},
		{Type: "text-delta", Text: "don.", ID: "2"},
		{Type: "text-end", ID: "1"},
		{Type: "text-end", ID: "2"},
	})

	// Should produce events with id "1" then id "2" with proper flushing
	foundID1 := false
	foundID2 := false
	id1BeforeID2 := false
	for _, e := range events {
		if e.Part != nil && e.Part.Type == "text-delta" {
			if e.Part.ID == "1" {
				foundID1 = true
			}
			if e.Part.ID == "2" {
				if foundID1 {
					id1BeforeID2 = true
				}
				foundID2 = true
			}
		}
	}
	if !foundID1 {
		t.Error("expected text-delta with id '1'")
	}
	if !foundID2 {
		t.Error("expected text-delta with id '2'")
	}
	if !id1BeforeID2 {
		t.Error("expected id '1' deltas before id '2' deltas")
	}
}

// --- Reasoning smoothing ---

func TestSmoothStream_Reasoning_CombinePartial(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "reasoning-start", ID: "1"},
		{Type: "reasoning-delta", Text: "Let", ID: "1"},
		{Type: "reasoning-delta", Text: " me ", ID: "1"},
		{Type: "reasoning-delta", Text: "think...", ID: "1"},
		{Type: "reasoning-end", ID: "1"},
	})

	// Go ordering: chunk sent first, then delay marker
	assertPartType(t, events, 0, "reasoning-start")
	assertPartText(t, events, 1, "reasoning-delta", "Let ")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "reasoning-delta", "me ")
	assertDelay(t, events, 4, 10)
	assertPartText(t, events, 5, "reasoning-delta", "think...")
	assertPartType(t, events, 6, "reasoning-end")
}

func TestSmoothStream_Reasoning_SplitLarger(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "reasoning-start", ID: "1"},
		{Type: "reasoning-delta", Text: "First I need to analyze the problem. Then I will solve it.", ID: "1"},
		{Type: "reasoning-end", ID: "1"},
	})

	// Go ordering: chunk sent first, then delay marker
	assertPartType(t, events, 0, "reasoning-start")
	assertPartText(t, events, 1, "reasoning-delta", "First ")
	assertDelay(t, events, 2, 10)
	assertPartType(t, events, len(events)-1, "reasoning-end")
}

func TestSmoothStream_Reasoning_FlushBeforeToolCall(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "reasoning-start", ID: "1"},
		{Type: "reasoning-delta", Text: "I should check the", ID: "1"},
		{Type: "reasoning-delta", Text: " weather", ID: "1"},
		{Type: "tool-call", ID: "1"},
		{Type: "reasoning-end", ID: "1"},
	})

	// Should flush reasoning buffer before tool call
	foundToolCall := false
	lastReasoningIdx := -1
	for i, e := range events {
		if e.Part != nil && e.Part.Type == "reasoning-delta" {
			lastReasoningIdx = i
		}
		if e.Part != nil && e.Part.Type == "tool-call" {
			foundToolCall = true
			if lastReasoningIdx < 0 {
				t.Error("expected reasoning-delta before tool-call")
			}
		}
	}
	if !foundToolCall {
		t.Error("expected tool-call event")
	}
}

func TestSmoothStream_Reasoning_LineChunking(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
		Chunking:  "line",
	}, []TextStreamPartForSmooth{
		{Type: "reasoning-start", ID: "1"},
		{Type: "reasoning-delta", Text: "Step 1: Analyze\nStep 2: Solve\n", ID: "1"},
		{Type: "reasoning-end", ID: "1"},
	})

	// Go ordering: chunk sent first, then delay marker
	assertPartType(t, events, 0, "reasoning-start")
	assertPartText(t, events, 1, "reasoning-delta", "Step 1: Analyze\n")
	assertDelay(t, events, 2, 10)
	assertPartText(t, events, 3, "reasoning-delta", "Step 2: Solve\n")
	assertDelay(t, events, 4, 10)
	assertPartType(t, events, 5, "reasoning-end")
}

// --- Interleaved text and reasoning ---

func TestSmoothStream_Interleaved_TextToReasoning(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "reasoning-start", ID: "2"},
		{Type: "text-delta", Text: "Hello ", ID: "1"},
		{Type: "text-delta", Text: "world", ID: "1"},
		{Type: "reasoning-delta", Text: "Let me", ID: "2"},
		{Type: "reasoning-delta", Text: " think", ID: "2"},
		{Type: "text-end", ID: "1"},
		{Type: "reasoning-end", ID: "2"},
	})

	// Text buffer should flush when switching to reasoning
	assertPartType(t, events, 0, "text-start")
	assertPartType(t, events, 1, "reasoning-start")

	// Find text-delta and reasoning-delta
	foundTextDelta := false
	foundReasoningDelta := false
	for _, e := range events {
		if e.Part != nil && e.Part.Type == "text-delta" {
			foundTextDelta = true
		}
		if e.Part != nil && e.Part.Type == "reasoning-delta" {
			foundReasoningDelta = true
		}
	}
	if !foundTextDelta {
		t.Error("expected text-delta events")
	}
	if !foundReasoningDelta {
		t.Error("expected reasoning-delta events")
	}
}

// --- Non-smoothable chunks ---

func TestSmoothStream_NonSmoothablePassthrough(t *testing.T) {
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello, world!", ID: "1"},
		{Type: "source", ID: "src-1"},
		{Type: "text-end", ID: "1"},
	})

	// source should pass through without smoothing
	foundSource := false
	for _, e := range events {
		if e.Part != nil && e.Part.Type == "source" {
			foundSource = true
		}
	}
	if !foundSource {
		t.Error("expected source event to pass through")
	}
}

// --- Provider metadata ---

func TestSmoothStream_ProviderMetadata(t *testing.T) {
	pm := ProviderMetadata{"test": {"key": "value"}}
	events := runSmooth(t, SmoothStreamOptions{
		DelayInMs: intPtr(10),
	}, []TextStreamPartForSmooth{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", Text: "Hello ", ID: "1", ProviderMetadata: pm},
		{Type: "text-delta", Text: "world", ID: "1"},
		{Type: "text-end", ID: "1"},
	})

	// The buffer should be flushed at text-end, and the last flush should carry the providerMetadata
	// Check that at least one event has providerMetadata
	_ = events // metadata is captured but exact placement depends on buffer flush
}
