package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// newTestCmd runs `.test.ts` files against a running brainkit
// server. The server-side `test.run` handler walks <dir> for files
// matching --pattern, executes them via the in-process test
// framework (Hardhat-style — test(), describe(), expect(), deploy(),
// sendTo(), sleep()), and returns aggregated results which the CLI
// formats as a ✓ / ✗ tree plus a summary line.
func newTestCmd() *cobra.Command {
	var (
		endpoint string
		pattern  string
		skipAI   bool
	)
	c := &cobra.Command{
		Use:   "test [dir]",
		Short: "Run .test.ts files against a running brainkit server",
		Long: `Test discovers .test.ts files under [dir] (default: current
directory) and runs each one inside the running brainkit runtime.
Tests use the Hardhat-style API:

  test("name", async () => { ... });
  describe("group", () => { ... });
  expect(value).toBe(expected);
  await deploy("pkg.ts", source);
  await sendTo("pkg", "topic", payload);
  await sleep(100);

Filter with --pattern (default "*.test.ts"). Add --skip-ai to skip
tests whose names start with "AI:" — handy when running without
OPENAI_API_KEY available.`,
		Args: cobra.RangeArgs(0, 1),
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

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)

			body, _ := json.Marshal(map[string]any{
				"dir":     absDir,
				"pattern": pattern,
				"skipAI":  skipAI,
			})
			reply, err := client.call(ctx, "test.run", body)
			if err != nil {
				return err
			}

			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}

			var resp struct {
				Results json.RawMessage `json:"results"`
			}
			if err := json.Unmarshal(reply, &resp); err != nil {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			var run testRunResult
			if err := json.Unmarshal(resp.Results, &run); err != nil {
				return writeJSONPretty(cmd.OutOrStdout(), resp.Results)
			}
			formatTestResults(cmd, &run)
			if run.Failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	c.Flags().StringVar(&pattern, "pattern", "*.test.ts", "test file glob pattern")
	c.Flags().BoolVar(&skipAI, "skip-ai", false, "skip tests whose names start with 'AI:'")
	return c
}

type testResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Duration int64  `json:"duration"`
	Error    string `json:"error,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
}

type testSuiteResult struct {
	File     string       `json:"file"`
	Tests    []testResult `json:"tests"`
	Passed   int          `json:"passed"`
	Failed   int          `json:"failed"`
	Skipped  int          `json:"skipped"`
	Duration int64        `json:"duration"`
}

type testRunResult struct {
	Suites   []testSuiteResult `json:"suites"`
	Total    int               `json:"total"`
	Passed   int               `json:"passed"`
	Failed   int               `json:"failed"`
	Skipped  int               `json:"skipped"`
	Duration int64             `json:"duration"`
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

func formatTestResults(cmd *cobra.Command, r *testRunResult) {
	for _, suite := range r.Suites {
		cmd.Printf("\n  %s\n", suite.File)
		for _, t := range suite.Tests {
			dur := time.Duration(t.Duration).Round(time.Millisecond)
			switch {
			case t.Skipped:
				cmd.Printf("    %s- %s (skipped)%s\n", colorYellow, t.Name, colorReset)
			case t.Passed:
				cmd.Printf("    %s✓%s %s %s(%s)%s\n", colorGreen, colorReset, t.Name, colorGray, dur, colorReset)
			default:
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
}
