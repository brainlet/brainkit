package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send <service> <topic> [payload]",
		Short: "Send a message to a deployed service and show all responses",
		Long: `Publishes to a .ts service's mailbox topic and streams all events.
Shows intermediate events (msg.send, msg.stream.*) as they arrive.
Stops when the service sends a terminal event (msg.reply, msg.stream.end).

Example: brainkit send hello greet '{"name":"David"}'`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			client, err := config.Connect(cfg)
			if err != nil {
				return err
			}
			defer client.Close()

			service := args[0]
			topic := args[1]
			fullTopic := "ts." + service + "." + topic

			var payload json.RawMessage
			if len(args) > 2 {
				payload = json.RawMessage(args[2])
			} else {
				payload = json.RawMessage(`null`)
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			events, err := client.Stream(ctx, fullTopic, payload)
			if err != nil {
				return err
			}

			for evt := range events {
				if evt.Error != "" {
					return fmt.Errorf("%s", evt.Error)
				}
				printEvent(cmd, evt.Payload, evt.Done)
			}
			return nil
		},
	}
}

// printEvent formats and prints a bus event payload.
func printEvent(cmd *cobra.Command, payload json.RawMessage, done bool) {
	// Try typed stream event (has "type" field)
	var typed struct {
		Type  string          `json:"type"`
		Data  json.RawMessage `json:"data"`
		Event string          `json:"event,omitempty"`
	}
	if json.Unmarshal(payload, &typed) == nil && typed.Type != "" {
		switch typed.Type {
		case "text":
			var text string
			json.Unmarshal(typed.Data, &text)
			fmt.Fprint(cmd.OutOrStdout(), text)
			return
		case "progress":
			var prog struct {
				Value   float64 `json:"value"`
				Message string  `json:"message"`
			}
			json.Unmarshal(typed.Data, &prog)
			if prog.Message != "" {
				cmd.Printf("[%.0f%%] %s\n", prog.Value*100, prog.Message)
			}
			return
		case "object":
			printPrettyJSON(cmd, typed.Data)
			return
		case "event":
			name := typed.Event
			if name == "" {
				name = "event"
			}
			cmd.Printf("[%s] %s\n", name, string(typed.Data))
			return
		case "error":
			var errData struct {
				Message string `json:"message"`
			}
			json.Unmarshal(typed.Data, &errData)
			cmd.PrintErrln("Error:", errData.Message)
			return
		case "end":
			if len(typed.Data) > 0 && string(typed.Data) != "null" {
				printPrettyJSON(cmd, typed.Data)
			}
			return
		}
	}

	// Untyped payload — print as pretty JSON
	printPrettyJSON(cmd, payload)
}

func printPrettyJSON(cmd *cobra.Command, data json.RawMessage) {
	var pretty any
	if json.Unmarshal(data, &pretty) == nil {
		formatted, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			cmd.Println(string(formatted))
			return
		}
	}
	cmd.Println(string(data))
}
