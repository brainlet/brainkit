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
			getResult, _ := sdk.Publish(rt, ctx, messages.PluginStateGetMsg{Key: "callCount"})
			countCh := make(chan messages.PluginStateGetResp, 1)
			unsub, _ := sdk.SubscribeTo[messages.PluginStateGetResp](rt, ctx, getResult.ReplyTo, func(r messages.PluginStateGetResp, m messages.Message) { countCh <- r })
			var countResp messages.PluginStateGetResp
			select {
			case countResp = <-countCh:
			case <-ctx.Done():
			}
			unsub()
			count := 0
			if countResp.Value != "" {
				fmt.Sscanf(countResp.Value, "%d", &count)
			}
			count++
			sdk.Publish(rt, ctx, messages.PluginStateSetMsg{
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
		listResult, err := sdk.Publish(rt, context.Background(), messages.ToolListMsg{})
		if err != nil {
			log.Printf("[testplugin] failed to publish tool list: %v", err)
		} else {
			toolsCh := make(chan messages.ToolListResp, 1)
			unsub, _ := sdk.SubscribeTo[messages.ToolListResp](rt, context.Background(), listResult.ReplyTo, func(r messages.ToolListResp, m messages.Message) { toolsCh <- r })
			select {
			case resp := <-toolsCh:
				log.Printf("[testplugin] host has %d tools", len(resp.Tools))
			case <-context.Background().Done():
			}
			unsub()
		}
		return nil
	})

	if err := plugin.Run(); err != nil {
		log.Fatalf("[testplugin] fatal: %v", err)
	}
}
