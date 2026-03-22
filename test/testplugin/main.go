// testplugin is a minimal brainkit plugin for e2e testing.
// It registers an "echo" tool and a "concat" tool, and uses plugin state.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

type EchoInput struct {
	Message string `json:"message"`
}

type EchoOutput struct {
	Echoed  string `json:"echoed"`
	Plugin  string `json:"plugin"`
}

type ConcatInput struct {
	A string `json:"a"`
	B string `json:"b"`
}

type ConcatOutput struct {
	Result string `json:"result"`
}

func main() {
	plugin := sdk.New("test", "testplugin", "1.0.0", sdk.WithDescription("Test plugin for e2e"))

	// Echo tool — echoes input and stamps the plugin name
	sdk.Tool[EchoInput, EchoOutput](plugin, "echo", "echoes the message with plugin stamp",
		func(ctx context.Context, rt sdk.Runtime, in EchoInput) (EchoOutput, error) {
			return EchoOutput{
				Echoed: in.Message,
				Plugin: "testplugin",
			}, nil
		})

	// Concat tool — concatenates two strings, also tests state
	sdk.Tool[ConcatInput, ConcatOutput](plugin, "concat", "concatenates two strings",
		func(ctx context.Context, rt sdk.Runtime, in ConcatInput) (ConcatOutput, error) {
			// Test plugin state: increment call count
			countResp, _ := sdk.PublishAwait[messages.PluginStateGetMsg, messages.PluginStateGetResp](rt, ctx, messages.PluginStateGetMsg{Key: "callCount"})
			count := 0
			if countResp.Value != "" {
				fmt.Sscanf(countResp.Value, "%d", &count)
			}
			count++
			sdk.PublishAwait[messages.PluginStateSetMsg, messages.PluginStateSetResp](rt, ctx, messages.PluginStateSetMsg{
				Key:   "callCount",
				Value: fmt.Sprintf("%d", count),
			})

			return ConcatOutput{
				Result: in.A + in.B,
			}, nil
		})

	// OnStart: log that we're running
	plugin.OnStart(func(rt sdk.Runtime) error {
		log.Println("[testplugin] started successfully")

		// List tools on the host to verify connectivity
		resp, err := sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](rt, context.Background(), messages.ToolListMsg{})
		if err != nil {
			log.Printf("[testplugin] failed to list host tools: %v", err)
		} else {
			log.Printf("[testplugin] host has %d tools", len(resp.Tools))
		}
		return nil
	})

	if err := plugin.Run(); err != nil {
		log.Fatalf("[testplugin] fatal: %v", err)
	}
}
