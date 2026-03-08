// Ported from: packages/core/src/utils/tiktoken.ts
package utils

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// Shared Tiktoken singleton -- lazy init, cached via sync.Once.
//
// Uses sync.Once so the tokenizer is never loaded unless code actually
// needs it. The instance is stored at the package level so it can be
// reused across callers without re-initializing (each init loads the
// full BPE rank table).

var (
	tiktokenOnce sync.Once
	tiktokenEnc  *tiktoken.Tiktoken
	tiktokenErr  error
)

// GetTiktoken returns the shared Tiktoken encoder instance (o200k_base).
// Uses sync.Once so the same instance is reused across callers.
func GetTiktoken() (*tiktoken.Tiktoken, error) {
	tiktokenOnce.Do(func() {
		tiktokenEnc, tiktokenErr = tiktoken.GetEncoding("o200k_base")
	})
	return tiktokenEnc, tiktokenErr
}

// CountTokens is a convenience wrapper that tokenizes the given text
// and returns the token count.
func CountTokens(text string) (int, error) {
	enc, err := GetTiktoken()
	if err != nil {
		return 0, err
	}
	tokens := enc.Encode(text, nil, nil)
	return len(tokens), nil
}
