package cmd

import (
	"fmt"
	"os"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func newEvalCmd() *cobra.Command {
	var evalFile string
	var evalMode string
	var evalSource string

	c := &cobra.Command{
		Use:   "eval [code]",
		Short: "Evaluate TypeScript code",
		Long: `Evaluate TypeScript code.

Modes (--mode):
  script  (default) — deploy Code as a temp .ts, read globalThis.__module_result
  ts                — evaluate Code directly via kernel.EvalTS (no deploy)
  module            — evaluate Code as an ES module (supports imports)

Use -f to read from a file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var code string
			if evalFile != "" {
				data, err := os.ReadFile(evalFile)
				if err != nil {
					return err
				}
				code = string(data)
			} else if len(args) > 0 {
				code = args[0]
			} else {
				return fmt.Errorf("provide code as argument or use -f <file>")
			}
			return connectAndPublish(cmd, sdk.KitEvalMsg{Source: evalSource, Code: code, Mode: evalMode},
				func(resp *sdk.KitEvalResp) {
					cmd.Println(resp.Result)
				},
			)
		},
	}
	c.Flags().StringVarP(&evalFile, "file", "f", "", "read code from file")
	c.Flags().StringVar(&evalMode, "mode", "", "eval mode: script (default), ts, or module")
	c.Flags().StringVar(&evalSource, "source", "", "optional source name (.ts extension auto-selects ts mode)")
	return c
}
