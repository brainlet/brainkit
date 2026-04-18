// Command hello-embedded demonstrates the library-mode usage of
// brainkit: build a Kit in-process, deploy a .ts handler, call the
// handler via the typed Call generic, print the reply.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "hello-embedded",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
	})
	if err != nil {
		log.Fatalf("new kit: %v", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deploy a tiny .ts package that answers "hello".
	if _, err := kit.Deploy(ctx, brainkit.PackageInline(
		"greeter", "greeter.ts",
		`bus.on("hello", (msg) => msg.reply({ greeting: "hello, " + msg.payload.name }));`,
	)); err != nil {
		log.Fatalf("deploy: %v", err)
	}

	// Call the deployed handler.
	payload, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](
		kit, ctx,
		sdk.CustomMsg{
			Topic:   "ts.greeter.hello",
			Payload: json.RawMessage(`{"name":"world"}`),
		},
		brainkit.WithCallTimeout(2*time.Second),
	)
	if err != nil {
		log.Fatalf("call: %v", err)
	}

	fmt.Println(string(payload))
}
