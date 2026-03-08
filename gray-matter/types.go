package graymatter

import "strings"

// File represents the result of parsing front-matter from input.
// Matches GrayMatterFile<I> interface from TypeScript.
type File struct {
	// Data is the parsed front-matter as a map of key-value pairs.
	Data map[string]any `json:"data"`
	// Content is the input string with front-matter removed.
	Content string `json:"content"`
	// Excerpt is an optional excerpt extracted from the content.
	Excerpt string `json:"excerpt,omitempty"`
	// Orig is the original input (string or []byte).
	Orig any `json:"orig"`
	// Language is the front-matter language that was parsed (e.g., "yaml").
	Language string `json:"language"`
	// Matter is the raw, un-parsed front-matter string.
	Matter string `json:"matter"`
	// Path is the filepath of the source file.
	Path string `json:"path,omitempty"`
	// IsEmpty is true when the front-matter is empty (whitespace/comments only).
	IsEmpty bool `json:"isEmpty"`
	// Empty is the original front-matter string when front-matter is empty.
	Empty string `json:"empty,omitempty"`
}

// Stringify converts the file's data back to a string in the given language,
// wrapping it in delimiters and prepending it to the content.
func (f *File) Stringify(lang string) string {
	if f == nil {
		return ""
	}

	if strings.TrimSpace(lang) == "" {
		out, err := StringifyFile(*f, nil)
		if err != nil {
			return ""
		}
		return out
	}

	out, err := StringifyFile(*f, nil, Options{Language: lang})
	if err != nil {
		return ""
	}
	return out
}

// Options holds configuration for parsing and stringifying front-matter.
// Matches GrayMatterOption<I, O> interface from TypeScript.
type Options struct {
	// Parser is an optional custom parser function.
	Parser any `json:"parser,omitempty"`
	// Eval enables evaluation of JavaScript front-matter.
	Eval bool `json:"eval,omitempty"`
	// Excerpt enables excerpt extraction. Can be true/false or a custom function.
	Excerpt any `json:"excerpt,omitempty"`
	// ExcerptSeparator defines the separator for extracting excerpts.
	ExcerptSeparator string `json:"excerpt_separator,omitempty"`
	// Engines is a map of language names to parser/stringifier engines.
	Engines map[string]Engine `json:"engines,omitempty"`
	// Language specifies the front-matter language to use (default: "yaml").
	Language string `json:"language,omitempty"`
	// Delimiters specifies the front-matter delimiters.
	// Can be a string (same open/close) or a 2-element slice [open, close].
	Delimiters any `json:"delimiters,omitempty"`
}

// Engine defines the interface for parsing and stringifying front-matter.
// Engines can be functions or objects with Parse/Stringify methods.
type Engine interface {
	// Parse converts a front-matter string to a map of key-value pairs.
	Parse(input string) (map[string]any, error)
	// Stringify converts a map of key-value pairs to a front-matter string.
	Stringify(data map[string]any) (string, error)
}

// EngineFunc is a function adapter that implements Engine for parse-only engines.
type EngineFunc func(input string) (map[string]any, error)

// Parse implements the Engine interface.
func (f EngineFunc) Parse(input string) (map[string]any, error) {
	return f(input)
}

// Stringify implements the Engine interface - returns error for parse-only engines.
func (f EngineFunc) Stringify(data map[string]any) (string, error) {
	return "", nil
}

// EngineWithStringify is an engine that has both Parse and Stringify methods.
type EngineWithStringify struct {
	ParseFunc     func(input string) (map[string]any, error)
	StringifyFunc func(data map[string]any) (string, error)
}

// Parse implements the Engine interface.
func (e EngineWithStringify) Parse(input string) (map[string]any, error) {
	return e.ParseFunc(input)
}

// Stringify implements the Engine interface.
func (e EngineWithStringify) Stringify(data map[string]any) (string, error) {
	return e.StringifyFunc(data)
}
