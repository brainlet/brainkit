// Command hello-server shows the smallest brainkit/server program:
// load a YAML config, build the composed runtime, run until signal.
// Matches the shape that `brainkit new server` scaffolds into
// downstream projects.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit/server"
)

func main() {
	cfgPath := flag.String("config", "examples/hello-server/brainkit.yaml",
		"path to server config")
	flag.Parse()

	cfg, err := server.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}
	defer srv.Close()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// The gateway module logs its own "listening" line at Init time.
	log.Printf("brainkit %s up — fs_root=%s", cfg.Namespace, cfg.FSRoot)
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server start: %v", err)
	}
}
