package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var newModuleDir string

var newModuleCmd = &cobra.Command{
	Use:   "module <name>",
	Short: "Scaffold a .ts module project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir := newModuleDir
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
  "services": {
    "hello": { "entry": "hello.ts" }
  }
}
`, name),
			"hello.ts": fmt.Sprintf(`import { bus, kit } from "kit";

bus.on("greet", (msg) => {
  msg.reply({ message: "Hello from " + kit.source });
});
`),
			"tsconfig.json": `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ES2020",
    "moduleResolution": "node",
    "strict": true,
    "paths": {
      "kit": ["./types/kit.d.ts"],
      "ai": ["./types/ai.d.ts"],
      "agent": ["./types/agent.d.ts"],
      "test": ["./types/test.d.ts"]
    }
  },
  "include": ["*.ts"],
  "exclude": ["types"]
}
`,
			filepath.Join("types", "kit.d.ts"):   kitDTS,
			filepath.Join("types", "ai.d.ts"):    aiDTS,
			filepath.Join("types", "agent.d.ts"):  agentDTS,
			filepath.Join("types", "test.d.ts"):   testDTS,
		}

		for path, content := range files {
			fullPath := filepath.Join(dir, path)
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}

		fmt.Printf("Created module %s in %s/\n", name, dir)
		fmt.Println("  manifest.json")
		fmt.Println("  hello.ts")
		fmt.Println("  tsconfig.json")
		fmt.Println("  types/kit.d.ts")
		fmt.Println("  types/ai.d.ts")
		fmt.Println("  types/agent.d.ts")
		fmt.Println("  types/test.d.ts")
		fmt.Printf("\nDeploy: brainkit deploy %s/\n", dir)
		return nil
	},
}

func init() {
	newModuleCmd.Flags().StringVar(&newModuleDir, "dir", "", "output directory (default: ./<name>)")
	newCmd.AddCommand(newModuleCmd)
}
