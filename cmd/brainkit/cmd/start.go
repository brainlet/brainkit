package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit/server"
	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/packageboot"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends"
	_ "github.com/brainlet/brainkit/server/configfile/transportbackends"
	_ "github.com/brainlet/brainkit/server/standard/full"
	_ "github.com/brainlet/brainkit/storagebridges/sqlite"
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

The CLI links the full standard YAML module catalog; the config
chooses which modules to mount. Gateway modules register POST
/api/bus and POST /api/stream — the canonical entry points used
by "brainkit deploy", "brainkit call", "brainkit inspect".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load `./.env` into the process environment so the yaml's
			// `$VAR` references (OPENAI_API_KEY, etc.) resolve without
			// the user having to `export` each one by hand. Existing
			// env vars always win; missing file is a silent no-op.
			loadDotEnv(".env")

			cfg, err := configfile.Load(configPath)
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

			// The gateway module logs its own "gateway listening"
			// line at mount time; this supervisor log stays high-level
			// with namespace + fs_root so operators have a single
			// anchor for correlation.
			slog.Info("brainkit started",
				slog.String("namespace", cfg.Namespace),
				slog.String("fs_root", cfg.FSRoot),
			)
			return srv.Start(ctx)
		},
	}
	c.Flags().StringVarP(&configPath, "config", "c", "brainkit.yaml", "path to server config YAML")
	return c
}
