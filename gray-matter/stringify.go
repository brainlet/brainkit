package graymatter

import (
	"fmt"
	"strings"

	gmengines "github.com/brainlet/brainkit/gray-matter/engines"
)

func Stringify(file string, data map[string]any, opts ...Options) (string, error) {
	if data == nil && len(opts) == 0 {
		return file, nil
	}

	parsed, err := Parse(file, opts...)
	if err != nil {
		return "", err
	}

	return StringifyFile(parsed, data, opts...)
}

func StringifyFile(file File, data map[string]any, opts ...Options) (string, error) {
	resolved := resolveOptions(opts...)

	language := strings.TrimSpace(file.Language)
	if language == "" {
		language = strings.TrimSpace(resolved.Language)
	}
	if language == "" {
		language = "yaml"
	}

	engine, err := getStringifyEngine(language, resolved)
	if err != nil {
		return "", err
	}

	merged := make(map[string]any, len(file.Data)+len(data))
	for key, value := range file.Data {
		merged[key] = value
	}
	for key, value := range data {
		merged[key] = value
	}

	matter, err := engine.Stringify(merged)
	if err != nil {
		return "", err
	}
	matter = strings.TrimSpace(matter)

	delims := NormalizeDelimiters(resolved.Delimiters)
	open := delims[0]
	close := delims[1]

	var buf strings.Builder
	if matter != "" && matter != "{}" {
		buf.WriteString(ensureTrailingNewline(open))
		buf.WriteString(ensureTrailingNewline(matter))
		buf.WriteString(ensureTrailingNewline(close))
	}

	content := file.Content
	if strings.TrimSpace(file.Excerpt) != "" {
		excerptTrimmed := strings.TrimSpace(file.Excerpt)
		if !strings.Contains(content, excerptTrimmed) {
			buf.WriteString(ensureTrailingNewline(file.Excerpt))
			buf.WriteString(ensureTrailingNewline(close))
		}
	}

	buf.WriteString(ensureTrailingNewline(content))
	return buf.String(), nil
}

func getStringifyEngine(language string, opts Options) (Engine, error) {
	lang := normalizeLanguage(language)

	if engine, ok := opts.Engines[lang]; ok && engine != nil {
		if err := validateStringifyEngine(engine, lang); err != nil {
			return nil, err
		}
		return engine, nil
	}

	if lang == "yml" {
		lang = "yaml"
	}

	switch lang {
	case "yaml":
		return gmengines.YAML{}, nil
	case "json":
		return gmengines.JSON{}, nil
	default:
		return nil, fmt.Errorf("graymatter: no stringify engine for language %q", language)
	}
}

func validateStringifyEngine(engine Engine, language string) error {
	switch e := engine.(type) {
	case EngineFunc:
		return fmt.Errorf("graymatter: expected %q stringify engine to be available", language)
	case EngineWithStringify:
		if e.StringifyFunc == nil {
			return fmt.Errorf("graymatter: expected %q stringify engine to be available", language)
		}
	case *EngineWithStringify:
		if e == nil || e.StringifyFunc == nil {
			return fmt.Errorf("graymatter: expected %q stringify engine to be available", language)
		}
	}
	return nil
}

func normalizeLanguage(language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	if lang == "" {
		return "yaml"
	}
	return lang
}

func ensureTrailingNewline(str string) string {
	if strings.HasSuffix(str, "\n") {
		return str
	}
	return str + "\n"
}
