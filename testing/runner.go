package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
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

// RunResult is the aggregate outcome of running all test files.
type RunResult struct {
	Suites   []SuiteResult `json:"suites"`
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration"`
}

// TestRunnerConfig configures a TestRunner.
type TestRunnerConfig struct {
	TestDir string        // directory with *.test.ts files
	Pattern string        // glob pattern (default: "*.test.ts")
	Timeout time.Duration // per-file timeout (default: 60s)
	SkipAI     bool          // skip tests whose names start with "AI:" (need API keys)
	ExpectJSON bool          // fixture mode: compare output() against expect.json
}

// Runtime is the minimal interface the TestRunner needs from a Kernel.
// Avoids import cycle between kit/testing and kit.
type Runtime interface {
	// EvalTS evaluates TypeScript code and returns the result string.
	EvalTS(ctx context.Context, source, code string) (string, error)
	// Deploy deploys .ts code into a Compartment.
	Deploy(ctx context.Context, source, code string) error
	// Teardown removes a deployment.
	Teardown(ctx context.Context, source string) error
}

// TestRunner discovers and runs .test.ts files against a Runtime.
type TestRunner struct {
	rt     Runtime
	config TestRunnerConfig
}

// NewTestRunner creates a TestRunner.
func NewTestRunner(rt Runtime, cfg TestRunnerConfig) *TestRunner {
	if cfg.Pattern == "" {
		cfg.Pattern = "*.test.ts"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &TestRunner{rt: rt, config: cfg}
}

// Run discovers test files, bundles each, deploys, executes __runTests(), collects results.
func (r *TestRunner) Run(ctx context.Context) (*RunResult, error) {
	files, err := r.discoverTestFiles()
	if err != nil {
		return nil, fmt.Errorf("test runner: discover files: %w", err)
	}

	result := &RunResult{}
	start := time.Now()

	for _, file := range files {
		suiteCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		suite := r.runFile(suiteCtx, file)
		cancel()

		result.Suites = append(result.Suites, suite)
		result.Total += suite.Passed + suite.Failed + suite.Skipped
		result.Passed += suite.Passed
		result.Failed += suite.Failed
		result.Skipped += suite.Skipped
	}

	result.Duration = time.Since(start)
	return result, nil
}

// RunCode runs inline test code (not from a file). Used for programmatic test execution.
func (r *TestRunner) RunCode(ctx context.Context, name, code string) (*SuiteResult, error) {
	return r.executeTestCode(ctx, name, code)
}

func (r *TestRunner) discoverTestFiles() ([]string, error) {
	if r.config.TestDir == "" {
		return nil, fmt.Errorf("TestDir not configured")
	}

	pattern := filepath.Join(r.config.TestDir, r.config.Pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Also check subdirectories
	subPattern := filepath.Join(r.config.TestDir, "**", r.config.Pattern)
	subMatches, _ := filepath.Glob(subPattern)
	seen := make(map[string]bool)
	var all []string
	for _, m := range append(matches, subMatches...) {
		if !seen[m] {
			seen[m] = true
			all = append(all, m)
		}
	}

	return all, nil
}

func (r *TestRunner) runFile(ctx context.Context, filePath string) SuiteResult {
	if r.config.ExpectJSON {
		return r.runExpectJSON(ctx, filePath)
	}

	start := time.Now()
	suite := SuiteResult{File: filePath}

	// Bundle the test file with esbuild — resolves relative imports,
	// strips TypeScript, marks brainkit modules as external (endowments).
	code, err := bundleTestFile(filePath)
	if err != nil {
		suite.Tests = append(suite.Tests, TestResult{
			Name: "bundle", Error: err.Error(), Duration: time.Since(start),
		})
		suite.Failed = 1
		suite.Duration = time.Since(start)
		return suite
	}

	result, err := r.executeTestCode(ctx, filePath, code)
	if err != nil {
		suite.Tests = append(suite.Tests, TestResult{
			Name: "execution", Error: err.Error(), Duration: time.Since(start),
		})
		suite.Failed = 1
		suite.Duration = time.Since(start)
		return suite
	}

	return *result
}

// runExpectJSON runs a fixture in expect.json mode:
// 1. Read index.ts from the fixture directory
// 2. Deploy and evaluate — capture output()
// 3. Compare against expect.json
func (r *TestRunner) runExpectJSON(ctx context.Context, filePath string) SuiteResult {
	start := time.Now()
	dir := filepath.Dir(filePath)
	name := filepath.Base(dir)
	suite := SuiteResult{File: name}

	// Read the .ts source
	code, err := readFileBytes(filePath)
	if err != nil {
		suite.Tests = append(suite.Tests, TestResult{Name: name, Error: err.Error()})
		suite.Failed = 1
		suite.Duration = time.Since(start)
		return suite
	}

	// Read expect.json
	expectPath := filepath.Join(dir, "expect.json")
	expectData, err := readFileBytes(expectPath)
	if err != nil {
		suite.Tests = append(suite.Tests, TestResult{Name: name, Error: "missing expect.json: " + err.Error()})
		suite.Failed = 1
		suite.Duration = time.Since(start)
		return suite
	}

	// Deploy and run, capture output()
	source := "__fixture_" + name + ".ts"
	if deployErr := r.rt.Deploy(ctx, source, string(code)); deployErr != nil {
		suite.Tests = append(suite.Tests, TestResult{Name: name, Error: deployErr.Error()})
		suite.Failed = 1
		suite.Duration = time.Since(start)
		return suite
	}
	defer r.rt.Teardown(ctx, source)

	// Get the output value
	output, err := r.rt.EvalTS(ctx, "__get_output.ts", `return JSON.stringify(globalThis.__module_result || null);`)
	if err != nil {
		suite.Tests = append(suite.Tests, TestResult{Name: name, Error: err.Error()})
		suite.Failed = 1
		suite.Duration = time.Since(start)
		return suite
	}

	// Compare output against expect.json
	expectedJSON, _ := json.Marshal(json.RawMessage(expectData))
	outputJSON, _ := json.Marshal(json.RawMessage(output))
	passed := string(expectedJSON) == string(outputJSON)

	tr := TestResult{Name: name, Passed: passed, Duration: time.Since(start)}
	if !passed {
		tr.Error = fmt.Sprintf("output mismatch: expected %s, got %s", string(expectData), output)
	}
	suite.Tests = append(suite.Tests, tr)
	if passed {
		suite.Passed = 1
	} else {
		suite.Failed = 1
	}
	suite.Duration = time.Since(start)
	return suite
}

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// bundleTestFile uses esbuild to bundle a .test.ts file with its relative imports.
// brainkit modules (test, kit, ai, agent, compiler) are external — provided by endowments.
func bundleTestFile(filePath string) (string, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{filePath},
		Bundle:      true,
		Format:      api.FormatIIFE,
		Platform:    api.PlatformBrowser,
		External:    []string{"test", "kit", "ai", "agent", "compiler"},
		Write:       false,
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
		},
		TreeShaking: api.TreeShakingTrue,
		Target:      api.ESNext,
	})

	if len(result.Errors) > 0 {
		msg := result.Errors[0]
		loc := ""
		if msg.Location != nil {
			loc = fmt.Sprintf(" at %s:%d:%d", msg.Location.File, msg.Location.Line, msg.Location.Column)
		}
		return "", fmt.Errorf("bundle test %s: %s%s", filePath, msg.Text, loc)
	}

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("bundle test %s: no output", filePath)
	}

	return string(result.OutputFiles[0].Contents), nil
}

