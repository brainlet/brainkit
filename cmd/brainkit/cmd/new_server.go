package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// newServerSubCmd scaffolds a brainkit server project — a Go main
// that composes brainkit.Server with a YAML config. Generated from
// the final session-11 server package template.
func newServerSubCmd() *cobra.Command {
	var newSrvDir string

	c := &cobra.Command{
		Use:   "server <name>",
		Short: "Scaffold a brainkit server project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dir := newSrvDir
			if dir == "" {
				dir = name
			}
			if _, err := os.Stat(dir); err == nil {
				return fmt.Errorf("directory %s already exists", dir)
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
			if err := os.MkdirAll(filepath.Join(dir, "data"), 0755); err != nil {
				return fmt.Errorf("create data directory: %w", err)
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
)

func main() {
	cfg := flag.String("config", "brainkit.yaml", "path to server config")
	flag.Parse()

	c, err := server.LoadConfig(*cfg)
	if err != nil {
		log.Fatalf("load config: %%v", err)
	}

	srv, err := server.New(c)
	if err != nil {
		log.Fatalf("build server: %%v", err)
	}
	defer srv.Close()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("%%s listening", %q)
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server start: %%v", err)
	}
}
`, name),

				"brainkit.yaml": fmt.Sprintf(`# brainkit server config for %s.
# Environment variables referenced as $NAME or ${NAME} are substituted
# at load time; missing variables expand to empty strings.

namespace: %s
fs_root: ./data

transport:
  type: embedded

gateway:
  listen: :8080

# Providers, storages, vectors, plugins, audit, tracing, probes —
# uncomment and populate as needed.
#
# providers:
#   - name: openai
#     type: openai
#     api_key: $OPENAI_API_KEY
#
# storages:
#   default:
#     type: sqlite
#     path: ./data/default.db
`, name, name),

				"go.mod": fmt.Sprintf(`module %s

go 1.26.0
`, name),

				"README.md": fmt.Sprintf(`# %s

brainkit server scaffolded via ` + "`brainkit new server`" + `.

## Run

` + "```" + `sh
go run . --config brainkit.yaml
` + "```" + `

The HTTP gateway listens on ` + "`:8080`" + ` by default. Override with
` + "`gateway.listen`" + ` in ` + "`brainkit.yaml`" + `.

Deploy packages under ` + "`./packages/<name>/`" + ` and list them under the
top-level ` + "`packages:`" + ` key to auto-deploy on boot.
`, name),

				".gitignore": `# brainkit runtime state
/data
*.db
/nats-data
`,
			}

			for path, content := range files {
				fullPath := filepath.Join(dir, path)
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("write %s: %w", path, err)
				}
			}

			cmd.Printf("Created server %s in %s/\n", name, dir)
			for path := range files {
				cmd.Printf("  %s\n", path)
			}
			cmd.Println()
			cmd.Println("Next steps:")
			cmd.Printf("  cd %s && go mod tidy\n", dir)
			cmd.Println("  go run . --config brainkit.yaml")
			return nil
		},
	}
	c.Flags().StringVar(&newSrvDir, "dir", "", "output directory (default: ./<name>)")
	return c
}
