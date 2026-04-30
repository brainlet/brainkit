package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/brainlet/brainkit"
	controlmod "github.com/brainlet/brainkit/modules/control"
	"github.com/spf13/cobra"

	// Blank-imports: the standard module set must register itself in
	// the registry before `brainkit modules list` can read it. The
	// server package already does this, but re-importing here keeps
	// the command useful in a custom binary that doesn't embed
	// server but still wants to introspect its own registry.
	_ "github.com/brainlet/brainkit/modules/audit"
	_ "github.com/brainlet/brainkit/modules/discovery"
	_ "github.com/brainlet/brainkit/modules/gateway"
	_ "github.com/brainlet/brainkit/modules/harness"
	_ "github.com/brainlet/brainkit/modules/mcp"
	_ "github.com/brainlet/brainkit/modules/plugins"
	_ "github.com/brainlet/brainkit/modules/probes"
	_ "github.com/brainlet/brainkit/modules/schedules"
	_ "github.com/brainlet/brainkit/modules/topology"
	_ "github.com/brainlet/brainkit/modules/tracing"
	_ "github.com/brainlet/brainkit/modules/workflow"
)

// newModulesCmd exposes the brainkit module registry on the CLI.
// Custom binaries that blank-import third-party modules will see
// them listed alongside the standard set — so operators can verify
// that the binary on disk actually contains the factory their YAML
// expects before debugging a "unknown module" load failure.
func newModulesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "modules",
		Short: "Inspect and control Brainkit modules",
	}
	c.AddCommand(newModulesListCmd())
	c.AddCommand(newModulesInspectCmd())
	c.AddCommand(newModulesMountCmd())
	c.AddCommand(newModulesUnmountCmd())
	return c
}

func newModulesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every module registered in this binary",
		Long: `List reads the global brainkit module registry populated by
each module package's init() and prints one row per entry.

Use this to verify that a custom binary — one that blank-imports
third-party modules — actually contains the factories the YAML
config references. A missing row means the module isn't compiled
into this binary, which is why "unknown module" errors fire at
server.LoadConfig time.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			descs := brainkit.RegisteredModules()
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), mustMarshalDescs(descs))
			}
			renderModuleTable(cmd.OutOrStdout(), descs)
			return nil
		},
	}
}

func renderModuleTable(w io.Writer, descs []brainkit.ModuleDescriptor) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATUS\tCOMMANDS\tEVENTS\tSUBS\tCAPS\tRES\tSUMMARY")
	for _, d := range descs {
		status := d.Status
		if status == "" {
			status = "-"
		}
		summary := d.Summary
		if summary == "" {
			summary = "(no summary)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\n", d.Name, status, len(d.Commands), len(d.Events), len(d.Subscriptions), len(d.Capabilities), len(d.Resources), summary)
	}
	tw.Flush()
}

func mustMarshalDescs(descs []brainkit.ModuleDescriptor) json.RawMessage {
	raw, _ := json.MarshalIndent(descs, "", "  ")
	return raw
}

func newModulesInspectCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a registered or mounted module in a running server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, _ := json.Marshal(controlmod.KitModuleDescribeMsg{ID: args[0]})
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			raw, err := newBusClient(endpoint).call(ctx, controlmod.KitModuleDescribeMsg{}.BusTopic(), payload)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), raw)
			}
			var resp controlmod.KitModuleDescribeResp
			if err := json.Unmarshal(raw, &resp); err != nil {
				return writeJSONPretty(cmd.OutOrStdout(), raw)
			}
			renderModuleTable(cmd.OutOrStdout(), []brainkit.ModuleDescriptor{resp.Module})
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

func newModulesMountCmd() *cobra.Command {
	var endpoint string
	var configPath string
	var configJSON string
	c := &cobra.Command{
		Use:   "mount <id>",
		Short: "Hot-mount a registered module into a running server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := controlmod.KitModuleMountMsg{ID: args[0]}
			if configPath != "" && configJSON != "" {
				return fmt.Errorf("--config and --config-json are mutually exclusive")
			}
			if configPath != "" {
				raw, err := os.ReadFile(configPath)
				if err != nil {
					return fmt.Errorf("read config %q: %w", configPath, err)
				}
				req.ConfigYAML = string(raw)
			}
			if configJSON != "" {
				raw := json.RawMessage(configJSON)
				if !json.Valid(raw) {
					return fmt.Errorf("--config-json is not valid JSON")
				}
				req.Config = raw
			}
			payload, _ := json.Marshal(req)
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			raw, err := newBusClient(endpoint).call(ctx, req.BusTopic(), payload)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), raw)
			}
			var resp controlmod.KitModuleMountResp
			if err := json.Unmarshal(raw, &resp); err != nil {
				return writeJSONPretty(cmd.OutOrStdout(), raw)
			}
			renderModuleTable(cmd.OutOrStdout(), []brainkit.ModuleDescriptor{resp.Module})
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	c.Flags().StringVar(&configPath, "config", "", "module config YAML/JSON file")
	c.Flags().StringVar(&configJSON, "config-json", "", "module config JSON object")
	return c
}

func newModulesUnmountCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "unmount <id>",
		Short: "Unmount a module from a running server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := controlmod.KitModuleUnmountMsg{ID: args[0]}
			payload, _ := json.Marshal(req)
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			raw, err := newBusClient(endpoint).call(ctx, req.BusTopic(), payload)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), raw)
			}
			var resp controlmod.KitModuleUnmountResp
			if err := json.Unmarshal(raw, &resp); err != nil {
				return writeJSONPretty(cmd.OutOrStdout(), raw)
			}
			renderModuleTable(cmd.OutOrStdout(), []brainkit.ModuleDescriptor{resp.Module})
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}
