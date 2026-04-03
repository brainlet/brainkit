package cmd

import (
	"bufio"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var secretsStdin bool

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets",
}

var secretsSetCmd = &cobra.Command{
	Use:   "set <name> [value]",
	Short: "Set a secret",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		var value string
		if secretsStdin {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				value = scanner.Text()
			}
		} else if len(args) > 1 {
			value = args[1]
		} else {
			return fmt.Errorf("provide value as argument or use --stdin")
		}
		return connectAndPublish(
			messages.SecretsSetMsg{Name: name, Value: value},
			func(resp *messages.SecretsSetResp) {
				fmt.Printf("Secret %s set (version %d)\n", name, resp.Version)
			},
		)
	},
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a secret value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.SecretsGetMsg{Name: args[0]},
			func(resp *messages.SecretsGetResp) {
				fmt.Println(resp.Value)
			},
		)
	},
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets (names only, not values)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.SecretsListMsg{},
			func(resp *messages.SecretsListResp) {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tVERSION\tUPDATED")
				for _, s := range resp.Secrets {
					fmt.Fprintf(w, "%s\t%d\t%s\n", s.Name, s.Version, s.UpdatedAt)
				}
				w.Flush()
			},
		)
	},
}

var secretsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.SecretsDeleteMsg{Name: args[0]},
			func(resp *messages.SecretsDeleteResp) {
				fmt.Printf("Secret %s deleted\n", args[0])
			},
		)
	},
}

func init() {
	secretsSetCmd.Flags().BoolVar(&secretsStdin, "stdin", false, "read value from stdin")
	secretsCmd.AddCommand(secretsSetCmd, secretsGetCmd, secretsListCmd, secretsDeleteCmd)
	rootCmd.AddCommand(secretsCmd)
}
