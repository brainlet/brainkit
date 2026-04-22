package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/brainlet/brainkit"
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
		Short: "Inspect the module registry baked into this binary",
	}
	c.AddCommand(newModulesListCmd())
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
	fmt.Fprintln(tw, "NAME\tSTATUS\tSUMMARY")
	for _, d := range descs {
		status := d.Status
		if status == "" {
			status = "-"
		}
		summary := d.Summary
		if summary == "" {
			summary = "(no summary)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", d.Name, status, summary)
	}
	tw.Flush()
}

func mustMarshalDescs(descs []brainkit.ModuleDescriptor) json.RawMessage {
	raw, _ := json.MarshalIndent(descs, "", "  ")
	return raw
}
