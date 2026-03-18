package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
)

// brainletClient implements BrainletClient over a gRPC stream.
type brainletClient struct {
	safeSend func(*pluginv1.PluginMessage) error // thread-safe sender

	replyMu sync.Mutex
	replies map[string]chan *pluginv1.PluginMessage
	timeout time.Duration
}

func newBrainletClient(safeSend func(*pluginv1.PluginMessage) error) *brainletClient {
	return &brainletClient{
		safeSend: safeSend,
		replies:  make(map[string]chan *pluginv1.PluginMessage),
		timeout:  30 * time.Second,
	}
}

// handleReply routes a bus.ask.reply message to the waiting Ask caller.
func (c *brainletClient) handleReply(msg *pluginv1.PluginMessage) {
	c.replyMu.Lock()
	ch, ok := c.replies[msg.ReplyTo]
	if ok {
		delete(c.replies, msg.ReplyTo)
	}
	c.replyMu.Unlock()

	if ok && ch != nil {
		ch <- msg
	}
}

func (c *brainletClient) Send(ctx context.Context, topic string, payload json.RawMessage) error {
	return c.safeSend(&pluginv1.PluginMessage{
		Id:      uuid.NewString(),
		Type:    "bus.send",
		Topic:   topic,
		Payload: payload,
	})
}

func (c *brainletClient) Ask(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error) {
	replyTo := "_plugin_reply." + uuid.NewString()
	ch := make(chan *pluginv1.PluginMessage, 1)

	c.replyMu.Lock()
	c.replies[replyTo] = ch
	c.replyMu.Unlock()

	err := c.safeSend(&pluginv1.PluginMessage{
		Id:      uuid.NewString(),
		Type:    "bus.ask",
		Topic:   topic,
		ReplyTo: replyTo,
		Payload: payload,
	})
	if err != nil {
		c.replyMu.Lock()
		delete(c.replies, replyTo)
		c.replyMu.Unlock()
		return nil, err
	}

	select {
	case reply := <-ch:
		return reply.Payload, nil
	case <-ctx.Done():
		c.replyMu.Lock()
		delete(c.replies, replyTo)
		c.replyMu.Unlock()
		return nil, ctx.Err()
	case <-time.After(c.timeout):
		c.replyMu.Lock()
		delete(c.replies, replyTo)
		c.replyMu.Unlock()
		return nil, fmt.Errorf("sdk: ask timeout for %q", topic)
	}
}

func (c *brainletClient) CallTool(ctx context.Context, name string, input json.RawMessage) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]any{"name": name, "input": json.RawMessage(input)})
	return c.Ask(ctx, "tools.call", payload)
}

func (c *brainletClient) CallAgent(ctx context.Context, name string, prompt string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"name": name, "prompt": prompt})
	result, err := c.Ask(ctx, "agent.generate", payload)
	if err != nil {
		return "", err
	}
	var resp struct {
		Text string `json:"text"`
	}
	json.Unmarshal(result, &resp)
	return resp.Text, nil
}

func (c *brainletClient) CompileWASM(ctx context.Context, source string, opts WASMCompileOpts) (*WASMModule, error) {
	payload, _ := json.Marshal(map[string]any{"source": source, "options": map[string]string{"name": opts.Name}})
	result, err := c.Ask(ctx, "wasm.compile", payload)
	if err != nil {
		return nil, err
	}
	var mod WASMModule
	json.Unmarshal(result, &mod)
	return &mod, nil
}

func (c *brainletClient) DeployWASM(ctx context.Context, name string) (*ShardDescriptor, error) {
	payload, _ := json.Marshal(map[string]string{"name": name})
	result, err := c.Ask(ctx, "wasm.deploy", payload)
	if err != nil {
		return nil, err
	}
	var desc ShardDescriptor
	json.Unmarshal(result, &desc)
	return &desc, nil
}

func (c *brainletClient) GetState(ctx context.Context, key string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"key": key})
	result, err := c.Ask(ctx, "plugin.state.get", payload)
	if err != nil {
		return "", err
	}
	var resp struct {
		Value string `json:"value"`
	}
	json.Unmarshal(result, &resp)
	return resp.Value, nil
}

func (c *brainletClient) SetState(ctx context.Context, key, value string) error {
	payload, _ := json.Marshal(map[string]string{"key": key, "value": value})
	_, err := c.Ask(ctx, "plugin.state.set", payload)
	return err
}
