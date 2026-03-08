package graymatter

import (
	"regexp"
	"strings"
)

var sectionSplitRE = regexp.MustCompile(`\r?\n`)

func parseSections(file *File, opts Options) {
	if file == nil {
		return
	}

	delimiter := opts.SectionDelimiter
	if delimiter == "" {
		delimiter = "---"
	}

	lines := sectionSplitRE.Split(file.Content, -1)
	var sections []Section
	section := Section{}
	content := []string{}
	stack := []string{}

	initSections := func(value string) {
		file.Content = value
		sections = []Section{}
		content = []string{}
	}

	closeSection := func(value string) {
		if len(stack) == 0 {
			return
		}

		section.Key = getSectionKey(stack[0], delimiter)
		section.Content = value
		if isCallable(opts.Section) {
			_ = callSection(opts.Section, &section, sections)
		}
		sections = append(sections, section)
		section = Section{}
		content = []string{}
		stack = []string{}
	}

	initialized := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		stackLen := len(stack)

		if isSectionDelimiter(trimmed, delimiter) {
			if len(trimmed) == 3 && i != 0 {
				if stackLen == 0 || stackLen == 2 {
					content = append(content, line)
					continue
				}
				stack = append(stack, trimmed)
				section.Data = strings.Join(content, "\n")
				content = []string{}
				continue
			}

			if !initialized {
				initSections(strings.Join(content, "\n"))
				initialized = true
			}

			if stackLen == 2 {
				closeSection(strings.Join(content, "\n"))
			}

			stack = append(stack, trimmed)
			continue
		}

		content = append(content, line)
	}

	if !initialized {
		initSections(strings.Join(content, "\n"))
	} else {
		closeSection(strings.Join(content, "\n"))
	}

	file.Sections = sections
}

func isSectionDelimiter(line, delimiter string) bool {
	if !strings.HasPrefix(line, delimiter) {
		return false
	}

	pos := len(delimiter) + 1
	if pos < len(line) && line[pos] == delimiter[len(delimiter)-1] {
		return false
	}
	return true
}

func getSectionKey(value, delimiter string) string {
	if value == "" || !strings.HasPrefix(value, delimiter) {
		return ""
	}
	return strings.TrimSpace(value[len(delimiter):])
}
