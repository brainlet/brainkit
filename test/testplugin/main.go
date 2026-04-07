// testplugin is a minimal brainkit plugin for e2e testing.
// It registers an "echo" tool and a "concat" tool, and uses plugin state.
package main

import (
	"context"
	"fmt"
	"log"

	bkplugin "github.com/brainlet/brainkit/plugin"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

type EchoInput struct {
	Message string `json:"message"`
}

type EchoOutput struct {
	Echoed string `json:"echoed"`
	Plugin string `json:"plugin"`
}

type ConcatInput struct {
	A string `json:"a"`
	B string `json:"b"`
}

type ConcatOutput struct {
	Result string `json:"result"`
}

func main() {
	p := bkplugin.New("test", "testplugin", "1.0.0", bkplugin.WithDescription("Test plugin for e2e"))

	bkplugin.Tool[EchoInput, EchoOutput](p, "echo", "echoes the message with plugin stamp",
		func(ctx context.Context, rt bkplugin.Client, in EchoInput) (EchoOutput, error) {
			return EchoOutput{
				Echoed: in.Message,
				Plugin: "testplugin",
			}, nil
		})

	bkplugin.Tool[ConcatInput, ConcatOutput](p, "concat", "concatenates two strings",
		func(ctx context.Context, rt bkplugin.Client, in ConcatInput) (ConcatOutput, error) {
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
			return ConcatOutput{Result: in.A + in.B}, nil
		})

	p.OnStart(func(rt bkplugin.Client) error {
		log.Println("[testplugin] started successfully")
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

	if err := p.Run(); err != nil {
		log.Fatalf("[testplugin] fatal: %v", err)
	}
}
