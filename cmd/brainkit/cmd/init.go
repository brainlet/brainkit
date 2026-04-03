package cmd

import (
	"fmt"
	"os"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/spf13/cobra"
)

const initTemplate = `namespace: %s

# Environment — loads .env file into os.Getenv before starting.
# AI provider keys (OPENAI_API_KEY, etc.) are auto-detected from env.
env_file: .env

# Bus transport — how CLI commands communicate with the running instance
# sql-sqlite works locally with no external deps. Use nats/redis/amqp for distributed.
transport: sql-sqlite
sqlite_path: ./data/transport.db

# Mastra storage — workflow snapshots, memory, agent state
storage:
  default:
    type: sqlite
    path: ./data/brainkit.db

# Workspace — filesystem root for deployed .ts code
fs_root: ./workspace

# Persistence — deployment/schedule/plugin state across restarts
store_path: ./data/store.db
`

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a brainkit.yaml config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "brainkit.yaml"
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists", path)
			}
			ns := config.DefaultNamespace()
			content := fmt.Sprintf(initTemplate, ns)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
			cmd.Printf("Created %s (namespace: %s)\n", path, ns)
			return nil
		},
	}
}
