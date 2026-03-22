package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// PublishAwaitTo sends a typed command to a specific Kit's namespace and blocks until the result.
func PublishAwaitTo[Req, Resp messages.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, req Req) (Resp, error) {
	var resp Resp
	xrt, ok := rt.(CrossNamespaceRuntime)
	if !ok {
		return resp, fmt.Errorf("runtime does not support cross-namespace operations")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return resp, fmt.Errorf("marshal %T: %w", req, err)
	}

	resultTopic := resp.BusTopic()
	resultCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan messages.Message, 1)
	stop, err := xrt.SubscribeRawTo(resultCtx, targetNamespace, resultTopic, func(msg messages.Message) {
		select {
		case resultCh <- msg:
		default:
		}
	})
	if err != nil {
		return resp, err
	}
	defer stop()

	correlationID, err := xrt.PublishRawTo(ctx, targetNamespace, req.BusTopic(), payload)
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

// PublishTo sends a typed fire-and-forget message to a specific Kit's namespace.
func PublishTo[T messages.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, msg T) (string, error) {
	xrt, ok := rt.(CrossNamespaceRuntime)
	if !ok {
		return "", fmt.Errorf("runtime does not support cross-namespace operations")
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal %T: %w", msg, err)
	}
	return xrt.PublishRawTo(ctx, targetNamespace, msg.BusTopic(), payload)
}

// SubscribeTo listens for typed messages on a topic in a specific Kit's namespace.
func SubscribeTo[T messages.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, handler func(T, messages.Message)) (func(), error) {
	xrt, ok := rt.(CrossNamespaceRuntime)
	if !ok {
		return nil, fmt.Errorf("runtime does not support cross-namespace operations")
	}
	var zero T
	return xrt.SubscribeRawTo(ctx, zero.BusTopic(), targetNamespace, func(msg messages.Message) {
		var typed T
		if err := json.Unmarshal(msg.Payload, &typed); err != nil {
			return
		}
		handler(typed, msg)
	})
}
