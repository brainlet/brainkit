// Ported from: node-ignore/index.js
package ignore

import (
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"
)

const (
	modeIgnore      = "regex"
	modeCheckIgnore = "checkRegex"
)

var (
	reBlankLine                 = regexp2.MustCompile(`^\s+$`, regexp2.None)
	reInvalidTrailingBackslash  = regexp2.MustCompile(`(?:[^\\]|^)\\$`, regexp2.None)
	reReplaceLeadingEscapedBang = regexp2.MustCompile(`^\\!`, regexp2.None)
	reReplaceLeadingEscapedHash = regexp2.MustCompile(`^\\#`, regexp2.None)
	reInvalidPath               = regexp2.MustCompile(`^\.{0,2}\/|^\.{1,2}$`, regexp2.None)
	reRegexpRange               = regexp2.MustCompile(`([0-z])-([0-z])`, regexp2.None)
	reReplaceTrailingWildcard   = regexp2.MustCompile(`(^|\\\/)?\\\*$`, regexp2.None)
	reTrailingSpaces            = regexp2.MustCompile(`((?:\\\\)*?)(\\?\s+)$`, regexp2.None)
	reEscapedSpace              = regexp2.MustCompile(`(\\+?)\s`, regexp2.None)
	reEscapeMeta                = regexp2.MustCompile(`[\\$.|*+(){^]`, regexp2.None)
	reQuestionMark              = regexp2.MustCompile(`(?!\\)\?`, regexp2.None)
	reLeadingSlash              = regexp2.MustCompile(`^\/`, regexp2.None)
	reSlash                     = regexp2.MustCompile(`/`, regexp2.None)
	reLeadingDoubleStarSlash    = regexp2.MustCompile(`^\^*\\\*\\\*\\\/`, regexp2.None)
	reStarting                  = regexp2.MustCompile(`^(?=[^^])`, regexp2.None)
	reDoubleGlobstar            = regexp2.MustCompile(`\\\/\\\*\\\*(?=\\\/|$)`, regexp2.None)
	reIntermediateWildcard      = regexp2.MustCompile(`(^|[^\\]+)(\\\*)+(?=.+)`, regexp2.None)
	reUnescapeMeta              = regexp2.MustCompile(`\\\\\\(?=[$.|*+(){^])`, regexp2.None)
	reDoubleBackslash           = regexp2.MustCompile(`\\\\`, regexp2.None)
	reRangeNotation             = regexp2.MustCompile(`(\\)?\[([^\]/]*?)(\\*)($|\])`, regexp2.None)
	reEnding                    = regexp2.MustCompile(`(?:[^*])$`, regexp2.None)
	reTestTrailingSlash         = regexp2.MustCompile(`\/$`, regexp2.None)
	reOriginalHasSlashNotAtEnd  = regexp2.MustCompile(`\/(?!$)`, regexp2.None)
)

type ruleManager struct {
	ignoreCase bool
	rules      []*Rule
}

func newRuleManager(ignoreCase bool) *ruleManager {
	return &ruleManager{
		ignoreCase: ignoreCase,
		rules:      []*Rule{},
	}
}

func (rm *ruleManager) add(pattern any) bool {
	added := false
	for _, item := range normalizePatterns(pattern) {
		if rm.addOne(item) {
			added = true
		}
	}
	return added
}

func (rm *ruleManager) addOne(pattern any) bool {
	if source, ok := pattern.(RuleSource); ok && source != nil {
		rm.rules = append(rm.rules, source.IgnoreRules()...)
		return true
	}

	switch value := pattern.(type) {
	case string:
		if checkPattern(value) {
			rm.rules = append(rm.rules, createRule(PatternParams{Pattern: value}, rm.ignoreCase))
			return true
		}
		return false
	case PatternParams:
		if checkPattern(value.Pattern) {
			rm.rules = append(rm.rules, createRule(value, rm.ignoreCase))
			return true
		}
		return false
	case *PatternParams:
		if value != nil && checkPattern(value.Pattern) {
			rm.rules = append(rm.rules, createRule(*value, rm.ignoreCase))
			return true
		}
		return false
	default:
		return false
	}
}

