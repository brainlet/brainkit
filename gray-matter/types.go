package graymatter

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// File mirrors the JS gray-matter return shape as closely as practical in Go.
type File struct {
	Data     any       `json:"data"`
	Content  string    `json:"content"`
	Excerpt  string    `json:"excerpt,omitempty"`
	Orig     any       `json:"orig"`
	Language string    `json:"language"`
	Matter   string    `json:"matter"`
	Path     string    `json:"path,omitempty"`
	IsEmpty  bool      `json:"isEmpty"`
	Empty    string    `json:"empty,omitempty"`
	Sections []Section `json:"sections,omitempty"`
}

// Section mirrors the objects produced by section-matter.
type Section struct {
	Key     string `json:"key"`
	Data    string `json:"data"`
	Content string `json:"content"`
}

// JSFunction wraps a JavaScript function parsed by the built-in javascript engine.
type JSFunction struct {
	runtime *goja.Runtime
	value   goja.Value
}

func (f JSFunction) String() string {
	if f.value == nil {
		return ""
	}
	return f.value.String()
}

// Call invokes the wrapped JavaScript function and exports the result back to Go values.
func (f JSFunction) Call(args ...any) (any, error) {
	if f.runtime == nil || f.value == nil {
		return nil, fmt.Errorf("graymatter: javascript function is not initialized")
	}

	fn, ok := goja.AssertFunction(f.value)
	if !ok {
		return nil, fmt.Errorf("graymatter: value is not callable")
	}

	values := make([]goja.Value, len(args))
	for i, arg := range args {
		values[i] = f.runtime.ToValue(arg)
	}

	result, err := fn(goja.Undefined(), values...)
	if err != nil {
		return nil, err
	}
	return exportJSValue(f.runtime, result), nil
}

// DataMap returns the parsed front-matter as a map when the parsed value is object-like.
func (f File) DataMap() map[string]any {
	return toMap(f.Data)
}

// Stringify mirrors the JS file.stringify helper.
func (f *File) Stringify(data any, opts ...Options) (string, error) {
	if f == nil {
		return "", nil
	}

	if len(opts) > 0 {
		resolved := resolveOptions(opts...)
		if resolved.Language != "" {
			f.Language = resolved.Language
		}
	}

	return StringifyFile(*f, data, opts...)
}

// Options mirrors gray-matter's options, including deprecated aliases still used by the JS package.
type Options struct {
	Parser           any               `json:"parser,omitempty"`
	Eval             bool              `json:"eval,omitempty"`
	Excerpt          any               `json:"excerpt,omitempty"`
	ExcerptSeparator string            `json:"excerpt_separator,omitempty"`
	Engines          map[string]Engine `json:"engines,omitempty"`
	Parsers          map[string]Engine `json:"parsers,omitempty"`
	Language         string            `json:"language,omitempty"`
	Lang             string            `json:"lang,omitempty"`
	Delimiters       any               `json:"delimiters,omitempty"`
	Delims           any               `json:"delims,omitempty"`
	Sections         bool              `json:"sections,omitempty"`
	Section          any               `json:"section,omitempty"`
	SectionDelimiter string            `json:"section_delimiter,omitempty"`
	Data             any               `json:"data,omitempty"`
}

// Engine mirrors gray-matter engines: parse is required, stringify is optional.
type Engine interface {
	Parse(input string) (any, error)
	Stringify(data any) (string, error)
}

// EngineFunc adapts a parse-only function into an Engine.
type EngineFunc func(input string) (any, error)

func (f EngineFunc) Parse(input string) (any, error) {
	return f(input)
}

func (f EngineFunc) Stringify(data any) (string, error) {
	return "", fmt.Errorf("stringify not implemented")
}

// EngineWithStringify adapts parse/stringify functions into an Engine.
type EngineWithStringify struct {
	ParseFunc     func(input string) (any, error)
	StringifyFunc func(data any) (string, error)
}

func (e EngineWithStringify) Parse(input string) (any, error) {
	if e.ParseFunc == nil {
		return nil, fmt.Errorf("parse not implemented")
	}
	return e.ParseFunc(input)
}

func (e EngineWithStringify) Stringify(data any) (string, error) {
	if e.StringifyFunc == nil {
		return "", fmt.Errorf("stringify not implemented")
	}
	return e.StringifyFunc(data)
}

func normalizeLanguageName(language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	if lang == "" {
		return "yaml"
	}
	return lang
}
