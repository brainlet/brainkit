package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
			if _, err := os.Stat(dir); err == nil {
				return fmt.Errorf("directory %s already exists", dir)
			}
			typesDir := filepath.Join(dir, "types")
			if err := os.MkdirAll(typesDir, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}

			files := map[string]string{
				"manifest.json": fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "description": "",
  "entry": "index.ts"
}
`, name),
				"index.ts": fmt.Sprintf(`import { bus } from "kit";

bus.on("hello", (msg) => {
  msg.reply({ message: "Hello from %s!" });
});
`, name),
				"tsconfig.json": `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ES2022",
    "moduleResolution": "bundler",
    "strict": false,
    "noImplicitAny": false,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "paths": {
      "kit": ["./types/kit.d.ts"],
      "ai": ["./types/ai.d.ts"],
      "agent": ["./types/agent.d.ts"]
    }
  },
  "include": ["*.ts", "./types/globals.d.ts"]
}
`,
				filepath.Join("types", "kit.d.ts"):     brainkit.KitDTS,
				filepath.Join("types", "ai.d.ts"):      brainkit.AiDTS,
				filepath.Join("types", "agent.d.ts"):    brainkit.AgentDTS,
				filepath.Join("types", "brainkit.d.ts"): brainkit.BrainkitDTS,
				filepath.Join("types", "globals.d.ts"):  brainkit.GlobalsDTS,
			}

			for path, content := range files {
				fullPath := filepath.Join(dir, path)
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("write %s: %w", path, err)
				}
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
