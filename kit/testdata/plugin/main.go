package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

type EchoInput struct {
	Message string `json:"message"`
}

type EchoOutput struct {
	Message string `json:"message"`
}

type TestEvent struct {
	Data string `json:"data"`
}

type testAckEvent struct {
	Data string `json:"data"`
}

func (testAckEvent) BusTopic() string { return "test.ack" }

func main() {
	p := sdk.New("brainlet", "test-echo", "1.0.0",
		sdk.WithDescription("Test plugin that echoes tool calls"),
	)

	// Echo tool — always registered, all modes
	sdk.Tool(p, "echo", "Echo input", func(ctx context.Context, client sdk.Client, in EchoInput) (EchoOutput, error) {
		return EchoOutput{Message: in.Message}, nil
	})

	// Mode-specific tools
	mode := os.Getenv("TEST_PLUGIN_MODE")
	switch mode {
	case "timeout":
		sdk.Tool(p, "hang", "Never responds", func(ctx context.Context, client sdk.Client, in json.RawMessage) (json.RawMessage, error) {
			select {} // block forever
		})

	case "crash":
		sdk.Tool(p, "crash", "Crashes the process", func(ctx context.Context, client sdk.Client, in json.RawMessage) (json.RawMessage, error) {
			os.Exit(1)
			return nil, nil
		})

	case "slow":
		sdk.Tool(p, "slow", "Slow handler", func(ctx context.Context, client sdk.Client, in json.RawMessage) (json.RawMessage, error) {
			time.Sleep(100 * time.Millisecond)
			return in, nil
		})

	case "ask-kit":
		sdk.Tool(p, "ask-kit", "Calls a Kit tool", func(ctx context.Context, c sdk.Client, in json.RawMessage) (json.RawMessage, error) {
			var req struct {
				Tool  string          `json:"tool"`
				Input json.RawMessage `json:"input"`
			}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}

			resultCh := make(chan []byte, 1)
			c.Ask(ctx, messages.ToolCallMsg{Name: req.Tool, Input: req.Input},
				func(msg messages.Message) {
					resultCh <- msg.Payload
				},
			)

			select {
			case result := <-resultCh:
				return result, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Second):
				return nil, fmt.Errorf("timeout calling Kit tool")
			}
		})

	case "intercept":
		sdk.Intercept(p, "test-interceptor", 200, "tools.*",
			func(ctx context.Context, msg sdk.InterceptMessage) (*sdk.InterceptMessage, error) {
				if msg.Metadata == nil {
					msg.Metadata = make(map[string]string)
				}
				msg.Metadata["x-intercepted-by"] = "test-plugin"
				return &msg, nil
			},
		)

	case "intercept-block":
		sdk.Intercept(p, "blocker", 100, "tools.*",
			func(ctx context.Context, msg sdk.InterceptMessage) (*sdk.InterceptMessage, error) {
				return nil, fmt.Errorf("blocked by test interceptor")
			},
		)

	case "intercept-slow":
		sdk.Intercept(p, "slow-interceptor", 200, "tools.*",
			func(ctx context.Context, msg sdk.InterceptMessage) (*sdk.InterceptMessage, error) {
				time.Sleep(10 * time.Second)
				return &msg, nil
			},
		)
	}

	// Subscribe to test.events.* (all modes)
	sdk.On[TestEvent](p, "test.events.*", func(ctx context.Context, event TestEvent, client sdk.Client) {
		client.Send(ctx, testAckEvent{Data: event.Data})
	})

	p.OnStart(func(client sdk.Client) error {
		return nil
	})

	p.OnStop(func() error {
		return nil
	})

	p.Run()
}
