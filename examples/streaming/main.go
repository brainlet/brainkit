// Command streaming demonstrates every streaming surface the
// Kit exposes:
//
//   - Bus CallStream: Go caller consumes ordered chunks from a
//     .ts handler that emits via msg.send, then the terminal
//     reply.
//   - Gateway SSE route: curl the Server-Sent Events endpoint
//     and watch chunks arrive.
//   - Gateway WebSocket route: client receives chunks over a
//     bidirectional connection.
//   - Gateway Webhook route: fire-and-forget (no reply).
//
// The Go-side bus path runs automatically when you `go run`
// the example. SSE / WebSocket / Webhook are exercised via
// curl / wscat — the README has the snippets.
//
// Run from the repo root:
//
//	go run ./examples/streaming
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("streaming: %v", err)
	}
}

func run() error {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("probe listen: %w", err)
	}
	listenAddr := probe.Addr().String()
	_ = probe.Close()

	gw := gateway.New(gateway.Config{Listen: listenAddr, Timeout: 30 * time.Second})
	gw.HandleStream("GET", "/sse/count", "ts.streaming-demo.count")
	gw.HandleWebSocket("/ws/count", "ts.streaming-demo.count")
	gw.HandleWebhook("POST", "/webhook/log", "ts.streaming-demo.write")

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "streaming-demo",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Modules:   []brainkit.Module{gw},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Deploy three handlers: counter (stream), chatter (stream),
	// and write (webhook — no reply).
	tsCode := `
		bus.on("count", (msg) => {
			const n = msg.payload.n || 3;
			for (let i = 1; i <= n; i++) msg.send({ tick: i });
			msg.reply({ done: true, total: n });
		});

		bus.on("write", (msg) => {
			// webhook — no reply required; bus still sends "done" ack
			// so the gateway returns 202 Accepted promptly.
			console.log("webhook write:", JSON.stringify(msg.payload));
			msg.reply({ received: true });
		});
	`
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("streaming-demo", "stream.ts", tsCode)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	fmt.Printf("listening on http://%s\n", listenAddr)
	fmt.Printf("  SSE:     curl -N 'http://%s/sse/count?n=5'\n", listenAddr)
	fmt.Printf("  WS:      wscat -c 'ws://%s/ws/count?n=5'\n", listenAddr)
	fmt.Printf("  Webhook: curl -X POST 'http://%s/webhook/log' -d '{\"msg\":\"hi\"}'\n", listenAddr)
	fmt.Println()

	// Bus CallStream — Go side consumes ordered chunks.
	fmt.Println("bus CallStream round trip (Go):")
	chunks := []map[string]any{}
	type countResp struct {
		Done  bool `json:"done"`
		Total int  `json:"total"`
	}
	result, err := brainkit.CallStream[sdk.CustomMsg, map[string]any, countResp](
		kit, ctx,
		sdk.CustomMsg{Topic: "ts.streaming-demo.count", Payload: json.RawMessage(`{"n":5}`)},
		func(chunk map[string]any) error { chunks = append(chunks, chunk); return nil },
		brainkit.WithCallTimeout(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("CallStream: %w", err)
	}
	for _, c := range chunks {
		fmt.Printf("  chunk: %v\n", c)
	}
	fmt.Printf("  terminal: done=%v total=%d\n", result.Done, result.Total)
	fmt.Println()
	fmt.Println("gateway HTTP endpoints are live — hit them from a second shell.")
	fmt.Println("press Ctrl+C to stop.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	return nil
}
