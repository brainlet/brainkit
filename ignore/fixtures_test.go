package ignore

import (
	_ "embed"
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
)

//go:embed testdata/upstream_cases.json
var upstreamCasesJSON []byte

type fixturePattern struct {
	Value any
}

func (p *fixturePattern) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, `"`) {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		p.Value = value
		return nil
	}

	var value PatternParams
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.Value = value
	return nil
}

type fixturePatterns struct {
	Values []fixturePattern
	Single bool
}

func (p *fixturePatterns) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, `[`) {
		var values []fixturePattern
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		p.Values = values
		p.Single = false
		return nil
	}

	var value fixturePattern
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.Values = []fixturePattern{value}
	p.Single = true
	return nil
}

func (p fixturePattern) toAddValue() any {
	return p.Value
}

func (p fixturePattern) stringValue() string {
	switch value := p.Value.(type) {
	case string:
		return value
	case PatternParams:
		return value.Pattern
	default:
		return ""
	}
}

type fixtureScopes []string

func (s *fixtureScopes) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "false" || trimmed == "null" || trimmed == "" {
		*s = nil
		return nil
	}

	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*s = fixtureScopes(values)
	return nil
}

func (s fixtureScopes) allows(scope string) bool {
	return len(s) == 0 || slices.Contains([]string(s), scope)
}

type upstreamCase struct {
	Description     string          `json:"description"`
	Patterns        fixturePatterns `json:"patterns"`
	PathsObject     map[string]int  `json:"paths_object"`
	TestOnly        bool            `json:"test_only"`
	SkipTestFixture bool            `json:"skip_test_fixture"`
	Scopes          fixtureScopes   `json:"scopes"`
	Paths           []string        `json:"paths"`
	Expected        []string        `json:"expected"`
}

var (
	loadCasesOnce sync.Once
	allCases      []upstreamCase
	loadCasesErr  error
)

func loadUpstreamCases(t *testing.T) []upstreamCase {
	t.Helper()

	loadCasesOnce.Do(func() {
		loadCasesErr = json.Unmarshal(upstreamCasesJSON, &allCases)
	})
	if loadCasesErr != nil {
		t.Fatalf("failed to load upstream fixture corpus: %v", loadCasesErr)
	}

	return allCases
}

func newIgnoreWithPatterns(patterns []fixturePattern, opts ...Options) *Ignore {
	ig := New(opts...)
	values := make([]any, len(patterns))
	for i, pattern := range patterns {
		values[i] = pattern.toAddValue()
	}
	ig.Add(values)
	return ig
}

func newIgnoreWithFixturePatterns(patterns fixturePatterns, opts ...Options) *Ignore {
	ig := New(opts...)
	if patterns.Single && len(patterns.Values) == 1 {
		ig.Add(patterns.Values[0].toAddValue())
		return ig
	}

	values := make([]any, len(patterns.Values))
	for i, pattern := range patterns.Values {
		values[i] = pattern.toAddValue()
	}
	ig.Add(values)
	return ig
}

func assertSameStrings(t *testing.T, got, want []string) {
	t.Helper()

	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)

	if !slices.Equal(gotCopy, wantCopy) {
		t.Fatalf("unexpected result\n got: %v\nwant: %v", gotCopy, wantCopy)
	}
}

func filterWithPredicate(paths []string, predicate func(any) bool) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if predicate(path) {
			result = append(result, path)
		}
	}
	return result
}

func makeWin32(path string) string {
	return strings.ReplaceAll(path, "/", `\`)
}

func withWindowsPathMode(t *testing.T, fn func()) {
	t.Helper()

	previous := forceWindowsPathMode
	forceWindowsPathMode = true
	defer func() {
		forceWindowsPathMode = previous
	}()

	fn()
}

func assertPanicsWith(t *testing.T, wantSubstring string, fn func()) {
	t.Helper()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic containing %q", wantSubstring)
		}

		message := recovered.(error).Error()
		if !strings.Contains(message, wantSubstring) {
			t.Fatalf("panic = %q, want substring %q", message, wantSubstring)
		}
	}()

	fn()
}
