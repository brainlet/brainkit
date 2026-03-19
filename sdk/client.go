package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/brainlet/brainkit/sdk/messages"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
)

// Client is the Kit API available to plugins via gRPC.
// All methods route through the bus as async messages.
type Client interface {
	Send(ctx context.Context, msg messages.BusMessage) error
	Ask(ctx context.Context, msg messages.BusMessage, callback func(messages.Message)) (cancel func())
	On(ctx context.Context, pattern string, handler func(messages.Message, messages.ReplyFunc)) (cancel func())
	GetState(ctx context.Context, key string) (string, error)
	SetState(ctx context.Context, key string, value string) error
}

// ReplyFunc is exported as a convenience alias for sdk.ReplyFunc in handler signatures.
type ReplyFunc = messages.ReplyFunc

// Ask sends a typed message and invokes the callback with the typed response.
func Ask[Resp any](client Client, ctx context.Context, msg messages.BusMessage, callback func(Resp, error)) (cancel func()) {
	return client.Ask(ctx, msg, func(raw messages.Message) {
		var resp Resp
		if len(raw.Payload) > 0 {
			var errResp struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(raw.Payload, &errResp) == nil && errResp.Error != "" {
				callback(resp, fmt.Errorf("%s", errResp.Error))
				return
			}
			if err := json.Unmarshal(raw.Payload, &resp); err != nil {
				callback(resp, fmt.Errorf("sdk: unmarshal response: %w", err))
				return
			}
		}
		callback(resp, nil)
	})
}

// --- gRPC Implementation ---

type grpcClient struct {
	safeSend func(*pluginv1.PluginMessage) error
	timeout  time.Duration

	replyMu sync.Mutex
	replies map[string]func(*pluginv1.PluginMessage)

	dynSubMu sync.Mutex
	dynSubs  map[string]func(messages.Message, messages.ReplyFunc)
}

func newGRPCClient(safeSend func(*pluginv1.PluginMessage) error) *grpcClient {
	return &grpcClient{
		safeSend: safeSend,
		timeout:  30 * time.Second,
		replies:  make(map[string]func(*pluginv1.PluginMessage)),
		dynSubs:  make(map[string]func(messages.Message, messages.ReplyFunc)),
	}
}

func (c *grpcClient) Send(ctx context.Context, msg messages.BusMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("sdk: marshal: %w", err)
	}
	return c.safeSend(&pluginv1.PluginMessage{
		Id:      uuid.NewString(),
		Type:    "send",
		Topic:   msg.BusTopic(),
		Payload: payload,
	})
}

func (c *grpcClient) Ask(ctx context.Context, msg messages.BusMessage, callback func(messages.Message)) (cancel func()) {
	payload, err := json.Marshal(msg)
	if err != nil {
		go callback(messages.Message{})
		return func() {}
	}

	replyTo := "_plugin_reply." + uuid.NewString()
	cancelled := make(chan struct{})
	var once sync.Once

	c.replyMu.Lock()
	c.replies[replyTo] = func(reply *pluginv1.PluginMessage) {
		callback(messages.Message{
			Topic:    reply.Topic,
			Payload:  reply.Payload,
			CallerID: reply.CallerId,
			TraceID:  reply.TraceId,
			Metadata: reply.Metadata,
		})
	}
	c.replyMu.Unlock()

	sendErr := c.safeSend(&pluginv1.PluginMessage{
		Id:      uuid.NewString(),
		Type:    "ask",
		Topic:   msg.BusTopic(),
		ReplyTo: replyTo,
		Payload: payload,
	})

	if sendErr != nil {
		c.replyMu.Lock()
		delete(c.replies, replyTo)
		c.replyMu.Unlock()
		go callback(messages.Message{})
		return func() {}
	}

	go func() {
		select {
		case <-time.After(c.timeout):
			c.replyMu.Lock()
			_, pending := c.replies[replyTo]
			if pending {
				delete(c.replies, replyTo)
			}
			c.replyMu.Unlock()
			if pending {
				errPayload, _ := json.Marshal(map[string]string{
					"error": fmt.Sprintf("timeout after %s for topic %q", c.timeout, msg.BusTopic()),
				})
				callback(messages.Message{
					Topic:   replyTo,
					Payload: errPayload,
				})
			}
		case <-ctx.Done():
			c.replyMu.Lock()
			delete(c.replies, replyTo)
			c.replyMu.Unlock()
		case <-cancelled:
		}
	}()

	return func() {
		once.Do(func() {
			close(cancelled)
			c.replyMu.Lock()
			delete(c.replies, replyTo)
			c.replyMu.Unlock()
		})
	}
}

func (c *grpcClient) On(ctx context.Context, pattern string, handler func(messages.Message, messages.ReplyFunc)) (cancel func()) {
	subID := uuid.NewString()
	key := subID + ":" + pattern

	c.dynSubMu.Lock()
	c.dynSubs[key] = handler
	c.dynSubMu.Unlock()

	c.safeSend(&pluginv1.PluginMessage{
		Id:    uuid.NewString(),
		Type:  "subscribe",
		Topic: pattern,
	})

	return func() {
		c.dynSubMu.Lock()
		delete(c.dynSubs, key)
		c.dynSubMu.Unlock()

		c.safeSend(&pluginv1.PluginMessage{
			Id:    uuid.NewString(),
			Type:  "unsubscribe",
			Topic: pattern,
		})
	}
}

func (c *grpcClient) GetState(ctx context.Context, key string) (string, error) {
	var result string
	var resultErr error
	done := make(chan struct{})

	Ask[stateGetResp](c, ctx, stateGetMsg{Key: key}, func(resp stateGetResp, err error) {
		result = resp.Value
		resultErr = err
		close(done)
	})

	select {
	case <-done:
		return result, resultErr
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *grpcClient) SetState(ctx context.Context, key, value string) error {
	var resultErr error
	done := make(chan struct{})

	Ask[stateSetResp](c, ctx, stateSetMsg{Key: key, Value: value}, func(_ stateSetResp, err error) {
		resultErr = err
		close(done)
	})

	select {
	case <-done:
		return resultErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *grpcClient) handleReply(msg *pluginv1.PluginMessage) {
	c.replyMu.Lock()
	handler, ok := c.replies[msg.ReplyTo]
	if ok {
		delete(c.replies, msg.ReplyTo)
	}
	c.replyMu.Unlock()

	if ok && handler != nil {
		go handler(msg)
	}
}

// --- Internal state messages ---

type stateGetMsg struct {
	Key string `json:"key"`
}

func (stateGetMsg) BusTopic() string { return "plugin.state.get" }

type stateGetResp struct {
	Value string `json:"value"`
}

type stateSetMsg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (stateSetMsg) BusTopic() string { return "plugin.state.set" }

type stateSetResp struct{}
