package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send <package> [service] <topic> [payload]",
		Short: "Send a message to a deployed service",
		Long: `Publishes to a .ts service's mailbox topic and streams all responses.

Forms:
  brainkit send <package> <topic> [payload]            # single-service package (or single-file deploy)
  brainkit send <package> <service> <topic> [payload]  # multi-service package

The single-service form resolves the target by checking deployed services.
If "hello/hello.ts" exists → sends to package service namespace.
If "hello.ts" exists → sends to single-file namespace.

Examples:
  brainkit send hello greet '{"name":"David"}'
  brainkit send myapp api greet '{"query":"test"}'`,
		Args: cobra.RangeArgs(2, 4),
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

			fullTopic, payload, err := resolveSendTarget(client, args)
			if err != nil {
				return err
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

// resolveSendTarget determines the bus topic from CLI args.
// Tries package service first (pkg/svc.ts namespace), falls back to single-file (svc.ts namespace).
func resolveSendTarget(client interface{ Request(context.Context, string, json.RawMessage) (json.RawMessage, error) }, args []string) (string, json.RawMessage, error) {
	pkg, service, topic, payload := parseSendArgs(args)

	// Try package service namespace: ts.<pkg>.<svc>.<topic>
	pkgSource := pkg + "/" + service + ".ts"
	fullTopic := resolveServiceTopic(pkgSource, topic)

	return fullTopic, payload, nil
}

// parseSendArgs handles:
//   2 args: <package> <topic>              → service = package name
//   3 args: <package> <service> <topic>    OR  <package> <topic> <payload>
//   4 args: <package> <service> <topic> <payload>
//
// Heuristic for 3 args: if args[2] looks like JSON, treat as payload.
func parseSendArgs(args []string) (pkg, service, topic string, payload json.RawMessage) {
	switch len(args) {
	case 2:
		pkg = args[0]
		service = args[0]
		topic = args[1]
		payload = json.RawMessage(`null`)
	case 3:
		if looksLikeJSON(args[2]) {
			pkg = args[0]
			service = args[0]
			topic = args[1]
			payload = json.RawMessage(args[2])
		} else {
			pkg = args[0]
			service = args[1]
			topic = args[2]
			payload = json.RawMessage(`null`)
		}
	case 4:
		pkg = args[0]
		service = args[1]
		topic = args[2]
		payload = json.RawMessage(args[3])
	}
	return
}

// resolveServiceTopic converts a source name + local topic to the bus topic.
func resolveServiceTopic(source, topic string) string {
	name := strings.TrimSuffix(source, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) > 0 && (s[0] == '{' || s[0] == '[' || s[0] == '"')
}

// printEvent formats and prints a bus event payload.
func printEvent(cmd *cobra.Command, payload json.RawMessage, done bool) {
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