func (r *TestRunner) executeTestCode(ctx context.Context, name, code string) (*SuiteResult, error) {
	start := time.Now()
	suite := &SuiteResult{File: name}

	// Strip ES imports from test code (same as kit.Deploy does for .ts)
	// The "test" module exports come from Compartment endowments
	cleanCode := code
	if strings.HasSuffix(name, ".ts") {
		// Remove import lines — test module symbols are globals in the Compartment
		lines := strings.Split(cleanCode, "\n")
		var filtered []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import ") && strings.Contains(trimmed, "from") {
				continue // strip imports
			}
			filtered = append(filtered, line)
		}
		cleanCode = strings.Join(filtered, "\n")
	}

	// Deploy the test code so it registers test() calls
	testSource := "__test_" + filepath.Base(name)
	if err := r.rt.Deploy(ctx, testSource, cleanCode); err != nil {
		return nil, fmt.Errorf("deploy test %s: %w", name, err)
	}
	defer r.rt.Teardown(ctx, testSource)

	// Execute __runTests() to run all registered tests
	resultJSON, err := r.rt.EvalTS(ctx, "__run_tests.ts", `return await globalThis.__runTests();`)
	if err != nil {
		return nil, fmt.Errorf("run tests %s: %w", name, err)
	}

	// Parse results
	var testResults []struct {
		Name     string `json:"name"`
		Passed   bool   `json:"passed"`
		Error    string `json:"error,omitempty"`
		Duration int    `json:"duration"` // ms
	}
	if err := json.Unmarshal([]byte(resultJSON), &testResults); err != nil {
		return nil, fmt.Errorf("parse results %s: %w", name, err)
	}

	for _, tr := range testResults {
		skipped := r.config.SkipAI && strings.HasPrefix(tr.Name, "AI:")
		result := TestResult{
			Name:     tr.Name,
			Passed:   tr.Passed && !skipped,
			Skipped:  skipped,
			Error:    tr.Error,
			Duration: time.Duration(tr.Duration) * time.Millisecond,
		}
		suite.Tests = append(suite.Tests, result)
		if skipped {
			suite.Skipped++
		} else if tr.Passed {
			suite.Passed++
		} else {
			suite.Failed++
		}
	}

	suite.Duration = time.Since(start)
	return suite, nil
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
	Total         int              `json:"total"`
	Passed        int              `json:"passed"`
	Failed        int              `json:"failed"`
	PassRate      float64          `json:"passRate"`
	TotalDuration time.Duration    `json:"totalDuration"`
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
