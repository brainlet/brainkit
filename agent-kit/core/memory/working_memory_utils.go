// Ported from: packages/core/src/memory/working-memory-utils.ts
package memory

import "strings"

const (
	WorkingMemoryStartTag = "<working_memory>"
	WorkingMemoryEndTag   = "</working_memory>"
)

// ExtractWorkingMemoryTags extracts all working memory tag contents from text
// using indexOf-based parsing. This avoids ReDoS vulnerability that exists
// with regex-based approaches.
// Returns a slice of full matches (including tags) or nil if no matches.
func ExtractWorkingMemoryTags(text string) []string {
	var results []string
	pos := 0

	for pos < len(text) {
		start := strings.Index(text[pos:], WorkingMemoryStartTag)
		if start == -1 {
			break
		}
		start += pos // adjust to absolute index

		contentStart := start + len(WorkingMemoryStartTag)
		end := strings.Index(text[contentStart:], WorkingMemoryEndTag)
		if end == -1 {
			break
		}
		end += contentStart // adjust to absolute index

		results = append(results, text[start:end+len(WorkingMemoryEndTag)])
		pos = end + len(WorkingMemoryEndTag)
	}

	if len(results) == 0 {
		return nil
	}
	return results
}

// RemoveWorkingMemoryTags removes all working memory tags and their contents
// from text. Uses indexOf-based parsing to avoid ReDoS vulnerability.
func RemoveWorkingMemoryTags(text string) string {
	var result strings.Builder
	pos := 0

	for pos < len(text) {
		start := strings.Index(text[pos:], WorkingMemoryStartTag)
		if start == -1 {
			result.WriteString(text[pos:])
			break
		}
		start += pos // adjust to absolute index

		result.WriteString(text[pos:start])

		contentStart := start + len(WorkingMemoryStartTag)
		end := strings.Index(text[contentStart:], WorkingMemoryEndTag)
		if end == -1 {
			// No closing tag found, keep the rest as-is
			result.WriteString(text[start:])
			break
		}
		end += contentStart // adjust to absolute index

		pos = end + len(WorkingMemoryEndTag)
	}

	return result.String()
}

// ExtractWorkingMemoryContent extracts the content of the first working memory
// tag (without the tags themselves). Uses indexOf-based parsing to avoid ReDoS
// vulnerability.
// Returns the content between the tags, or empty string if no valid tag pair found.
func ExtractWorkingMemoryContent(text string) string {
	start := strings.Index(text, WorkingMemoryStartTag)
	if start == -1 {
		return ""
	}

	contentStart := start + len(WorkingMemoryStartTag)
	end := strings.Index(text[contentStart:], WorkingMemoryEndTag)
	if end == -1 {
		return ""
	}

	return text[contentStart : contentStart+end]
}
