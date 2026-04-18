// Command gateway-routes spins up a Kit + gateway module, registers
// an HTTP route that forwards to a deployed .ts handler, and prints
// the gateway's listen address. Hit `GET /hello?name=world` to see
// the round-trip in action.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
)

func main() {
	// Pick a fresh port so running the example twice never collides.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("probe listen: %v", err)
	}
	listenAddr := probe.Addr().String()
	_ = probe.Close()

	gw := gateway.New(gateway.Config{Listen: listenAddr})
	// Route /hello onto the bus topic ts.greeter.hello.
	gw.Handle(http.MethodGet, "/hello", "ts.greeter.hello")

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "gateway-routes",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Modules:   []brainkit.Module{gw},
	})
	if err != nil {
		log.Fatalf("new kit: %v", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline(
		"greeter", "greeter.ts",
		`bus.on("hello", (msg) => {
			const name = (msg.payload && msg.payload.name) || "stranger";
			msg.reply({ greeting: "hello, " + name });
		});`,
	)); err != nil {
		log.Fatalf("deploy: %v", err)
	}

	fmt.Printf("listening on http://%s\n", listenAddr)
	fmt.Printf("  curl 'http://%s/hello?name=world'\n", listenAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
