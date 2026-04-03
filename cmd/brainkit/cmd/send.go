package cmd

import (
	"encoding/json"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send <service> <topic> [payload]",
		Short: "Send a message to a deployed service and wait for the reply",
		Long: `Publishes to a .ts service's mailbox topic and waits for the response.
Example: brainkit send hello greet '{"name":"David"}'`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			service := args[0]
			topic := args[1]
			fullTopic := "ts." + service + "." + topic

			var payload json.RawMessage
			if len(args) > 2 {
				payload = json.RawMessage(args[2])
			} else {
				payload = json.RawMessage(`null`)
			}

			return connectAndPublish(cmd, messages.KitSendMsg{Topic: fullTopic, Payload: payload},
				func(resp *messages.KitSendResp) {
					var pretty json.RawMessage
					if json.Unmarshal(resp.Payload, &pretty) == nil {
						formatted, err := json.MarshalIndent(pretty, "", "  ")
						if err == nil {
							cmd.Println(string(formatted))
							return
						}
					}
					cmd.Println(string(resp.Payload))
				},
			)
		},
	}
}
