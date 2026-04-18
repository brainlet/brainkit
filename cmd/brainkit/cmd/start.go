package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit/server"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var configPath string
	c := &cobra.Command{
		Use:   "start",
		Short: "Start a brainkit server from a YAML config",
		Long: `Start loads a brainkit server from a YAML config file and runs it
until SIGINT or SIGTERM. The config file shape is documented at
brainkit/server/testdata/example.yaml; environment variables
referenced as $VAR or ${VAR} are substituted at load time.

The composed Server wires the standard module set (gateway,
probes, tracing, audit) and registers POST /api/bus +
POST /api/stream on the gateway — the canonical entry points
used by "brainkit deploy", "brainkit call", "brainkit inspect".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := server.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("load config %q: %w", configPath, err)
			}

			srv, err := server.New(cfg)
			if err != nil {
				return fmt.Errorf("build server: %w", err)
			}
			defer srv.Close()

			ctx, stop := signal.NotifyContext(context.Background(),
				syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			slog.Info("brainkit started",
				slog.String("namespace", cfg.Namespace),
				slog.String("listen", cfg.Gateway.Listen),
				slog.String("fs_root", cfg.FSRoot),
			)
			return srv.Start(ctx)
		},
	}
	c.Flags().StringVarP(&configPath, "config", "c", "brainkit.yaml", "path to server config YAML")
	return c
}
