package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	var testPattern string
	var testSkipAI bool

	c := &cobra.Command{
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
			msg := sdk.TestRunMsg{Dir: absDir, Pattern: testPattern, SkipAI: testSkipAI}
			return connectAndPublish(cmd, msg, func(resp *sdk.TestRunResp) {
				var result runResult
				if err := json.Unmarshal(resp.Results, &result); err != nil {
					cmd.PrintErrln("failed to parse results:", err)
					return
				}
				formatTestResults(cmd, &result)
			})
		},
	}
	c.Flags().StringVar(&testPattern, "pattern", "*.test.ts", "test file glob pattern")
	c.Flags().BoolVar(&testSkipAI, "skip-ai", false, "skip tests whose names start with AI:")
	return c
}

type testResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Duration int64  `json:"duration"`
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

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

func formatTestResults(cmd *cobra.Command, r *runResult) {
	for _, suite := range r.Suites {
		cmd.Printf("\n  %s\n", suite.File)
		for _, t := range suite.Tests {
			dur := time.Duration(t.Duration).Round(time.Millisecond)
			if t.Skipped {
				cmd.Printf("    %s- %s (skipped)%s\n", colorYellow, t.Name, colorReset)
			} else if t.Passed {
				cmd.Printf("    %s✓%s %s %s(%s)%s\n", colorGreen, colorReset, t.Name, colorGray, dur, colorReset)
			} else {
				cmd.Printf("    %s✗ %s%s %s(%s)%s\n", colorRed, t.Name, colorReset, colorGray, dur, colorReset)
				if t.Error != "" {
					cmd.Printf("      %s%s%s\n", colorRed, t.Error, colorReset)
				}
			}
		}
	}
	cmd.Println()
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
	cmd.Println(summary)
	if r.Failed > 0 {
		os.Exit(1)
	}
}
