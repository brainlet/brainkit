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

var startCmd = &cobra.Command{
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

		fmt.Println("brainkit started")
		fmt.Printf("  namespace:  %s\n", node.Kernel.Namespace())
		fmt.Printf("  transport:  %s\n", cfg.Transport)
		if cfg.FSRoot != "" {
			fmt.Printf("  workspace:  %s\n", cfg.FSRoot)
		}
		fmt.Println("\nPress Ctrl+C to stop.")

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return node.Shutdown(ctx)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
