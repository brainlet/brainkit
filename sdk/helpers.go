package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// PublishAwait sends a typed command and blocks until the correlated result arrives.
// Internally: SubscribeRaw (waits until active) → PublishRaw → filter by correlationID.
func PublishAwait[Req, Resp messages.BrainkitMessage](rt Runtime, ctx context.Context, req Req) (Resp, error) {
	var resp Resp
	payload, err := json.Marshal(req)
	if err != nil {
		return resp, fmt.Errorf("marshal %T: %w", req, err)
	}

	resultTopic := resp.BusTopic()
	resultCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan messages.Message, 1)
	stop, err := rt.SubscribeRaw(resultCtx, resultTopic, func(msg messages.Message) {
		select {
		case resultCh <- msg:
		default:
		}
	})
	if err != nil {
		return resp, err
	}
	defer stop()

	// Subscribe is active (contract guarantee). Now publish.
	correlationID, err := rt.PublishRaw(ctx, req.BusTopic(), payload)
	if err != nil {
		return resp, err
	}

	for {
		select {
		case <-ctx.Done():
			return resp, ctx.Err()
		case msg := <-resultCh:
			if msg.Metadata["correlationId"] != correlationID {
				continue
			}
			cancel()
			if err := json.Unmarshal(msg.Payload, &resp); err != nil {
				return resp, fmt.Errorf("unmarshal %T: %w", resp, err)
			}
			if errMsg := messages.ResultErrorOf(resp); errMsg != "" {
				return resp, fmt.Errorf("%s", errMsg)
			}
			return resp, nil
		}
	}
}

// Publish sends a typed fire-and-forget message. Returns correlationID.
func Publish[T messages.BrainkitMessage](rt Runtime, ctx context.Context, msg T) (string, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal %T: %w", msg, err)
	}
	return rt.PublishRaw(ctx, msg.BusTopic(), payload)
}

// Subscribe listens for typed messages on the message's topic.
// Handler receives the typed message and the raw Message for metadata access (correlationID, callerID, etc).
func Subscribe[T messages.BrainkitMessage](rt Runtime, ctx context.Context, handler func(T, messages.Message)) (func(), error) {
	var zero T
	return rt.SubscribeRaw(ctx, zero.BusTopic(), func(msg messages.Message) {
		var typed T
		if err := json.Unmarshal(msg.Payload, &typed); err != nil {
			return
		}
		handler(typed, msg)
	})
}
