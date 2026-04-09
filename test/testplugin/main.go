// testplugin is a minimal brainkit plugin for e2e testing.
// It registers an "echo" tool and a "concat" tool.
package main

import (
	"context"
	"log"

	bkplugin "github.com/brainlet/brainkit/plugin"
	"github.com/brainlet/brainkit/sdk"
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
			return ConcatOutput{Result: in.A + in.B}, nil
		})

	p.OnStart(func(rt bkplugin.Client) error {
		log.Println("[testplugin] started successfully")
		listResult, err := sdk.Publish(rt, context.Background(), sdk.ToolListMsg{})
		if err != nil {
			log.Printf("[testplugin] failed to publish tool list: %v", err)
		} else {
			toolsCh := make(chan sdk.ToolListResp, 1)
			unsub, _ := sdk.SubscribeTo[sdk.ToolListResp](rt, context.Background(), listResult.ReplyTo, func(r sdk.ToolListResp, m sdk.Message) { toolsCh <- r })
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
