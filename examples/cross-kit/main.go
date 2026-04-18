// Command cross-kit shows two Kits sharing a single in-process
// NATS server, routing Kit B → Kit A by peer name through
// modules/topology + brainkit.WithCallTo.
//
// Uses github.com/nats-io/nats-server/v2 directly to boot a
// standalone NATS server both Kits can connect to. That's the
// pragmatic way to share a transport between two Kits in one
// process — brainkit.EmbeddedNATS() is per-Kit and doesn't
// expose its URL.
//
// Run from the repo root:
//
//	go run ./examples/cross-kit
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/topology"
	"github.com/brainlet/brainkit/sdk"
	natsserver "github.com/nats-io/nats-server/v2/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("cross-kit: %v", err)
	}
}

func run() error {
	natsSrv, err := startNATS()
	if err != nil {
		return fmt.Errorf("start nats: %w", err)
	}
	defer natsSrv.Shutdown()
	natsURL := natsSrv.ClientURL()
	fmt.Printf("shared NATS at %s\n", natsURL)

	// Target kit — `analytics-prod` — answers a report request.
	target, err := brainkit.New(brainkit.Config{
		Namespace: "analytics-prod",
		CallerID:  "analytics-prod",
		Transport: brainkit.NATS(natsURL),
		FSRoot:    ".",
	})
	if err != nil {
		return fmt.Errorf("target kit: %w", err)
	}
	defer target.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := target.Deploy(ctx, brainkit.PackageInline("report-svc", "report.ts", `
		bus.on("quarterly", (msg) => {
			msg.reply({ revenue: 1234567, quarter: msg.payload.quarter });
		});
	`)); err != nil {
		return fmt.Errorf("deploy report-svc: %w", err)
	}

	// Caller kit — `orchestrator` — wires topology with a static
	// peer pointing at analytics-prod.
	caller, err := brainkit.New(brainkit.Config{
		Namespace: "orchestrator",
		CallerID:  "orchestrator",
		Transport: brainkit.NATS(natsURL),
		FSRoot:    ".",
		Modules: []brainkit.Module{
			topology.NewModule(topology.Config{
				Peers: []topology.Peer{
					{Name: "analytics", Namespace: "analytics-prod"},
				},
			}),
		},
	})
	if err != nil {
		return fmt.Errorf("caller kit: %w", err)
	}
	defer caller.Close()

	// Cross-kit call: caller publishes to ts.report-svc.quarterly
	// via WithCallTo("analytics") → topology resolves to the
	// analytics-prod namespace → NATS routes to the target kit.
	fmt.Println("orchestrator → analytics (WithCallTo):")
	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](caller, ctx,
		sdk.CustomMsg{
			Topic:   "ts.report-svc.quarterly",
			Payload: json.RawMessage(`{"quarter":"Q4"}`),
		},
		brainkit.WithCallTo("analytics"),
		brainkit.WithCallTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}

	var pretty any
	if err := json.Unmarshal(reply, &pretty); err == nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("  ", "  ")
		_ = enc.Encode(pretty)
	} else {
		fmt.Println(string(reply))
	}
	return nil
}

func startNATS() (*natsserver.Server, error) {
	opts := &natsserver.Options{
		Host:      "127.0.0.1",
		Port:      -1, // pick an open port
		JetStream: true,
		StoreDir:  mustTempDir("brainkit-cross-kit-nats-"),
	}
	s, err := natsserver.NewServer(opts)
	if err != nil {
		return nil, err
	}
	go s.Start()
	if !s.ReadyForConnections(5 * time.Second) {
		return nil, fmt.Errorf("NATS server not ready in 5s")
	}
	return s, nil
}

func mustTempDir(prefix string) string {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		log.Fatalf("tempdir: %v", err)
	}
	return dir
}
