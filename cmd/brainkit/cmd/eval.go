package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// newEvalCmd evaluates TypeScript code against a running server.
// Three modes map to how the runtime executes the source:
//
//	script  (default) — deploy as a temp .ts, read globalThis.__module_result
//	ts                — kernel.EvalTS, no deploy
//	module            — ES module evaluation (imports allowed)
//
// Used for ad-hoc debugging, one-off scripts, or wiring quick checks
// without scaffolding a package.
func newEvalCmd() *cobra.Command {
	var (
		endpoint string
		file     string
		mode     string
		source   string
	)
	c := &cobra.Command{
		Use:   "eval [code]",
		Short: "Evaluate TypeScript code against a running server",
		Long: `Eval runs TypeScript code inside the running brainkit runtime and
prints whatever the code returns.

Modes:
  script  (default)  deploy as a temp .ts, read globalThis.__module_result
  ts                 kernel.EvalTS, no deploy (top-level await supported)
  module             ES module evaluation (supports import statements)

The code can come from an argument, a file (--file / -f), or stdin
when neither is provided.

Examples:
  brainkit eval '1 + 1'
  brainkit eval --mode ts 'return await (await fetch("https://example.com")).text()'
  brainkit eval -f ./scripts/probe.ts --mode module`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := readCodeSource(args, file)
			if err != nil {
				return err
			}
			if strings.TrimSpace(code) == "" {
				return fmt.Errorf("no code supplied — pass an arg, --file, or pipe on stdin")
			}

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			client := newBusClient(endpoint)
			payload, err := json.Marshal(map[string]string{
				"source": source,
				"code":   code,
				"mode":   mode,
			})
			if err != nil {
				return err
			}
			reply, err := client.call(ctx, "kit.eval", payload)
			if err != nil {
				return err
			}
			var resp struct {
				Result string `json:"result"`
				Error  string `json:"error"`
			}
			if err := json.Unmarshal(reply, &resp); err != nil {
				cmd.Println(string(reply))
				return nil
			}
			if resp.Error != "" {
				return fmt.Errorf("%s", resp.Error)
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			cmd.Println(resp.Result)
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	c.Flags().StringVarP(&file, "file", "f", "", "read code from file")
	c.Flags().StringVar(&mode, "mode", "", "eval mode: script (default), ts, or module")
	c.Flags().StringVar(&source, "source", "", "source name (.ts extension auto-selects ts mode)")
	return c
}

// readCodeSource resolves the code input from (in priority order):
// positional arg, --file, or stdin. Returns an empty string when
// none are available.
func readCodeSource(args []string, file string) (string, error) {
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", file, err)
		}
		return string(b), nil
	}
	if len(args) > 0 {
		return args[0], nil
	}
	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		// Non-tty stdin — consume.
		b, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(b), nil
	}
	return "", nil
}
