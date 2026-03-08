// Ported from: packages/provider-utils/src/parse-json-event-stream.ts
package providerutils

import (
	"bufio"
	"io"
	"strings"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	// Event is the event type (optional).
	Event string
	// Data is the event data.
	Data string
	// ID is the event ID (optional).
	ID string
	// Retry is the retry value (optional).
	Retry string
}

// ParseSSEStream reads a Server-Sent Events stream and sends events to the returned channel.
func ParseSSEStream(reader io.Reader) <-chan SSEEvent {
	ch := make(chan SSEEvent)
	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(reader)
		var event SSEEvent
		var dataLines []string

		for scanner.Scan() {
			line := scanner.Text()

			// Empty line means end of event
			if line == "" {
				if len(dataLines) > 0 {
					event.Data = strings.Join(dataLines, "\n")
					ch <- event
				}
				event = SSEEvent{}
				dataLines = nil
				continue
			}

			if strings.HasPrefix(line, ":") {
				// Comment, skip
				continue
			}

			field := line
			value := ""
			if idx := strings.Index(line, ":"); idx >= 0 {
				field = line[:idx]
				value = line[idx+1:]
				if strings.HasPrefix(value, " ") {
					value = value[1:]
				}
			}

			switch field {
			case "event":
				event.Event = value
			case "data":
				dataLines = append(dataLines, value)
			case "id":
				event.ID = value
			case "retry":
				event.Retry = value
			}
		}

		// Flush remaining event
		if len(dataLines) > 0 {
			event.Data = strings.Join(dataLines, "\n")
			ch <- event
		}
	}()
	return ch
}

// ParseJsonEventStream parses a JSON event stream, returning a channel of ParseResult[T].
// It reads SSE events from the reader, skipping [DONE] events, and parses each event's
// data as JSON validated against the provided schema.
func ParseJsonEventStream[T any](reader io.Reader, schema *Schema[T]) <-chan ParseResult[T] {
	ch := make(chan ParseResult[T])
	go func() {
		defer close(ch)
		events := ParseSSEStream(reader)
		for event := range events {
			// ignore the 'DONE' event that e.g. OpenAI sends
			if event.Data == "[DONE]" {
				continue
			}
			result := SafeParseJSON(event.Data, schema)
			ch <- result
		}
	}()
	return ch
}
