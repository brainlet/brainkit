// testplugin is a minimal brainkit plugin for e2e testing.
// It registers an "echo" tool and a "concat" tool.
package main

import (
	"context"
	"log"
	"time"

	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	bkplugin "github.com/brainlet/brainkit/sdk/plugin"
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := sdk.Call[toolmsg.ToolListMsg, toolmsg.ToolListResp](rt, ctx, toolmsg.ToolListMsg{}, sdk.WithCallNoCancelSignal())
		if err != nil {
			log.Printf("[testplugin] failed to list host tools: %v", err)
		} else {
			log.Printf("[testplugin] host has %d tools", len(resp.Tools))
		}
		return nil
	})

	if err := p.Run(); err != nil {
		log.Fatalf("[testplugin] fatal: %v", err)
	}
}
