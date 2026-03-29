package testing

import (
	"encoding/json"
	"fmt"
	"time"
)

// TestResult is the outcome of one test case.
type TestResult struct {
	Name     string        `json:"name"`
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	Skipped  bool          `json:"skipped,omitempty"`
}

// SuiteResult is the outcome of running all tests in a file.
type SuiteResult struct {
	File     string        `json:"file"`
	Tests    []TestResult  `json:"tests"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration"`
}

// Summary returns a one-line summary.
func (s SuiteResult) Summary() string {
	return fmt.Sprintf("%s: %d passed, %d failed, %d skipped (%s)",
		s.File, s.Passed, s.Failed, s.Skipped, s.Duration.Round(time.Millisecond))
}

// Evaluator runs a deployed service against a dataset.
type Evaluator struct{}

// EvalCase is one test case in an evaluation dataset.
type EvalCase struct {
	Input  json.RawMessage `json:"input"`
	Expect json.RawMessage `json:"expect"`
}

// EvalResult is the outcome of an evaluation run.
type EvalResult struct {
	Total         int           `json:"total"`
	Passed        int           `json:"passed"`
	Failed        int           `json:"failed"`
	PassRate      float64       `json:"passRate"`
	TotalDuration time.Duration `json:"totalDuration"`
	Cases         []EvalCaseResult `json:"cases"`
}

// EvalCaseResult is the outcome of one evaluation case.
type EvalCaseResult struct {
	Input    json.RawMessage `json:"input"`
	Expected json.RawMessage `json:"expected"`
	Actual   json.RawMessage `json:"actual"`
	Passed   bool            `json:"passed"`
	Error    string          `json:"error,omitempty"`
	Duration time.Duration   `json:"duration"`
}
