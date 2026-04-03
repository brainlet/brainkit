package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var newPluginOwner string

var newPluginCmd = &cobra.Command{
	Use:   "plugin <name>",
	Short: "Scaffold a Go plugin project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		owner := newPluginOwner

		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("directory %s already exists", name)
		}

		if err := os.MkdirAll(name, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		files := map[string]string{
			"main.go": fmt.Sprintf(`package main

import (
	"context"

	"github.com/brainlet/brainkit/sdk"
)

func main() {
	p := sdk.New("%s", "%s", "0.1.0",
		sdk.WithDescription("A brainkit plugin"),
	)

	sdk.Tool[EchoInput, EchoOutput](p, "echo", "Echo the input",
		func(ctx context.Context, client sdk.Client, in EchoInput) (EchoOutput, error) {
			return EchoOutput{Echoed: in.Message}, nil
		},
	)

	p.Run()
}

type EchoInput struct {
	Message string `+"`"+`json:"message"`+"`"+`
}
type EchoOutput struct {
	Echoed string `+"`"+`json:"echoed"`+"`"+`
}
`, owner, name),
			"go.mod": fmt.Sprintf(`module github.com/%s/%s

go 1.26

require github.com/brainlet/brainkit v0.0.0
`, owner, name),
			"manifest.json": fmt.Sprintf(`{
  "name": "%s",
  "owner": "%s",
  "version": "0.1.0",
  "description": "A brainkit plugin",
  "capabilities": [],
  "platforms": {}
}
`, name, owner),
		}

		for path, content := range files {
			fullPath := filepath.Join(name, path)
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}

		fmt.Printf("Created plugin %s/%s in %s/\n", owner, name, name)
		fmt.Println("  main.go")
		fmt.Println("  go.mod")
		fmt.Println("  manifest.json")
		fmt.Printf("\nBuild: cd %s && go build -o %s .\n", name, name)
		fmt.Printf("Start: brainkit plugin start %s --binary ./%s/%s\n", name, name, name)
		return nil
	},
}

func init() {
	newPluginCmd.Flags().StringVar(&newPluginOwner, "owner", "yourorg", "plugin owner organization")
	newCmd.AddCommand(newPluginCmd)
}
