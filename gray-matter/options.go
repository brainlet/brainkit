package graymatter

import (
	"strings"

	gmengines "github.com/brainlet/brainkit/gray-matter/engines"
)

// DefaultOptions returns an Options struct with sensible defaults applied.
func DefaultOptions() Options {
	return Options{
		Language:   "yaml",
		Delimiters: [2]string{"---", "---"},
		Engines: map[string]Engine{
			"yaml":       gmengines.YAML{},
			"json":       gmengines.JSON{},
			"javascript": javascriptEngine{},
		},
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

func resolveOptions(opts ...Options) Options {
	resolved := DefaultOptions()
	if len(opts) == 0 {
		return resolved
	}

	in := opts[0]
	resolved.Parser = in.Parser
	resolved.Eval = in.Eval
	resolved.Excerpt = in.Excerpt
	resolved.ExcerptSeparator = in.ExcerptSeparator
	resolved.Sections = in.Sections
	resolved.Section = in.Section
	resolved.SectionDelimiter = in.SectionDelimiter
	resolved.Data = in.Data

	if in.Language != "" {
		resolved.Language = strings.ToLower(in.Language)
	} else if in.Lang != "" {
		resolved.Language = strings.ToLower(in.Lang)
	}

	if in.Delims != nil {
		resolved.Delimiters = in.Delims
	} else if in.Delimiters != nil {
		resolved.Delimiters = in.Delimiters
	}

	if in.Parsers != nil {
		for key, engine := range in.Parsers {
			resolved.Engines[key] = engine
		}
	}

	if in.Engines != nil {
		for key, engine := range in.Engines {
			resolved.Engines[key] = engine
		}
	}

	return resolved
}
