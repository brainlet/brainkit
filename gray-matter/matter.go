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

	// Matter starts after the opening delimiter line (firstEOL + 1)
	// But TS includes the newline after the opening delimiter in matter
	matterStart := firstEOL + 1
	closeStart, contentStart, found := findClosingDelimiterLine(input, close, matterStart)

	if !found {
		// No closing delimiter found - treat rest of string as matter
		// Matter includes the newline after opening delimiter
		file.Matter = "\n" + input[matterStart:]
	} else {
		// Matter includes the newline after opening delimiter
		matterWithNewline := input[matterStart:closeStart]
		// TS keeps one leading newline but strips trailing newline
		// First add leading newline if not present (for consistency)
		if !strings.HasPrefix(matterWithNewline, "\n") {
			matterWithNewline = "\n" + matterWithNewline
		}
		// Strip only ONE trailing newline (if present) - TS behavior
		file.Matter = strings.TrimSuffix(matterWithNewline, "\n")
	}

	// Check if matter is empty (whitespace/comments only)
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

	// Set content - TS preserves one leading newline when there are multiple
	if found && contentStart <= len(input) {
		content := input[contentStart:]
		// TS preserves ONE leading newline when there are multiple newlines
		// e.g., "\n\ncontent" becomes "\ncontent"
		if len(content) >= 2 && content[0] == '\n' && content[1] == '\n' {
			// Multiple leading newlines - strip to just one
			content = content[1:]
		}
		// Also handle \r\n
		if len(content) >= 2 && content[0] == '\r' && content[1] == '\r' {
			content = content[1:]
		}
		file.Content = content
	} else if !found {
		// No closing delimiter - content is empty
		file.Content = ""
	}

	// Extract excerpt
	excerptFile(&file, resolved)

	return file, nil
}

// isEmptyMatter checks if the matter is empty (whitespace/comments only)
func isEmptyMatter(matter, input string) (bool, string) {
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
	// If explicitly disabled, don't extract
	if opts.Excerpt == false {
		file.Excerpt = ""
		return
	}

	// Determine separator - default is the front-matter delimiter "---"
	sep := opts.ExcerptSeparator
	if sep == "" {
		// No custom separator - use default
		if opts.Excerpt == true {
			// Default separator is the front-matter delimiter "---"
			sep = "---"
		} else {
			file.Excerpt = ""
			return
		}
	}

	// Find the separator in content and extract excerpt (but NOT including the separator)
	idx := strings.Index(file.Content, sep)
	if idx != -1 {
		// Excerpt is content BEFORE the separator (matching TS)
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