func (rm *ruleManager) test(path string, checkUnignored bool, mode string) TestResult {
	ignored := false
	unignored := false
	var matchedRule *Rule

	for _, rule := range rm.rules {
		negative := rule.Negative
		if (unignored == negative && ignored != unignored) || (negative && !ignored && !unignored && !checkUnignored) {
			continue
		}

		var matched bool
		if mode == modeCheckIgnore {
			matched = rule.checkRegexMatch(path)
		} else {
			matched = rule.regexMatch(path)
		}
		if !matched {
			continue
		}

		ignored = !negative
		unignored = negative
		if negative {
			matchedRule = nil
		} else {
			matchedRule = rule
		}
	}

	result := TestResult{
		Ignored:   ignored,
		Unignored: unignored,
	}
	if matchedRule != nil {
		result.Rule = matchedRule
	}

	return result
}

func normalizePatterns(pattern any) []any {
	switch value := pattern.(type) {
	case string:
		return stringsToAny(splitPattern(value))
	default:
		return makeArray(pattern)
	}
}

func stringsToAny(items []string) []any {
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

func checkPattern(pattern string) bool {
	if pattern == "" {
		return false
	}

	if matched, _ := reBlankLine.MatchString(pattern); matched {
		return false
	}
	if matched, _ := reInvalidTrailingBackslash.MatchString(pattern); matched {
		return false
	}

	return !strings.HasPrefix(pattern, "#")
}

func createRule(pattern PatternParams, ignoreCase bool) *Rule {
	body := pattern.Pattern
	negative := false
	if strings.HasPrefix(body, "!") {
		negative = true
		body = body[1:]
	}

	body = regexp2MustReplace(reReplaceLeadingEscapedBang, body, "!")
	body = regexp2MustReplace(reReplaceLeadingEscapedHash, body, "#")

	return &Rule{
		Pattern:     pattern.Pattern,
		Mark:        pattern.Mark,
		Negative:    negative,
		body:        body,
		ignoreCase:  ignoreCase,
		regexPrefix: makeRegexPrefix(body),
	}
}

func (r *Rule) regexMatch(path string) bool {
	if r.regex == nil {
		r.regex = r.makeRegex(modeIgnore)
	}
	matched, _ := r.regex.MatchString(path)
	return matched
}

func (r *Rule) checkRegexMatch(path string) bool {
	if r.checkRegex == nil {
		r.checkRegex = r.makeRegex(modeCheckIgnore)
	}
	matched, _ := r.checkRegex.MatchString(path)
	return matched
}

func (r *Rule) makeRegex(mode string) *regexp2.Regexp {
	replacer := trailingWildCardReplacerIgnore
	if mode == modeCheckIgnore {
		replacer = trailingWildCardReplacerCheckIgnore
	}

	pattern := regexp2ReplaceAllStringFunc(reReplaceTrailingWildcard, r.regexPrefix, func(match string, groups []string, index int, input string) string {
		prefix := ""
		if len(groups) > 1 {
			prefix = groups[1]
		}
		return replacer(prefix)
	})

	flags := regexp2.None
	if r.ignoreCase {
		flags |= regexp2.IgnoreCase
	}

	re, err := regexp2.Compile(pattern, flags)
	if err != nil {
		pattern = strings.ReplaceAll(pattern, "[]", "(?!)")
		re, err = regexp2.Compile(pattern, flags)
	}
	if err != nil {
		panic(fmt.Sprintf("ignore: failed to compile %q from %q: %v", pattern, r.Pattern, err))
	}
	return re
}

func trailingWildCardReplacerIgnore(prefix string) string {
	if prefix != "" {
		return prefix + `[^/]+(?=$|\/$)`
	}
	return `[^/]*(?=$|\/$)`
}

func trailingWildCardReplacerCheckIgnore(prefix string) string {
	if prefix != "" {
		return prefix + `[^/]*(?=$|\/$)`
	}
	return `[^/]*(?=$|\/$)`
}

func makeRegexPrefix(pattern string) string {
	output := pattern

	if strings.HasPrefix(output, "\uFEFF") {
		output = output[len("\uFEFF"):]
	}

	output = regexp2ReplaceAllStringFunc(reTrailingSpaces, output, func(match string, groups []string, index int, input string) string {
		m1 := groupAt(groups, 1)
		m2 := groupAt(groups, 2)
		if strings.HasPrefix(m2, `\`) {
			return m1 + " "
		}
		return m1
	})

	output = regexp2ReplaceAllStringFunc(reEscapedSpace, output, func(match string, groups []string, index int, input string) string {
		m1 := groupAt(groups, 1)
		return m1[:len(m1)-len(m1)%2] + " "
	})

	output = regexp2ReplaceAllStringFunc(reEscapeMeta, output, func(match string, groups []string, index int, input string) string {
		return `\` + match
	})

	output = regexp2ReplaceAllStringFunc(reQuestionMark, output, func(match string, groups []string, index int, input string) string {
		return `[^/]`
	})

	output = regexp2MustReplace(reLeadingSlash, output, "^")
	output = regexp2MustReplace(reSlash, output, `\/`)
	output = regexp2MustReplace(reLeadingDoubleStarSlash, output, `^(?:.*\/)?`)

	output = regexp2ReplaceAllStringFunc(reStarting, output, func(match string, groups []string, index int, input string) string {
		hasSlash, _ := reOriginalHasSlashNotAtEnd.MatchString(pattern)
		if hasSlash {
			return "^"
		}
		return `(?:^|\/)`
	})

	output = regexp2ReplaceAllStringFunc(reDoubleGlobstar, output, func(match string, groups []string, index int, input string) string {
		if index+len(match) < len(input) {
			return `(?:\/[^\/]+)*`
		}
		return `\/.+`
	})

	output = regexp2ReplaceAllStringFunc(reIntermediateWildcard, output, func(match string, groups []string, index int, input string) string {
		p1 := groupAt(groups, 1)
		p2 := groupAt(groups, 2)
		return p1 + strings.ReplaceAll(p2, `\*`, `[^\/]*`)
	})

	output = regexp2MustReplace(reUnescapeMeta, output, `\`)
	output = regexp2MustReplace(reDoubleBackslash, output, `\`)

	output = regexp2ReplaceAllStringFunc(reRangeNotation, output, func(match string, groups []string, index int, input string) string {
		leadEscape := groupAt(groups, 1)
		rng := groupAt(groups, 2)
		endEscape := groupAt(groups, 3)
		close := groupAt(groups, 4)

		if leadEscape == `\` {
			return `\[` + rng + cleanRangeBackSlash(endEscape) + close
		}

		if close == "]" {
			if len(endEscape)%2 == 0 {
				return "[" + sanitizeRange(rng) + endEscape + "]"
			}
			return "[]"
		}

		return "[]"
	})

	output = regexp2ReplaceAllStringFunc(reEnding, output, func(match string, groups []string, index int, input string) string {
		trailingSlash, _ := reTestTrailingSlash.MatchString(match)
		if trailingSlash {
			return match + "$"
		}
		return match + `(?=$|\/$)`
	})

	return output
}

func sanitizeRange(rng string) string {
	return regexp2ReplaceAllStringFunc(reRegexpRange, rng, func(match string, groups []string, index int, input string) string {
		from := groupAt(groups, 1)
		to := groupAt(groups, 2)
		if from != "" && to != "" && from[0] <= to[0] {
			return match
		}
		return ""
	})
}

func cleanRangeBackSlash(slashes string) string {
	return slashes[:len(slashes)-len(slashes)%2]
}

func regexp2MustReplace(re *regexp2.Regexp, input, replacement string) string {
	output, err := re.Replace(input, replacement, -1, -1)
	if err != nil {
		panic(err)
	}
	return output
}

func regexp2ReplaceAllStringFunc(re *regexp2.Regexp, input string, fn func(match string, groups []string, index int, input string) string) string {
	var builder strings.Builder
	lastIndex := 0

	match, err := re.FindStringMatch(input)
	if err != nil {
		panic(err)
	}

	for match != nil {
		builder.WriteString(input[lastIndex:match.Index])
		groups := match.Groups()
		groupStrings := make([]string, len(groups))
		for i, group := range groups {
			groupStrings[i] = group.String()
		}
		builder.WriteString(fn(match.String(), groupStrings, match.Index, input))
		lastIndex = match.Index + match.Length

		match, err = re.FindNextMatch(match)
		if err != nil {
			panic(err)
		}
	}

	builder.WriteString(input[lastIndex:])
	return builder.String()
}

func groupAt(groups []string, index int) string {
	if index < len(groups) {
		return groups[index]
	}
	return ""
}
