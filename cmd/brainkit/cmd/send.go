package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newSendCmd — package-shorthand streaming client. Publishes to
// `ts.<pkg>.<topic>` and streams every event the handler emits. The
// type-aware renderer knows how to format `msg.stream.text` /
// `progress` / `object` / `event` / `error` and the terminal
// `msg.reply`.
//
//	brainkit send hello greet '{"name":"David"}'
//	brainkit send support-team ask '{"query":"help"}'
//
// Differs from `brainkit call`:
//
//   - `call` takes a raw bus topic (`ts.math.ask`) and does a
//     one-shot request/reply.
//   - `send` takes package + local topic, hits `/api/stream`, and
//     renders streaming chunks as they arrive.
func newSendCmd() *cobra.Command {
	var endpoint string
	var format string

	c := &cobra.Command{
		Use:   "send <package> <topic> [payload]",
		Short: "Publish to a deployed package and stream responses",
		Long: `Send publishes to a deployed package's bus topic and streams
every event — intermediate chunks via msg.stream.* and the terminal
reply — rendering each one according to its shape (text tokens print
inline, progress prints as "[NN%] message", objects pretty-print as
JSON, etc).

Topic composition follows the deployment convention:
  package + localTopic  →  ts.<package>.<localTopic>

Examples:
  brainkit send hello greet '{"name":"David"}'
  brainkit send support-team ask '{"query":"help"}'

For a one-shot request/reply on a raw topic, use "brainkit call"
instead.`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, topic := args[0], args[1]
			if pkg == "" || topic == "" {
				return fmt.Errorf("package and topic are required")
			}
			payload := json.RawMessage(`null`)
			if len(args) > 2 && args[2] != "" {
				payload = json.RawMessage(args[2])
			}

			fullTopic := "ts." + pkg + "." + topic

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			client := newBusClient(endpoint)
			render := renderEventAuto
			switch format {
			case "auto":
				render = renderEventAuto
			case "json":
				render = renderEventJSON
			case "text":
				render = renderEventText
			default:
				return fmt.Errorf("invalid --format %q (want auto, json, or text)", format)
			}

			if jsonOutput {
				render = renderEventJSON
			}

			_, err := client.stream(ctx, fullTopic, payload, func(raw json.RawMessage) {
				render(cmd, raw)
			})
			return err
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	c.Flags().StringVar(&format, "format", "auto", "output format: auto (shape-aware), json (raw per-event), text (text chunks only)")
	return c
}

// ── Event renderers ─────────────────────────────────────────────

// typedStreamEvent matches the shape produced by `msg.stream.text`,
// `msg.stream.progress`, etc. Any payload without a matching `type`
// field falls back to JSON pretty-print.
type typedStreamEvent struct {
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
	Event string          `json:"event,omitempty"`
	Seq   int             `json:"seq,omitempty"`
}

// renderEventAuto inspects each event for a `type` field and
// formats it accordingly. Mirrors the wire format defined in
// internal/engine/runtime/bus.js.
func renderEventAuto(cmd *cobra.Command, raw json.RawMessage) {
	var evt typedStreamEvent
	if err := json.Unmarshal(raw, &evt); err == nil && evt.Type != "" {
		switch evt.Type {
		case "text":
			var s string
			if json.Unmarshal(evt.Data, &s) == nil {
				fmt.Fprint(cmd.OutOrStdout(), s)
				return
			}
		case "progress":
			var prog struct {
				Value   float64 `json:"value"`
				Message string  `json:"message"`
			}
			if json.Unmarshal(evt.Data, &prog) == nil {
				if prog.Message != "" {
					cmd.Printf("[%.0f%%] %s\n", prog.Value*100, prog.Message)
				} else {
					cmd.Printf("[%.0f%%]\n", prog.Value*100)
				}
				return
			}
		case "object":
			renderPrettyJSON(cmd, evt.Data)
			return
		case "event":
			name := evt.Event
			if name == "" {
				name = "event"
			}
			cmd.Printf("[%s] %s\n", name, string(evt.Data))
			return
		case "error":
			var errData struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(evt.Data, &errData)
			if errData.Message == "" {
				errData.Message = string(evt.Data)
			}
			fmt.Fprintln(os.Stderr, "Error:", errData.Message)
			return
		case "end":
			if len(evt.Data) > 0 && string(evt.Data) != "null" {
				renderPrettyJSON(cmd, evt.Data)
			}
			return
		}
	}
	renderPrettyJSON(cmd, raw)
}

// renderEventJSON emits every event as a single compact JSON line —
// machine-friendly.
func renderEventJSON(cmd *cobra.Command, raw json.RawMessage) {
	cmd.Println(string(raw))
}

// renderEventText keeps only `text` chunks (the token stream) and
// drops every other event type. Useful for piping an LLM response
// into another process without the progress/event framing noise.
func renderEventText(cmd *cobra.Command, raw json.RawMessage) {
	var evt typedStreamEvent
	if err := json.Unmarshal(raw, &evt); err == nil && evt.Type == "text" {
		var s string
		if json.Unmarshal(evt.Data, &s) == nil {
			fmt.Fprint(cmd.OutOrStdout(), s)
		}
	}
}

// renderPrettyJSON indents + prints a JSON payload. Falls back to
// the raw bytes when the payload isn't valid JSON.
func renderPrettyJSON(cmd *cobra.Command, data json.RawMessage) {
	var v any
	if err := json.Unmarshal(data, &v); err == nil {
		if formatted, err := json.MarshalIndent(v, "", "  "); err == nil {
			cmd.Println(string(formatted))
			return
		}
	}
	cmd.Println(string(data))
}
