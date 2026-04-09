package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/pluginws"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// wsClient implements sdk.Runtime over WebSocket.
type wsClient struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	eventSubs map[string][]func(sdk.Message) // topic → handlers
	subMu     sync.RWMutex
}

func (c *wsClient) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	data, _ := json.Marshal(pluginws.PublishMsg{Topic: topic, Payload: payload})
	msg := pluginws.Message{Type: pluginws.TypePublish, Data: data}
	c.mu.Lock()
	defer c.mu.Unlock()
	return "", wsjson.Write(ctx, c.conn, msg)
}

// SubscribeRaw sends a subscribe request to the host via WS.
// The host creates a bus subscription and forwards events back over WS.
func (c *wsClient) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	// Register local handler
	c.subMu.Lock()
	c.eventSubs[topic] = append(c.eventSubs[topic], handler)
	c.subMu.Unlock()

	// Tell host to subscribe
	data, _ := json.Marshal(pluginws.SubscribeMsg{Topic: topic})
	c.mu.Lock()
	wsjson.Write(ctx, c.conn, pluginws.Message{Type: pluginws.TypeSubscribe, Data: data})
	c.mu.Unlock()

	return func() {
		// Note: no unsubscribe over WS yet — cleanup happens on disconnect
	}, nil
}

// dispatchEvent routes an incoming event to registered handlers.
func (c *wsClient) dispatchEvent(evt pluginws.EventMsg) {
	c.subMu.RLock()
	handlers := c.eventSubs[evt.Topic]
	c.subMu.RUnlock()

	msg := sdk.Message{
		Topic:    evt.Topic,
		Payload:  evt.Payload,
		CallerID: evt.CallerID,
	}
	for _, h := range handlers {
		h(msg)
	}
}

func (c *wsClient) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "plugin shutdown")
}

// Run connects to the host via WebSocket, sends the manifest, and handles tool calls.
//
// Flow:
//  1. Connect to BRAINKIT_PLUGIN_WS_URL
//  2. Send manifest over WS
//  3. Receive manifest ack
//  4. Print READY to stdout (host reads this)
//  5. Read tool calls from WS, execute, write results
//  6. Exit on SIGTERM/SIGINT or WS close
func (p *Plugin) Run() error {
	wsURL := os.Getenv("BRAINKIT_PLUGIN_WS_URL")
	if wsURL == "" {
		return fmt.Errorf("BRAINKIT_PLUGIN_WS_URL not set — plugin must be started by a brainkit host")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("plugin: connect to host: %w", err)
	}
	conn.SetReadLimit(10 * 1024 * 1024)

	rt := &wsClient{conn: conn, eventSubs: make(map[string][]func(sdk.Message))}

	if p.onStartFn != nil {
		if err := p.onStartFn(rt); err != nil {
			conn.Close(websocket.StatusInternalError, "OnStart failed")
			return fmt.Errorf("plugin OnStart: %w", err)
		}
	}

	// Send manifest
	manifest := pluginws.Manifest{
		Owner:       p.owner,
		Name:        p.name,
		Version:     p.version,
		Description: p.description,
	}
	for _, t := range p.tools {
		manifest.Tools = append(manifest.Tools, pluginws.ToolDef{
			Name:        t.name,
			Description: t.description,
			InputSchema: t.inputSchema,
		})
	}
	for _, s := range p.subscriptions {
		manifest.Subscriptions = append(manifest.Subscriptions, s.topic)
	}
	manifestData, _ := json.Marshal(manifest)
	if err := wsjson.Write(ctx, conn, pluginws.Message{
		Type: pluginws.TypeManifest,
		Data: manifestData,
	}); err != nil {
		return fmt.Errorf("plugin: send manifest: %w", err)
	}

	// Wait for ack
	var ackMsg pluginws.Message
	if err := wsjson.Read(ctx, conn, &ackMsg); err != nil {
		return fmt.Errorf("plugin: read manifest ack: %w", err)
	}
	if ackMsg.Type != pluginws.TypeManifestAck {
		return fmt.Errorf("plugin: expected manifest.ack, got %s", ackMsg.Type)
	}
	var ack pluginws.ManifestAck
	json.Unmarshal(ackMsg.Data, &ack)
	if ack.Error != "" {
		return fmt.Errorf("plugin: manifest rejected: %s", ack.Error)
	}

	// Register subscription handlers in the wsClient so dispatchEvent can find them.
	// On[E]() stores handlers in p.subscriptions; the manifest tells the host which
	// topics to subscribe to; but we also need the local handlers wired up.
	for _, sub := range p.subscriptions {
		handler := sub.handler
		rt.subMu.Lock()
		rt.eventSubs[sub.topic] = append(rt.eventSubs[sub.topic], func(msg sdk.Message) {
			handler(ctx, msg.Payload, rt)
		})
		rt.subMu.Unlock()
	}

	// READY — host reads this from stdout
	fmt.Fprintf(os.Stdout, "READY:%s/%s@%s\n", p.owner, p.name, p.version)

	// Tool lookup
	toolMap := make(map[string]func(context.Context, Client, json.RawMessage) (json.RawMessage, error))
	for _, t := range p.tools {
		toolMap[t.name] = t.handler
	}

	// Shutdown handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cancel()
		if p.onStopFn != nil {
			p.onStopFn()
		}
		conn.Close(websocket.StatusNormalClosure, "shutdown")
		os.Exit(0)
	}()

	// Tool call loop
	for {
		var msg pluginws.Message
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("plugin: read: %w", err)
		}

		switch msg.Type {
		case pluginws.TypeToolCall:
			var call pluginws.ToolCall
			json.Unmarshal(msg.Data, &call)

			handler, ok := toolMap[call.Tool]
			if !ok {
				writeResult(ctx, conn, msg.ID, nil, fmt.Errorf("unknown tool: %s", call.Tool))
				continue
			}

			result, toolErr := handler(ctx, rt, call.Input)
			writeResult(ctx, conn, msg.ID, result, toolErr)

		case pluginws.TypeEvent:
			var evt pluginws.EventMsg
			json.Unmarshal(msg.Data, &evt)
			rt.dispatchEvent(evt)

		case pluginws.TypeShutdown:
			return nil
		}
	}
}

func writeResult(ctx context.Context, conn *websocket.Conn, id string, result json.RawMessage, err error) {
	tr := pluginws.ToolResult{}
	if err != nil {
		tr.Error = err.Error()
	} else {
		tr.Result = result
	}
	data, _ := json.Marshal(tr)
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	wsjson.Write(ctx2, conn, pluginws.Message{
		Type: pluginws.TypeToolResult,
		ID:   id,
		Data: data,
	})
}
