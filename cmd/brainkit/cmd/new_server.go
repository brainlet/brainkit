package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newServerSubCmd() *cobra.Command {
	var serverDir string

	c := &cobra.Command{
		Use:   "server <name>",
		Short: "Scaffold a brainkit server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dir := serverDir
			if dir == "" {
				dir = name
			}
			if _, err := os.Stat(dir); err == nil {
				return fmt.Errorf("directory %s already exists", dir)
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}

			files := map[string]string{
				"main.go": fmt.Sprintf(`package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit/server"
	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/packageboot"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
	_ "github.com/brainlet/brainkit/server/configfile/transportbackends/embeddednats"
	_ "github.com/brainlet/brainkit/server/standard/commands"
	_ "github.com/brainlet/brainkit/server/standard/packages"
	_ "github.com/brainlet/brainkit/server/standard/server"
)

func main() {
	cfgPath := flag.String("config", "brainkit.yaml", "path to server config")
	flag.Parse()

	cfg, err := configfile.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %%v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("build server: %%v", err)
	}
	defer srv.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server start: %%v", err)
	}
}
`),
				"brainkit.yaml": fmt.Sprintf(`namespace: %s
fs_root: ./data

transport:
  type: embedded

modules:
  jsruntime: {}
  gateway:
    listen: :8080
  agents: {}
  reference: {}
  control: {}
  eval: {}
  health: {}
  messaging: {}
  metrics: {}
  registry: {}
  secrets: {}
  tools: {}
  packages: {}
  probes: {}
`, name),
				"go.mod": fmt.Sprintf(`module %s

go 1.26

require github.com/brainlet/brainkit v0.0.0
`, name),
				"README.md": fmt.Sprintf(`# %s

Run the server:

`+"```sh"+`
go run . --config brainkit.yaml
`+"```"+`
`, name),
			}

			for path, content := range files {
				fullPath := filepath.Join(dir, path)
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("write %s: %w", path, err)
				}
			}

			cmd.Printf("Created server %s in %s/\n", name, dir)
			cmd.Println("  main.go")
			cmd.Println("  brainkit.yaml")
			cmd.Println("  go.mod")
			cmd.Println("  README.md")
			cmd.Printf("\nRun: cd %s && go run . --config brainkit.yaml\n", dir)
			return nil
		},
	}
	c.Flags().StringVar(&serverDir, "dir", "", "output directory (default: ./<name>)")
	return c
}
