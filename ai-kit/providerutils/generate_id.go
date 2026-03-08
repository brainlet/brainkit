// Ported from: packages/provider-utils/src/generate-id.ts
package providerutils

import (
	"fmt"
	"math/rand"
	"strings"
)

// IdGenerator is a function that generates an ID string.
type IdGenerator func() string

// CreateIdGeneratorOptions configures CreateIdGenerator.
type CreateIdGeneratorOptions struct {
	// Prefix for the generated ID. Optional.
	Prefix *string
	// Separator between the prefix and the random part. Default: "-".
	Separator *string
	// Size of the random part. Default: 16.
	Size *int
	// Alphabet to use for the random part.
	// Default: "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz".
	Alphabet *string
}

const defaultAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const defaultSeparator = "-"
const defaultSize = 16

// CreateIdGenerator creates an ID generator.
// The total length of the ID is the sum of the prefix, separator, and random part length.
// Not cryptographically secure.
func CreateIdGenerator(opts *CreateIdGeneratorOptions) IdGenerator {
	alphabet := defaultAlphabet
	separator := defaultSeparator
	size := defaultSize
	var prefix *string

	if opts != nil {
		if opts.Alphabet != nil {
			alphabet = *opts.Alphabet
		}
		if opts.Separator != nil {
			separator = *opts.Separator
		}
		if opts.Size != nil {
			size = *opts.Size
		}
		prefix = opts.Prefix
	}

	generator := func() string {
		alphabetLen := len(alphabet)
		chars := make([]byte, size)
		for i := 0; i < size; i++ {
			chars[i] = alphabet[rand.Intn(alphabetLen)]
		}
		return string(chars)
	}

	if prefix == nil {
		return generator
	}

	// Check that the separator is not part of the alphabet
	if strings.Contains(alphabet, separator) {
		panic(fmt.Sprintf(
			"invalid argument: separator: The separator %q must not be part of the alphabet %q.",
			separator, alphabet,
		))
	}

	return func() string {
		return *prefix + separator + generator()
	}
}

// GenerateId generates a 16-character random string to use for IDs.
// Not cryptographically secure.
var GenerateId = CreateIdGenerator(nil)
