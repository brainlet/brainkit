package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// newSecretsCmd groups the secret-store verbs: set / get / list /
// delete. All five land on the running server's `secrets.*` bus
// topics via the configured gateway endpoint.
func newSecretsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "secrets",
		Short: "Manage brainkit secrets",
		Long: `Set, read, list, or delete values in the running server's secret
store. Secrets are stored encrypted at rest when the server is
launched with a secret key; otherwise they're kept in plaintext in
the configured storage backend (with a warning at startup).`,
	}
	c.AddCommand(
		newSecretsSetCmd(),
		newSecretsGetCmd(),
		newSecretsListCmd(),
		newSecretsDeleteCmd(),
	)
	return c
}

func newSecretsSetCmd() *cobra.Command {
	var (
		endpoint string
		stdinIn  bool
	)
	c := &cobra.Command{
		Use:   "set <name> [value]",
		Short: "Set a secret",
		Long: `Set writes a named secret. The value can come from the argument,
--stdin (read one line), or the BRAINKIT_SECRET environment variable
(useful for CI without putting the secret in the command history).`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var value string
			switch {
			case stdinIn:
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					value = scanner.Text()
				}
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
			case len(args) > 1:
				value = args[1]
			default:
				if v, ok := os.LookupEnv("BRAINKIT_SECRET"); ok {
					value = v
				} else {
					return fmt.Errorf("provide value as arg, --stdin, or $BRAINKIT_SECRET")
				}
			}

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			client := newBusClient(endpoint)
			body, _ := json.Marshal(map[string]string{"name": name, "value": value})
			reply, err := client.call(ctx, "secrets.set", body)
			if err != nil {
				return err
			}
			var resp struct {
				Stored  bool `json:"stored"`
				Version int  `json:"version"`
			}
			_ = json.Unmarshal(reply, &resp)
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			cmd.Printf("Secret %s set (version %d)\n", name, resp.Version)
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	c.Flags().BoolVar(&stdinIn, "stdin", false, "read value from stdin")
	return c
}

func newSecretsGetCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "get <name>",
		Short: "Read a secret value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)
			body, _ := json.Marshal(map[string]string{"name": args[0]})
			reply, err := client.call(ctx, "secrets.get", body)
			if err != nil {
				return err
			}
			var resp struct {
				Value string `json:"value"`
			}
			_ = json.Unmarshal(reply, &resp)
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			cmd.Println(resp.Value)
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

func newSecretsListCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "list",
		Short: "List every stored secret (names only, no values)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)
			reply, err := client.call(ctx, "secrets.list", json.RawMessage("{}"))
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			var resp struct {
				Secrets []struct {
					Name      string `json:"name"`
					Version   int    `json:"version"`
					UpdatedAt string `json:"updatedAt"`
				} `json:"secrets"`
			}
			if err := json.Unmarshal(reply, &resp); err != nil {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			return renderSecretsTable(cmd.OutOrStdout(), resp.Secrets)
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

func newSecretsDeleteCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)
			body, _ := json.Marshal(map[string]string{"name": args[0]})
			reply, err := client.call(ctx, "secrets.delete", body)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			cmd.Printf("Secret %s deleted\n", args[0])
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

func renderSecretsTable(w io.Writer, secrets []struct {
	Name      string `json:"name"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVERSION\tUPDATED")
	for _, s := range secrets {
		fmt.Fprintf(tw, "%s\t%d\t%s\n", nonEmpty(s.Name, "-"), s.Version, nonEmpty(s.UpdatedAt, "-"))
	}
	return tw.Flush()
}
