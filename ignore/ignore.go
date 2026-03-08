// Ported from: node-ignore/index.js
package ignore

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/dlclark/regexp2"
)

// RuleSource mirrors node-ignore's ability to inherit rules from another ignore
// instance, while remaining open to compatible external implementations.
type RuleSource interface {
	IgnoreRules() []*Rule
}

// Rule mirrors the public ignore rule shape returned in TestResult.Rule.
type Rule struct {
	Pattern  string `json:"pattern"`
	Mark     string `json:"mark,omitempty"`
	Negative bool   `json:"negative"`

	body        string
	ignoreCase  bool
	regexPrefix string
	regex       *regexp2.Regexp
	checkRegex  *regexp2.Regexp
}

// PatternParams mirrors node-ignore's {pattern, mark?} object form.
type PatternParams struct {
	Pattern string `json:"pattern"`
	Mark    string `json:"mark,omitempty"`
}

// TestResult mirrors node-ignore's result object.
type TestResult struct {
	Ignored   bool  `json:"ignored"`
	Unignored bool  `json:"unignored"`
	Rule      *Rule `json:"rule,omitempty"`
}

// Options mirrors node-ignore's constructor options. The ignore-case flags are
// pointers so Go can preserve the upstream default of true while still allowing
// callers to explicitly set false.
type Options struct {
	Ignorecase         *bool `json:"ignorecase,omitempty"`
	IgnoreCase         *bool `json:"ignoreCase,omitempty"`
	AllowRelativePaths bool  `json:"allowRelativePaths,omitempty"`
}

// Bool is a helper for constructing pointer-backed option fields.
func Bool(v bool) *bool { return &v }

type resolvedOptions struct {
	ignoreCase         bool
	allowRelativePaths bool
}

type pathMode struct {
	convert       func(string) string
	isNotRelative func(string) bool
}

var (
	regexInvalidWindowsPathChars = regexp.MustCompile(`["<>|\x00-\x1F]+`)
	regexWindowsAbsolute         = regexp.MustCompile(`^[a-z]:/`)
	forceWindowsPathMode         bool
)

var posixPathMode = pathMode{
	convert: func(path string) string {
		return path
	},
	isNotRelative: isNotRelativePath,
}

