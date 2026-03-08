package graymatter

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	emptyMatterCommentRE = regexp.MustCompile(`(?m)^\s*#[^\n]+`)

	cacheMu sync.RWMutex
	cache   = map[string]File{}
)

// Parse extracts and parses front-matter from the input string.
func Parse(input string, opts ...Options) (File, error) {
	if input == "" {
		return File{
			Data:    map[string]any{},
			Content: "",
			Excerpt: "",
			Orig:    input,
		}, nil
	}

	normalized := stripBOM(input)
	if len(opts) == 0 {
		if cached, ok := getCachedFile(normalized); ok {
			return cached, nil
		}
	}

	file := File{
		Data:     map[string]any{},
		Content:  normalized,
		Excerpt:  "",
		Orig:     []byte(input),
		Language: "",
		Matter:   "",
		IsEmpty:  false,
	}

	parsed, err := parseMatter(file, resolveOptions(opts...))
	if err != nil {
		return File{}, err
	}

	if len(opts) == 0 {
		setCachedFile(normalized, parsed)
	}

	return parsed, nil
}

func parseMatter(file File, opts Options) (File, error) {
	open, close := normalizedFence(opts)
	str := file.Content

	if opts.Language != "" {
		file.Language = opts.Language
	}

	openLen := len(open)
	if !strings.HasPrefix(str, open) {
		if err := excerptFile(&file, opts); err != nil {
			return File{}, err
		}
		return file, nil
	}

	if len(str) > openLen && str[openLen] == open[len(open)-1] {
		return file, nil
	}

	str = str[openLen:]
	language := Language(str, opts)
	if language.Name != "" {
		file.Language = language.Name
		str = str[len(language.Raw):]
	}

	closeIndex := strings.Index(str, "\n"+close)
	if closeIndex == -1 {
		closeIndex = len(str)
	}

	file.Matter = str[:closeIndex]

	block := strings.TrimSpace(emptyMatterCommentRE.ReplaceAllString(file.Matter, ""))
	if block == "" {
		file.IsEmpty = true
		file.Empty = file.Content
		file.Data = map[string]any{}
	} else {
		parsed, err := parseMatterByLanguage(file.Language, file.Matter, opts)
		if err != nil {
			return File{}, err
		}
		if parsed == nil {
			parsed = map[string]any{}
		}
		file.Data = parsed
	}

	if closeIndex == len(str) {
		file.Content = ""
	} else {
		file.Content = str[closeIndex+len("\n"+close):]
		if strings.HasPrefix(file.Content, "\r") {
			file.Content = file.Content[1:]
		}
		if strings.HasPrefix(file.Content, "\n") {
			file.Content = file.Content[1:]
		}
	}

	if err := excerptFile(&file, opts); err != nil {
		return File{}, err
	}

	if opts.Sections || isCallable(opts.Section) {
		parseSections(&file, opts)
	}

	return file, nil
}

func parseMatterByLanguage(language, matter string, opts Options) (any, error) {
	if opts.Parser != nil {
		return callParser(opts.Parser, matter, opts)
	}

	engine, err := getEngine(language, opts)
	if err != nil {
		return nil, err
	}
	if engine == nil {
		return nil, fmt.Errorf("gray-matter engine %q is not registered", language)
	}

	parsed, err := engine.Parse(matter)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return map[string]any{}, nil
	}
	return parsed, nil
}

func getEngine(language string, opts Options) (Engine, error) {
	name := language
	if strings.TrimSpace(name) == "" {
		name = opts.Language
	}

	engine, ok := opts.Engines[name]
	if !ok {
		engine, ok = opts.Engines[engineAlias(name)]
	}
	if !ok || engine == nil {
		return nil, fmt.Errorf("gray-matter engine %q is not registered", name)
	}
	return engine, nil
}

func engineAlias(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "js", "javascript":
		return "javascript"
	case "coffee", "coffeescript", "cson":
		return "coffee"
	case "yaml", "yml":
		return "yaml"
	default:
		return name
	}
}

func normalizedFence(opts Options) (string, string) {
	delims := NormalizeDelimiters(opts.Delimiters)
	return delims[0], delims[1]
}

// ClearCache mirrors gray-matter.clearCache.
func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = map[string]File{}
}

func getCachedFile(key string) (File, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	file, ok := cache[key]
	return file, ok
}

func setCachedFile(key string, file File) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache[key] = file
}
