package graymatter

// DefaultOptions returns an Options struct with sensible defaults applied.
// The defaults are:
//   - Language: "yaml"
//   - Delimiters: [2]string{"---", "---"}
//   - Engines: empty map (ready for custom engines)
//   - ExcerptSeparator: "\n"
func DefaultOptions() Options {
	return Options{
		Language:         "yaml",
		Delimiters:       [2]string{"---", "---"},
		Engines:          make(map[string]Engine),
		ExcerptSeparator: "\n",
	}
}

// NormalizeDelimiters converts delimiters to a [2]string if needed.
// If delimiters is already a [2]string, returns it as-is.
// If delimiters is a string, returns [2]string{str, str} (same delimiter for open and close).
// Returns nil if delimiters is nil or not a recognized type.
func NormalizeDelimiters(delimiters any) [2]string {
	if delimiters == nil {
		return [2]string{"---", "---"}
	}

	switch d := delimiters.(type) {
	case [2]string:
		return d
	case []string:
		if len(d) >= 2 {
			return [2]string{d[0], d[1]}
		}
		if len(d) == 1 {
			return [2]string{d[0], d[0]}
		}
		return [2]string{"---", "---"}
	case string:
		return [2]string{d, d}
	default:
		return [2]string{"---", "---"}
	}
}
