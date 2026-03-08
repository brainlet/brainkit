// Ported from: packages/provider-utils/src/parse-json-event-stream (integration test)
package providerutils

import (
	"strings"
	"testing"
)

func TestParseSSEStream_BasicEvents(t *testing.T) {
	input := "data: {\"id\": 1}\n\ndata: {\"id\": 2}\n\n"
	reader := strings.NewReader(input)
	events := ParseSSEStream(reader)

	var collected []SSEEvent
	for event := range events {
		collected = append(collected, event)
	}

	if len(collected) != 2 {
		t.Fatalf("expected 2 events, got %d", len(collected))
	}
	if collected[0].Data != `{"id": 1}` {
		t.Errorf("expected first event data %q, got %q", `{"id": 1}`, collected[0].Data)
	}
	if collected[1].Data != `{"id": 2}` {
		t.Errorf("expected second event data %q, got %q", `{"id": 2}`, collected[1].Data)
	}
}

func TestParseSSEStream_SkipsDONE(t *testing.T) {
	input := "data: {\"id\": 1}\n\ndata: [DONE]\n\n"
	reader := strings.NewReader(input)

	ch := ParseJsonEventStream[map[string]interface{}](reader, nil)

	var results []ParseResult[map[string]interface{}]
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (DONE should be skipped), got %d", len(results))
	}
	if !results[0].Success {
		t.Fatalf("expected success, got error: %v", results[0].Error)
	}
}

func TestParseSSEStream_Comments(t *testing.T) {
	input := ": this is a comment\ndata: {\"id\": 1}\n\n"
	reader := strings.NewReader(input)
	events := ParseSSEStream(reader)

	var collected []SSEEvent
	for event := range events {
		collected = append(collected, event)
	}

	if len(collected) != 1 {
		t.Fatalf("expected 1 event (comment skipped), got %d", len(collected))
	}
}

func TestParseSSEStream_MultiLineData(t *testing.T) {
	input := "data: line1\ndata: line2\n\n"
	reader := strings.NewReader(input)
	events := ParseSSEStream(reader)

	var collected []SSEEvent
	for event := range events {
		collected = append(collected, event)
	}

	if len(collected) != 1 {
		t.Fatalf("expected 1 event, got %d", len(collected))
	}
	if collected[0].Data != "line1\nline2" {
		t.Errorf("expected multi-line data, got %q", collected[0].Data)
	}
}
