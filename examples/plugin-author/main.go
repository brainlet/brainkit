// Command plugin-author is a minimal brainkit plugin. It registers
// one tool (`echo`) and one event subscription (`demo.events`),
// then connects back to its host Kit over WebSocket.
//
// Plugins build as standalone Go binaries; a running brainkit host
// launches them via the modules/plugins supervisor. Because plugins
// live outside the brainkit module, this directory carries its own
// go.mod. Run from the repo root with:
//
//	cd examples/plugin-author && go mod tidy && go build .
//
// Then point a host Kit at the resulting binary:
//
//	plugins.NewModule(plugins.Config{
//	    Plugins: []brainkit.PluginConfig{{
//	        Name:   "demo",
//	        Binary: "./examples/plugin-author/plugin-author",
//	    }},
//	})
package main

import (
	"context"
	"encoding/json"
	"log"

	bkplugin "github.com/brainlet/brainkit/sdk/plugin"
)

type EchoIn struct {
	Text string `json:"text"`
}

type EchoOut struct {
	Echoed string `json:"echoed"`
}

func main() {
	p := bkplugin.New("brainlet", "plugin-author", "0.1.0",
		bkplugin.WithDescription("Minimal brainkit plugin example"))

	bkplugin.Tool(p, "echo", "Echo the input text back.",
		func(_ context.Context, _ bkplugin.Client, in EchoIn) (EchoOut, error) {
			return EchoOut{Echoed: in.Text}, nil
		})

	bkplugin.On[json.RawMessage](p, "demo.events",
		func(_ context.Context, payload json.RawMessage, _ bkplugin.Client) {
			log.Printf("received demo.events: %s", string(payload))
		})

	if err := p.Run(); err != nil {
		log.Fatalf("plugin run: %v", err)
	}
}
