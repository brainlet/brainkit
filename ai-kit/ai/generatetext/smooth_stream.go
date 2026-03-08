// Ported from: packages/ai/src/generate-text/smooth-stream.ts
package generatetext

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ChunkDetector detects the first chunk in a buffer.
// Returns the first detected chunk, or empty string if no chunk was detected.
type ChunkDetector func(buffer string) string

// SmoothStreamOptions contains options for the smooth stream transformer.
type SmoothStreamOptions struct {
	// DelayInMs is the delay in milliseconds between each chunk. Defaults to 10ms.
	// Set to 0 to skip the delay.
	DelayInMs *int

	// Chunking controls how the text is chunked for streaming.
	// Can be "word", "line", a *regexp.Regexp, or a ChunkDetector function.
	// Defaults to "word".
	Chunking interface{} // "word" | "line" | *regexp.Regexp | ChunkDetector

	// DelayFunc is an internal override for the delay function (for testing).
	DelayFunc func(ms int)
}

var chunkingRegexps = map[string]*regexp.Regexp{
	"word": regexp.MustCompile(`\S+\s+`),
	"line": regexp.MustCompile(`\n+`),
}

// TextStreamPartForSmooth represents a text stream part used with smooth stream.
type TextStreamPartForSmooth struct {
	Type             string
	Text             string
	ID               string
	ProviderMetadata ProviderMetadata
}

// SmoothStream creates a transformer that smooths text and reasoning streaming output.
// It returns a function that transforms a channel of TextStreamPartForSmooth.
func SmoothStream(opts SmoothStreamOptions) func(input <-chan TextStreamPartForSmooth, output chan<- TextStreamPartForSmooth) {
	delayMs := 10
	if opts.DelayInMs != nil {
		delayMs = *opts.DelayInMs
	}

	delayFn := func(ms int) {
		if ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}
	if opts.DelayFunc != nil {
		delayFn = opts.DelayFunc
	}

	var detectChunk ChunkDetector

	chunking := opts.Chunking
	if chunking == nil {
		chunking = "word"
	}

	switch c := chunking.(type) {
	case ChunkDetector:
		detectChunk = func(buffer string) string {
			match := c(buffer)
			if match == "" {
				return ""
			}
			if !strings.HasPrefix(buffer, match) {
				panic(fmt.Sprintf("chunking function must return a match that is a prefix of the buffer. Received: %q expected to start with %q", match, buffer))
			}
			return match
		}
	case string:
		re, ok := chunkingRegexps[c]
		if !ok {
			panic(fmt.Sprintf("chunking must be \"word\", \"line\", a *regexp.Regexp, or a ChunkDetector function. Received: %s", c))
		}
		detectChunk = makeRegexpDetector(re)
	case *regexp.Regexp:
		detectChunk = makeRegexpDetector(c)
	default:
		panic("invalid chunking type")
	}

	return func(input <-chan TextStreamPartForSmooth, output chan<- TextStreamPartForSmooth) {
		defer close(output)

		buffer := ""
		id := ""
		var partType string
		var providerMetadata ProviderMetadata

		flushBuffer := func() {
			if buffer != "" && partType != "" {
				output <- TextStreamPartForSmooth{
					Type:             partType,
					Text:             buffer,
					ID:               id,
					ProviderMetadata: providerMetadata,
				}
				buffer = ""
				providerMetadata = nil
			}
		}

		for chunk := range input {
			// Handle non-smoothable chunks: flush buffer and pass through
			if chunk.Type != "text-delta" && chunk.Type != "reasoning-delta" {
				flushBuffer()
				output <- chunk
				continue
			}

			// Flush buffer when type or id changes
			if (chunk.Type != partType || chunk.ID != id) && buffer != "" {
				flushBuffer()
			}

			buffer += chunk.Text
			id = chunk.ID
			partType = chunk.Type

			if chunk.ProviderMetadata != nil {
				providerMetadata = chunk.ProviderMetadata
			}

			for {
				match := detectChunk(buffer)
				if match == "" {
					break
				}
				output <- TextStreamPartForSmooth{
					Type: partType,
					Text: match,
					ID:   id,
				}
				buffer = buffer[len(match):]
				delayFn(delayMs)
			}
		}

		// Flush remaining buffer
		flushBuffer()
	}
}

func makeRegexpDetector(re *regexp.Regexp) ChunkDetector {
	return func(buffer string) string {
		loc := re.FindStringIndex(buffer)
		if loc == nil {
			return ""
		}
		return buffer[:loc[1]]
	}
}
