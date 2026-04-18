package cmd

import (
	"fmt"

	"github.com/brainlet/brainkit"
	"github.com/spf13/cobra"
)

func newPackageSubCmd() *cobra.Command {
	var newPkgDir string

	c := &cobra.Command{
		Use:   "package <name>",
		Short: "Scaffold a brainkit package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dir := newPkgDir
			if dir == "" {
				dir = name
			}
			source := fmt.Sprintf(`import { bus } from "kit";

bus.on("hello", (msg) => {
  msg.reply({ message: "Hello from %s!" });
});
`, name)
			if err := brainkit.ScaffoldPackage(dir, name, "index.ts", source); err != nil {
				return err
			}
			cmd.Printf("Created package %s in %s/\n", name, dir)
			cmd.Println("  manifest.json")
			cmd.Println("  index.ts")
			cmd.Println("  tsconfig.json")
			cmd.Println("  types/kit.d.ts")
			cmd.Println("  types/ai.d.ts")
			cmd.Println("  types/agent.d.ts")
			cmd.Println("  types/brainkit.d.ts")
			cmd.Println("  types/globals.d.ts")
			cmd.Printf("\nDeploy: brainkit deploy %s/\n", dir)
			return nil
		},
	}
	c.Flags().StringVar(&newPkgDir, "dir", "", "output directory (default: ./<name>)")
	return c
}
