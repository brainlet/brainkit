package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

// connectAndPublish connects to the running instance via HTTP, sends a typed
// bus command, waits for the response, and calls format to print it.
func connectAndPublish[Req messages.BrainkitMessage, Resp any](cmd *cobra.Command, req Req, format func(*Resp)) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	client, err := config.Connect(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := httpBusRequest[Req, Resp](client, req)
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	format(resp)
	return nil
}

// httpBusRequest sends a typed bus command over HTTP and returns the typed response.
func httpBusRequest[Req messages.BrainkitMessage, Resp any](client *brainkit.BusClient, req Req) (*Resp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respPayload, err := client.Request(ctx, req.BusTopic(), payload)
	if err != nil {
		return nil, err
	}

	var resp Resp
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if errMsg := messages.ResultErrorOf(resp); errMsg != "" {
		return nil, fmt.Errorf("%s", errMsg)
	}
	return &resp, nil
}

// w is a shorthand to get the command's output writer for tabwriter etc.
func w(cmd *cobra.Command) io.Writer {
	return cmd.OutOrStdout()
}
