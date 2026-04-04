package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/spf13/cobra"
)

func newStreamCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stream <service> <topic> [payload]",
		Short: "Stream events from a deployed service",
		Long: `Publishes to a .ts service's mailbox and streams all events until completion.
Intermediate events (msg.send, msg.stream.text, etc.) are printed as they arrive.
The stream ends when the service calls msg.reply or msg.stream.end.

Example: brainkit stream hello greet '{"name":"David"}'`,
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

				// Try to parse as a typed stream event (has "type" field)
				var typed struct {
					Type  string          `json:"type"`
					Data  json.RawMessage `json:"data"`
					Event string          `json:"event,omitempty"`
				}
				if json.Unmarshal(evt.Payload, &typed) == nil && typed.Type != "" {
					switch typed.Type {
					case "text":
						var text string
						json.Unmarshal(typed.Data, &text)
						fmt.Fprint(cmd.OutOrStdout(), text)
					case "progress":
						var prog struct {
							Value   float64 `json:"value"`
							Message string  `json:"message"`
						}
						json.Unmarshal(typed.Data, &prog)
						if prog.Message != "" {
							cmd.Printf("[%.0f%%] %s\n", prog.Value*100, prog.Message)
						}
					case "object":
						formatted, _ := json.MarshalIndent(typed.Data, "", "  ")
						cmd.Println(string(formatted))
					case "event":
						name := typed.Event
						if name == "" {
							name = "event"
						}
						cmd.Printf("[%s] %s\n", name, string(typed.Data))
					case "error":
						var errData struct {
							Message string `json:"message"`
						}
						json.Unmarshal(typed.Data, &errData)
						return fmt.Errorf("%s", errData.Message)
					case "end":
						if len(typed.Data) > 0 && string(typed.Data) != "null" {
							formatted, _ := json.MarshalIndent(typed.Data, "", "  ")
							cmd.Println(string(formatted))
						}
					}
				} else if evt.Done {
					// Untyped terminal — print raw payload
					var pretty json.RawMessage
					if json.Unmarshal(evt.Payload, &pretty) == nil {
						formatted, _ := json.MarshalIndent(pretty, "", "  ")
						cmd.Println(string(formatted))
					} else {
						cmd.Println(string(evt.Payload))
					}
				} else {
					// Untyped intermediate — print raw
					cmd.Println(string(evt.Payload))
				}
			}

			fmt.Fprintln(cmd.OutOrStdout()) // newline after streaming text
			return nil
		},
	}
}
