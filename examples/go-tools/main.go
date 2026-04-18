// Command go-tools demonstrates how to register typed Go
// functions as first-class brainkit tools. Deployed .ts code and
// Go callers both invoke the tools over the bus at tools.call.
//
// Run from the repo root:
//
//	go run ./examples/go-tools
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

// WeatherInput / WeatherOutput are the typed I/O for the weather
// tool. JSON tags drive both the wire format and the autogen-
// erated JSON Schema exposed on tools.list.
type WeatherInput struct {
	City string `json:"city"`
}

type WeatherOutput struct {
	City      string `json:"city"`
	TempC     int    `json:"tempC"`
	Condition string `json:"condition"`
}

// AddInput / AddOutput — a pure-function tool, useful for showing
// that typed tools are unrestricted about what they compute.
type AddInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddOutput struct {
	Sum int `json:"sum"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("go-tools: %v", err)
	}
}

func run() error {
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "go-tools-demo",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	// Register two typed Go tools. Schema is derived from
	// `WeatherInput` / `AddInput` struct tags via reflection;
	// `tools.list` returns the generated schema so callers can
	// validate inputs up-front.
	if err := brainkit.RegisterTool(kit, "weather", brainkit.TypedTool[WeatherInput]{
		Description: "Stubbed weather lookup for demo purposes.",
		Execute: func(_ context.Context, in WeatherInput) (any, error) {
			return WeatherOutput{City: in.City, TempC: 18, Condition: "cloudy"}, nil
		},
	}); err != nil {
		return fmt.Errorf("register weather: %w", err)
	}

	if err := brainkit.RegisterTool(kit, "math.add", brainkit.TypedTool[AddInput]{
		Description: "Return a + b as a typed sum.",
		Execute: func(_ context.Context, in AddInput) (any, error) {
			return AddOutput{Sum: in.A + in.B}, nil
		},
	}); err != nil {
		return fmt.Errorf("register math.add: %w", err)
	}

	// Deploy a .ts that calls both tools via bus.call and
	// collates the results.
	tsCode := `
		bus.on("demo", async (msg) => {
			const weather = await bus.call("tools.call", {
				name: "weather",
				input: { city: msg.payload.city },
			}, { timeoutMs: 2000 });
			const sum = await bus.call("tools.call", {
				name: "math.add",
				input: { a: 2, b: 3 },
			}, { timeoutMs: 2000 });
			msg.reply({
				weather: weather.result,
				sum: sum.result,
			});
		});
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("go-tools-demo", "demo.ts", tsCode)); err != nil {
		return fmt.Errorf("deploy demo.ts: %w", err)
	}

	// Invoke the .ts handler from Go. The .ts in turn invokes
	// both Go tools over the bus.
	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.go-tools-demo.demo",
		Payload: json.RawMessage(`{"city":"Paris"}`),
	}, brainkit.WithCallTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("call .ts: %w", err)
	}

	fmt.Println("typed Go tools invoked from .ts:")
	fmt.Println(string(reply))

	// Show that the same tools are invokable directly from Go
	// through the generated CallToolCall wrapper.
	direct, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{
		Name:  "math.add",
		Input: map[string]any{"a": 40, "b": 2},
	}, brainkit.WithCallTimeout(2*time.Second))
	if err != nil {
		return fmt.Errorf("call math.add directly: %w", err)
	}

	fmt.Println("same tool invoked directly from Go (math.add):")
	fmt.Println(string(direct.Result))
	return nil
}