var windowsPathMode = pathMode{
	convert: func(path string) string {
		if strings.HasPrefix(path, `\\?\`) || regexInvalidWindowsPathChars.MatchString(path) {
			return path
		}

		return strings.ReplaceAll(path, `\`, `/`)
	},
	isNotRelative: func(path string) bool {
		return regexWindowsAbsolute.MatchString(strings.ToLower(path)) || isNotRelativePath(path)
	},
}

func currentPathMode() pathMode {
	if forceWindowsPathMode || runtime.GOOS == "windows" {
		return windowsPathMode
	}

	return posixPathMode
}

// New creates a new ignore manager.
func New(opts ...Options) *Ignore {
	resolved := resolveOptions(opts...)

	return &Ignore{
		rules:           newRuleManager(resolved.ignoreCase),
		strictPathCheck: !resolved.allowRelativePaths,
		pathMode:        currentPathMode(),
		ignoreCache:     map[string]TestResult{},
		testCache:       map[string]TestResult{},
	}
}

// Ignore is the core matcher/manager ported from node-ignore.
type Ignore struct {
	rules           *ruleManager
	strictPathCheck bool
	pathMode        pathMode
	ignoreCache     map[string]TestResult
	testCache       map[string]TestResult
}

// IgnoreRules returns the currently registered rules.
func (ig *Ignore) IgnoreRules() []*Rule {
	if ig == nil || ig.rules == nil {
		return nil
	}

	out := make([]*Rule, len(ig.rules.rules))
	copy(out, ig.rules.rules)
	return out
}

// Add mirrors node-ignore's add() API.
func (ig *Ignore) Add(pattern any) *Ignore {
	if ig == nil {
		return ig
	}

	if ig.rules.add(pattern) {
		ig.initCache()
	}

	return ig
}

// AddPattern is the legacy alias preserved by node-ignore.
func (ig *Ignore) AddPattern(pattern any) *Ignore {
	return ig.Add(pattern)
}

// Ignores returns whether the path should be ignored. It panics on invalid
// input to preserve node-ignore's throwing API.
func (ig *Ignore) Ignores(path any) bool {
	return ig.mustTest(path, ig.ignoreCache, false, nil).Ignored
}

// CreateFilter returns a filter predicate suitable for slice filtering.
func (ig *Ignore) CreateFilter() func(any) bool {
	return func(path any) bool {
		return !ig.Ignores(path)
	}
}

// Filter filters a string or slice/array of strings.
func (ig *Ignore) Filter(paths any) []string {
	items := makeArray(paths)
	filter := ig.CreateFilter()
	result := make([]string, 0, len(items))
	for _, item := range items {
		if filter(item) {
			result = append(result, item.(string))
		}
	}
	return result
}

// Test returns the full ignored/unignored state for a path.
func (ig *Ignore) Test(path any) TestResult {
	return ig.mustTest(path, ig.testCache, true, nil)
}

// CheckIgnore mirrors git check-ignore behavior more closely for directory
// paths that end with a trailing slash.
func (ig *Ignore) CheckIgnore(path any) TestResult {
	original, ok := path.(string)
	if !ok || !strings.HasSuffix(original, "/") {
		return ig.Test(path)
	}

	converted, err := ig.normalizePath(path)
	if err != nil {
		panic(err)
	}

	slices := splitPath(converted)
	if len(slices) > 0 {
		slices = slices[:len(slices)-1]
	}

	if len(slices) > 0 {
		parent := ig.t(strings.Join(slices, "/")+"/", ig.testCache, true, slices)
		if parent.Ignored {
			return parent
		}
	}

	return ig.rules.test(converted, false, modeCheckIgnore)
}

func (ig *Ignore) initCache() {
	ig.ignoreCache = map[string]TestResult{}
	ig.testCache = map[string]TestResult{}
}

func (ig *Ignore) mustTest(originalPath any, cache map[string]TestResult, checkUnignored bool, slices []string) TestResult {
	path, err := ig.normalizePath(originalPath)
	if err != nil {
		panic(err)
	}

	return ig.t(path, cache, checkUnignored, slices)
}

func (ig *Ignore) normalizePath(originalPath any) (string, error) {
	path, ok := originalPath.(string)
	if !ok {
		return "", newTypeError(fmt.Sprintf("path must be a string, but got `%s`", jsLiteral(originalPath)))
	}

	if path == "" {
		return "", newTypeError("path must not be empty")
	}

	converted := ig.pathMode.convert(path)
	if ig.pathMode.isNotRelative(converted) && ig.strictPathCheck {
		return "", newRangeError(fmt.Sprintf("path should be a `path.relative()`d string, but got %q", path))
	}

	return converted, nil
}

func (ig *Ignore) t(path string, cache map[string]TestResult, checkUnignored bool, slices []string) TestResult {
	if cached, ok := cache[path]; ok {
		return cached
	}

	if slices == nil {
		slices = splitPath(path)
	}

	if len(slices) > 0 {
		slices = slices[:len(slices)-1]
	}

	if len(slices) == 0 {
		result := ig.rules.test(path, checkUnignored, modeIgnore)
		cache[path] = result
		return result
	}

	parent := ig.t(strings.Join(slices, "/")+"/", cache, checkUnignored, slices)
	if parent.Ignored {
		cache[path] = parent
		return parent
	}

	result := ig.rules.test(path, checkUnignored, modeIgnore)
	cache[path] = result
	return result
}

// IsPathValid reports whether the path would be accepted by ignore's matcher.
func IsPathValid(path any) bool {
	stringPath, ok := path.(string)
	if !ok || stringPath == "" {
		return false
	}

	mode := currentPathMode()
	return !mode.isNotRelative(mode.convert(stringPath))
}

func resolveOptions(opts ...Options) resolvedOptions {
	resolved := resolvedOptions{
		ignoreCase: true,
	}
	if len(opts) == 0 {
		return resolved
	}

	opt := opts[0]
	if opt.Ignorecase != nil {
		resolved.ignoreCase = *opt.Ignorecase
	}
	if opt.IgnoreCase != nil {
		resolved.ignoreCase = *opt.IgnoreCase
	}
	resolved.allowRelativePaths = opt.AllowRelativePaths
	return resolved
}

func makeArray(subject any) []any {
	if subject == nil {
		return []any{nil}
	}

	value := reflect.ValueOf(subject)
	switch value.Kind() {
	case reflect.Array, reflect.Slice:
		result := make([]any, value.Len())
		for i := 0; i < value.Len(); i++ {
			result[i] = value.Index(i).Interface()
		}
		return result
	default:
		return []any{subject}
	}
}

func splitPattern(pattern string) []string {
	lines := strings.Split(strings.ReplaceAll(pattern, "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func isNotRelativePath(path string) bool {
	matched, _ := reInvalidPath.MatchString(path)
	return matched
}

func jsLiteral(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

type typeError struct{ message string }

func (e *typeError) Error() string { return e.message }

func newTypeError(message string) error {
	return &typeError{message: message}
}

type rangeError struct{ message string }

func (e *rangeError) Error() string { return e.message }

func newRangeError(message string) error {
	return &rangeError{message: message}
}
