package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var (
	testPattern string
	testSkipAI  bool
)

var testCmd = &cobra.Command{
	Use:   "test [dir]",
	Short: "Run .test.ts files against a running brainkit instance",
	Long: `Discovers and runs .test.ts files using the brainkit test framework.
Requires a running instance (brainkit start). Tests use the Hardhat-style
API: test(), describe(), expect(), deploy(), sendTo(), sleep().`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		if _, err := os.Stat(absDir); err != nil {
			return fmt.Errorf("directory not found: %s", absDir)
		}

		msg := messages.TestRunMsg{
			Dir:     absDir,
			Pattern: testPattern,
			SkipAI:  testSkipAI,
		}

		return connectAndPublish(msg, func(resp *messages.TestRunResp) {
			var result runResult
			if err := json.Unmarshal(resp.Results, &result); err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse results: %v\n", err)
				return
			}
			formatTestResults(&result)
		})
	},
}

// --- Result types (match testing.RunResult JSON) ---

type testResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Duration int64  `json:"duration"` // nanoseconds
	Error    string `json:"error,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
}

type suiteResult struct {
	File     string       `json:"file"`
	Tests    []testResult `json:"tests"`
	Passed   int          `json:"passed"`
	Failed   int          `json:"failed"`
	Skipped  int          `json:"skipped"`
	Duration int64        `json:"duration"`
}

type runResult struct {
	Suites   []suiteResult `json:"suites"`
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration int64         `json:"duration"`
}

// --- ANSI colors ---

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

// --- Terminal formatter ---

func formatTestResults(r *runResult) {
	for _, suite := range r.Suites {
		fmt.Printf("\n  %s\n", suite.File)
		for _, t := range suite.Tests {
			dur := time.Duration(t.Duration).Round(time.Millisecond)
			if t.Skipped {
				fmt.Printf("    %s- %s (skipped)%s\n", colorYellow, t.Name, colorReset)
			} else if t.Passed {
				fmt.Printf("    %s✓%s %s %s(%s)%s\n", colorGreen, colorReset, t.Name, colorGray, dur, colorReset)
			} else {
				fmt.Printf("    %s✗ %s%s %s(%s)%s\n", colorRed, t.Name, colorReset, colorGray, dur, colorReset)
				if t.Error != "" {
					fmt.Printf("      %s%s%s\n", colorRed, t.Error, colorReset)
				}
			}
		}
	}

	fmt.Println()
	summary := fmt.Sprintf("  %d tests", r.Total)
	if r.Passed > 0 {
		summary += fmt.Sprintf(", %s%d passed%s", colorGreen, r.Passed, colorReset)
	}
	if r.Failed > 0 {
		summary += fmt.Sprintf(", %s%d failed%s", colorRed, r.Failed, colorReset)
	}
	if r.Skipped > 0 {
		summary += fmt.Sprintf(", %s%d skipped%s", colorYellow, r.Skipped, colorReset)
	}
	totalDur := time.Duration(r.Duration).Round(time.Millisecond)
	summary += fmt.Sprintf(" %s(%s)%s", colorGray, totalDur, colorReset)
	fmt.Println(summary)

	if r.Failed > 0 {
		os.Exit(1)
	}
}

func init() {
	testCmd.Flags().StringVar(&testPattern, "pattern", "*.test.ts", "test file glob pattern")
	testCmd.Flags().BoolVar(&testSkipAI, "skip-ai", false, "skip tests whose names start with AI:")
	rootCmd.AddCommand(testCmd)
}
