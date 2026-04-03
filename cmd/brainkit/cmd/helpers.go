package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

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

	resp, err := busRequest[Req, Resp](client, req)
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

func busRequest[Req messages.BrainkitMessage, Resp any](rt sdk.Runtime, req Req) (*Resp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, req)
	if err != nil {
		return nil, err
	}

	replyCh := make(chan messages.Message, 1)
	unsub, err := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		select {
		case replyCh <- msg:
		default:
		}
	})
	if err != nil {
		return nil, err
	}
	defer unsub()

	select {
	case msg := <-replyCh:
		var resp Resp
		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		if errMsg := messages.ResultErrorOf(resp); errMsg != "" {
			return nil, fmt.Errorf("%s", errMsg)
		}
		return &resp, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("no response within %s. Is `brainkit start` running?", timeout)
	}
}

// w is a shorthand to get the command's output writer for tabwriter etc.
func w(cmd *cobra.Command) io.Writer {
	return cmd.OutOrStdout()
}
