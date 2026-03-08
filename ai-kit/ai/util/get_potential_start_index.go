// Ported from: packages/ai/src/util/get-potential-start-index.ts
package util

import "strings"

// GetPotentialStartIndex finds the potential starting index where searchedText
// could begin in text.
//
// This function checks for both complete and partial matches:
//   - If searchedText is found as a complete substring, returns the index of the first occurrence.
//   - If the end of text matches the beginning of searchedText (partial match),
//     returns the index where that partial match starts.
//
// Returns -1 if searchedText is empty or no match is found.
// (The TS version returns null; we use -1 as a sentinel in Go, with a bool return.)
func GetPotentialStartIndex(text, searchedText string) (int, bool) {
	if len(searchedText) == 0 {
		return -1, false
	}

	// Check if the searchedText exists as a direct substring of text.
	directIndex := strings.Index(text, searchedText)
	if directIndex != -1 {
		return directIndex, true
	}

	// Otherwise, look for the largest suffix of "text" that matches
	// a prefix of "searchedText". We go from the end of text inward.
	for i := len(text) - 1; i >= 0; i-- {
		suffix := text[i:]
		if strings.HasPrefix(searchedText, suffix) {
			return i, true
		}
	}

	return -1, false
}
