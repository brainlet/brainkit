package graymatter

import (
	"fmt"
	"strings"
)

// Stringify stringifies front matter and appends it to the provided content string.
func Stringify(file string, data any, opts ...Options) (string, error) {
	if data == nil && len(opts) == 0 {
		return file, nil
	}

	parsed, err := Parse(file, opts...)
	if err != nil {
		return "", err
	}

	return StringifyFile(parsed, data, opts...)
}

// StringifyFile stringifies a parsed file object back into front-matter plus content.
func StringifyFile(file File, data any, opts ...Options) (string, error) {
	resolved := resolveOptions(opts...)
	content := file.Content

	if data == nil {
		if file.Data != nil {
			data = file.Data
		} else if resolved.Data != nil {
			data = resolved.Data
		} else {
			return content, nil
		}
	}

	language := file.Language
	if strings.TrimSpace(language) == "" {
		language = resolved.Language
	}
	if strings.TrimSpace(language) == "" {
		language = "yaml"
	}

	engine, err := getEngine(language, resolved)
	if err != nil {
		return "", err
	}
	if err := ensureStringifyEngine(engine, language); err != nil {
		return "", err
	}

	merged := map[string]any{}
	for key, value := range toMap(file.Data) {
		merged[key] = value
	}
	for key, value := range toMap(data) {
		merged[key] = value
	}

	matter, err := engine.Stringify(merged)
	if err != nil {
		return "", err
	}
	matter = strings.TrimSpace(matter)

	open, close := normalizedFence(resolved)
	var buf strings.Builder
	if matter != "{}" && matter != "" {
		buf.WriteString(ensureTrailingNewline(open))
		buf.WriteString(ensureTrailingNewline(matter))
		buf.WriteString(ensureTrailingNewline(close))
	}

	if strings.TrimSpace(file.Excerpt) != "" {
		trimmed := strings.TrimSpace(file.Excerpt)
		if !strings.Contains(content, trimmed) {
			buf.WriteString(ensureTrailingNewline(file.Excerpt))
			buf.WriteString(ensureTrailingNewline(close))
		}
	}

	buf.WriteString(ensureTrailingNewline(content))
	return buf.String(), nil
}

func ensureStringifyEngine(engine Engine, language string) error {
	switch e := engine.(type) {
	case EngineFunc:
		return fmt.Errorf("expected %q.stringify to be a function", language)
	case EngineWithStringify:
		if e.StringifyFunc == nil {
			return fmt.Errorf("expected %q.stringify to be a function", language)
		}
	case *EngineWithStringify:
		if e == nil || e.StringifyFunc == nil {
			return fmt.Errorf("expected %q.stringify to be a function", language)
		}
	}

	if _, err := engine.Stringify(map[string]any{}); err != nil && strings.Contains(err.Error(), "not supported") {
		return fmt.Errorf("expected %q.stringify to be a function", language)
	}
	return nil
}

func ensureTrailingNewline(str string) string {
	if strings.HasSuffix(str, "\n") {
		return str
	}
	return str + "\n"
}
