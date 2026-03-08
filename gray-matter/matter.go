package graymatter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func Parse(input string, opts ...Options) (File, error) {
	resolved := resolveOptions(opts...)
	delims := NormalizeDelimiters(resolved.Delimiters)
	open := delims[0]
	close := delims[1]

	file := File{
		Data:     map[string]any{},
		Content:  input,
		Excerpt:  "",
		Orig:     input,
		Language: "",
		IsEmpty:  false,
		Empty:    "",
	}

	if input == "" {
		return file, nil
	}

	if !strings.HasPrefix(input, open) {
		excerptFile(&file, resolved)
		return file, nil
	}

	if len(input) > len(open) && input[len(open)] == open[len(open)-1] {
		return file, nil
	}

	firstEOL := strings.IndexByte(input, '\n')
	if firstEOL == -1 {
		return file, nil
	}

	firstLine := strings.TrimRight(input[:firstEOL], "\r")
	langRaw := ""
	if len(firstLine) > len(open) {
		langRaw = strings.TrimSpace(firstLine[len(open):])
	}

	// Only use resolved.Language if explicitly set by user
	if len(opts) > 0 && opts[0].Language != "" {
		file.Language = resolved.Language
	} else if langRaw != "" {
		file.Language = langRaw
	} else {
		file.Language = "yaml"
	}

	matterStart := firstEOL + 1
	closeStart, contentStart, found := findClosingDelimiterLine(input, close, matterStart)
	if !found {
		return file, nil
	}

	file.Matter = input[matterStart:closeStart]

	// Check if matter is empty (whitespace/comments only)
	// TS does: file.matter.replace(/^\s*#[^\n]+/gm, '').trim()
	isEmpty, emptyStr := isEmptyMatter(file.Matter, input)
	if isEmpty {
		file.IsEmpty = true
		file.Empty = emptyStr
		file.Data = map[string]any{}
	} else {
		parsed, err := parseMatterByLanguage(file.Language, file.Matter, resolved)
		if err != nil {
			return File{}, err
		}
		file.Data = parsed
	}

	// Set content - strip leading newline from content
	if contentStart <= len(input) {
		content := input[contentStart:]
		// Strip leading newlines
		for len(content) > 0 && content[0] == '\n' {
			content = content[1:]
		}
		for len(content) > 0 && content[0] == '\r' {
			content = content[1:]
		}
		file.Content = content
	}

	// Extract excerpt
	excerptFile(&file, resolved)

	return file, nil
}

// isEmptyMatter checks if the matter is empty (whitespace/comments only)
// Returns (isEmpty, emptyString) where emptyString is the original input if empty
func isEmptyMatter(matter, input string) (bool, string) {
	// Remove comment lines and check if anything remains
	re := regexp.MustCompile(`(?m)^\s*#[^\n]*\n?`)
	cleaned := re.ReplaceAllString(matter, "")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return true, input
	}
	return false, ""
}

// excerptFile extracts an excerpt from file.Content based on options
func excerptFile(file *File, opts Options) {
	// Check if excerpt is disabled
	if opts.Excerpt == false {
		return
	}

	// Get separator - check data first, then options
	sep := opts.ExcerptSeparator
	if sep == "" {
		sep = "\n"
	}

	if sep == "" {
		return
	}

	// Find the separator in content and extract excerpt
	idx := strings.Index(file.Content, sep)
	if idx != -1 {
		file.Excerpt = file.Content[:idx]
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
	if in.Language != "" {
		resolved.Language = in.Language
	}
	if in.Delimiters != nil {
		resolved.Delimiters = in.Delimiters
	}
	if in.Engines != nil {
		resolved.Engines = make(map[string]Engine, len(in.Engines))
		for key, engine := range in.Engines {
			resolved.Engines[key] = engine
		}
	}

	return resolved
}

func findClosingDelimiterLine(input, close string, from int) (closeStart int, contentStart int, found bool) {
	pos := from
	for pos <= len(input) {
		nextEOLRel := strings.IndexByte(input[pos:], '\n')
		if nextEOLRel == -1 {
			line := strings.TrimRight(input[pos:], "\r")
			if line == close {
				return pos, len(input), true
			}
			return 0, 0, false
		}

		nextEOL := pos + nextEOLRel
		line := strings.TrimRight(input[pos:nextEOL], "\r")
		if line == close {
			return pos, nextEOL + 1, true
		}

		pos = nextEOL + 1
	}

	return 0, 0, false
}

func parseMatterByLanguage(language, matter string, opts Options) (map[string]any, error) {
	block := strings.TrimSpace(matter)
	if block == "" {
		return map[string]any{}, nil
	}

	lang := strings.ToLower(strings.TrimSpace(language))
	if lang == "" {
		lang = "yaml"
	}

	if engine, ok := opts.Engines[lang]; ok && engine != nil {
		return engine.Parse(matter)
	}

	switch lang {
	case "yaml", "yml":
		var out map[string]any
		if err := yaml.Unmarshal([]byte(matter), &out); err != nil {
			return nil, err
		}
		if out == nil {
			return map[string]any{}, nil
		}
		return out, nil
	case "json":
		var out map[string]any
		if err := json.Unmarshal([]byte(block), &out); err != nil {
			return nil, err
		}
		if out == nil {
			return map[string]any{}, nil
		}
		return out, nil
	default:
		return nil, fmt.Errorf("graymatter: no parser engine for language %q", language)
	}
}
