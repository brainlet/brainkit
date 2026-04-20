package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

// newTeardownCmd removes a deployed package by name. Mirrors the
// bus command the old CLI exposed via `brainkit teardown`; today
// it's the only way to evict a package without a full server
// restart short of issuing `brainkit call package.teardown ...`
// manually.
func newTeardownCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "teardown <name>",
		Short: "Remove a deployed package from a running server",
		Long: `Teardown publishes a package.teardown command for <name>. The
server unregisters every subscription, schedule, and resource owned
by that deployment and frees the Compartment.

Takes the package name (not the source path). Example:

  brainkit deploy ./hello
  brainkit teardown hello`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			client := newBusClient(endpoint)
			body, _ := json.Marshal(map[string]string{"name": args[0]})
			reply, err := client.call(ctx, "package.teardown", body)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			cmd.Printf("Package %s torn down\n", args[0])
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

