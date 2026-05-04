// Command multi-kit runs two brainkit Kits in one process and
// resolves cross-kit calls by peer name through modules/topology.
// It's the smallest useful demo of cross-kit routing without
// requiring an external transport — both kits share an in-process
// memory bus wired through brainkit.Memory(). Cross-kit *replies*
// need a real transport in production; this example focuses on the
// name-to-namespace resolution step that WithCallTo exercises.
package main

import (
	"context"
	"fmt"
	bkmodule "github.com/brainlet/brainkit/module"
	"log"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/topology"
	"github.com/brainlet/brainkit/transports/embeddednats"
)

func main() {
	// The target Kit publishes a "hello" handler under its own
	// namespace. It doesn't know anything about topology — it just
	// serves its own mailbox.
	target, err := brainkit.New(brainkit.Config{
		Namespace: "analytics-prod",
		Transport: embeddednats.New(),
		FSRoot:    ".",
	})
	if err != nil {
		log.Fatalf("target: %v", err)
	}
	defer target.Close()

	// The caller Kit wires the topology module with a single peer
	// entry mapping "analytics" → "analytics-prod". Later calls
	// through WithCallTo("analytics") resolve to that namespace.
	caller, err := brainkit.New(brainkit.Config{
		Namespace: "orchestrator",
		Transport: embeddednats.New(),
		FSRoot:    ".",
		Modules: []bkmodule.Module{
			topology.NewModule(topology.Config{
				Peers: []topology.Peer{
					{Name: "analytics", Namespace: "analytics-prod"},
				},
			}),
		},
	})
	if err != nil {
		log.Fatalf("caller: %v", err)
	}
	defer caller.Close()

	// Ask the topology module directly — same codepath WithCallTo
	// uses under the hood.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := brainkit.Call[topology.PeersResolveMsg, topology.PeersResolveResp](
		caller, ctx, topology.PeersResolveMsg{Name: "analytics"},
	)
	if err != nil {
		log.Fatalf("resolve: %v", err)
	}
	fmt.Printf("analytics resolves to %q\n", resp.Namespace)
}
