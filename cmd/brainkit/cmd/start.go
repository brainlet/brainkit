package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit"
	cliconfig "github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start a brainkit instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cliconfig.LoadConfig()
			if err != nil {
				return err
			}
			nodeCfg, err := cliconfig.BuildNodeConfig(cfg)
			if err != nil {
				return fmt.Errorf("build config: %w", err)
			}
			node, err := brainkit.NewNode(nodeCfg)
			if err != nil {
				return fmt.Errorf("create node: %w", err)
			}
			if err := node.Start(context.Background()); err != nil {
				node.Close()
				return fmt.Errorf("start: %w", err)
			}
			cmd.Println("brainkit started")
			cmd.Printf("  namespace:  %s\n", node.Kernel.Namespace())
			cmd.Printf("  transport:  %s\n", cfg.Transport)
			if cfg.FSRoot != "" {
				cmd.Printf("  workspace:  %s\n", cfg.FSRoot)
			}
			cmd.Println("\nPress Ctrl+C to stop.")

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			cmd.Println("\nShutting down...")
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			return node.Shutdown(ctx)
		},
	}
}
