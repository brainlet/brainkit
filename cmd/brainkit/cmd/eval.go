package cmd

import (
	"fmt"
	"os"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

func newEvalCmd() *cobra.Command {
	var evalFile string

	c := &cobra.Command{
		Use:   "eval [code]",
		Short: "Evaluate TypeScript code",
		Long:  "Evaluate TypeScript code using output() to return results. Use -f to read from a file.",
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
			return connectAndPublish(cmd, messages.KitEvalMsg{Code: code},
				func(resp *messages.KitEvalResp) {
					cmd.Println(resp.Result)
				},
			)
		},
	}
	c.Flags().StringVarP(&evalFile, "file", "f", "", "read code from file")
	return c
}
