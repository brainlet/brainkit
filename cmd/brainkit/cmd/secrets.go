package cmd

import (
	"bufio"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

func newSecretsCmd() *cobra.Command {
	var secretsStdin bool

	secretsCmd := &cobra.Command{Use: "secrets", Short: "Manage secrets"}

	setCmd := &cobra.Command{
		Use: "set <name> [value]", Short: "Set a secret", Args: cobra.RangeArgs(1, 2),
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
			return connectAndPublish(cmd, messages.SecretsSetMsg{Name: name, Value: value},
				func(resp *messages.SecretsSetResp) { cmd.Printf("Secret %s set (version %d)\n", name, resp.Version) },
			)
		},
	}
	setCmd.Flags().BoolVar(&secretsStdin, "stdin", false, "read value from stdin")

	getCmd := &cobra.Command{
		Use: "get <name>", Short: "Get a secret value", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, messages.SecretsGetMsg{Name: args[0]},
				func(resp *messages.SecretsGetResp) { cmd.Println(resp.Value) },
			)
		},
	}

	listCmd := &cobra.Command{
		Use: "list", Short: "List all secrets (names only, not values)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, messages.SecretsListMsg{},
				func(resp *messages.SecretsListResp) {
					tw := tabwriter.NewWriter(w(cmd), 0, 0, 2, ' ', 0)
					fmt.Fprintln(tw, "NAME\tVERSION\tUPDATED")
					for _, s := range resp.Secrets {
						fmt.Fprintf(tw, "%s\t%d\t%s\n", s.Name, s.Version, s.UpdatedAt)
					}
					tw.Flush()
				},
			)
		},
	}

	deleteCmd := &cobra.Command{
		Use: "delete <name>", Short: "Delete a secret", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, messages.SecretsDeleteMsg{Name: args[0]},
				func(resp *messages.SecretsDeleteResp) { cmd.Printf("Secret %s deleted\n", args[0]) },
			)
		},
	}

	secretsCmd.AddCommand(setCmd, getCmd, listCmd, deleteCmd)
	return secretsCmd
}
