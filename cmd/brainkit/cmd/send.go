package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send <service> <topic> [payload]",
	Short: "Send a message to a deployed service and wait for the reply",
	Long: `Publishes to a .ts service's mailbox topic and waits for the response.
Example: brainkit send hello greet '{"name":"David"}'`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		topic := args[1]

		// Build the full bus topic: ts.<service>.<topic>
		fullTopic := "ts." + service + "." + topic

		var payload json.RawMessage
		if len(args) > 2 {
			payload = json.RawMessage(args[2])
		} else {
			payload = json.RawMessage(`null`)
		}

		return connectAndPublish(
			messages.KitSendMsg{Topic: fullTopic, Payload: payload},
			func(resp *messages.KitSendResp) {
				// Pretty-print if it's valid JSON
				var pretty json.RawMessage
				if json.Unmarshal(resp.Payload, &pretty) == nil {
					formatted, err := json.MarshalIndent(pretty, "", "  ")
					if err == nil {
						fmt.Println(string(formatted))
						return
					}
				}
				fmt.Println(string(resp.Payload))
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)
}
