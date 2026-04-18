package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// newCallCmd creates the `brainkit call` verb — a generic HTTP
// client over the gateway's POST /api/bus endpoint. Reads a JSON
// payload from stdin (default), an inline --payload flag, or a
// file via --payload-file, and prints the reply to stdout.
func newCallCmd() *cobra.Command {
	var (
		endpoint    string
		payloadFlag string
		payloadFile string
		stream      bool
	)
	c := &cobra.Command{
		Use:   "call <topic>",
		Short: "Issue a bus request to a running brainkit server",
		Long: `Call publishes a message on the given bus topic through the
server's gateway and prints the reply as JSON.

The payload comes from (in priority order): --payload, --payload-file,
or stdin. Use --stream for NDJSON-streamed replies; each event is
printed on its own line.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := args[0]

			payload, err := readPayload(cmd, payloadFlag, payloadFile)
			if err != nil {
				return err
			}

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			client := newBusClient(endpoint)

			if stream {
				enc := json.NewEncoder(cmd.OutOrStdout())
				_, err := client.stream(ctx, topic, payload, func(raw json.RawMessage) {
					_ = enc.Encode(map[string]any{"event": json.RawMessage(raw)})
				})
				return err
			}

			reply, err := client.call(ctx, topic, payload)
			if err != nil {
				return err
			}

			if len(reply) == 0 {
				return nil
			}
			return writeJSONPretty(cmd.OutOrStdout(), reply)
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	c.Flags().StringVar(&payloadFlag, "payload", "", "inline JSON payload")
	c.Flags().StringVar(&payloadFile, "payload-file", "", "path to JSON payload file (use - for stdin)")
	c.Flags().BoolVar(&stream, "stream", false, "consume NDJSON stream from POST /api/stream")
	return c
}

// readPayload returns the JSON payload for a bus call. Priority:
// --payload flag, --payload-file, stdin. Empty input becomes `{}`.
func readPayload(cmd *cobra.Command, inline, file string) (json.RawMessage, error) {
	if inline != "" {
		if !json.Valid([]byte(inline)) {
			return nil, fmt.Errorf("--payload is not valid JSON")
		}
		return json.RawMessage(inline), nil
	}

	var reader io.Reader
	switch {
	case file == "" || file == "-":
		// stdin only when it's a pipe / redirected, not an
		// interactive terminal (we don't want to hang).
		if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			reader = cmd.InOrStdin()
		}
	case file != "":
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("open %q: %w", file, err)
		}
		defer f.Close()
		reader = f
	}

	if reader == nil {
		return json.RawMessage("{}"), nil
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	raw = []byte(trimSpace(string(raw)))
	if len(raw) == 0 {
		return json.RawMessage("{}"), nil
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("payload is not valid JSON")
	}
	return json.RawMessage(raw), nil
}

// writeJSONPretty re-encodes payload with 2-space indent to make
// the output human-friendly without sacrificing JSON shape.
func writeJSONPretty(w io.Writer, payload json.RawMessage) error {
	var pretty any
	if err := json.Unmarshal(payload, &pretty); err != nil {
		_, _ = w.Write(payload)
		_, _ = w.Write([]byte{'\n'})
		return nil
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(pretty)
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
