package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/pluginws"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

// pluginWSServer hosts a WebSocket endpoint for plugin connections.
// Started by the Module when plugins are configured.
type pluginWSServer struct {
	mod      *Module
	listener net.Listener
	server   *http.Server
	mu       sync.Mutex
	conns    map[string]*pluginWSConn // name → connection
}

type pluginWSConn struct {
	conn     *websocket.Conn
	name     string
	manifest pluginws.Manifest
	pending  map[string]chan pluginws.ToolResult // id → result channel
	unsubs   []func()                            // bus subscription cancellers
	mu       sync.Mutex

	// Health tracking
	healthy    bool
	lastPong   time.Time
	toolCalls  int64 // total tool calls handled
	toolErrors int64 // total tool call errors
	cancelPing context.CancelFunc
}

func newPluginWSServer(mod *Module) (*pluginWSServer, error) {
	s := &pluginWSServer{
		mod:   mod,
		conns: make(map[string]*pluginWSConn),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /plugin/ws", s.handleConnection)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("plugin ws: listen: %w", err)
	}
	s.listener = listener

	s.server = &http.Server{Handler: mux}
	go s.server.Serve(listener)

	return s, nil
}

func (s *pluginWSServer) Addr() string {
	return s.listener.Addr().String()
}

func (s *pluginWSServer) URL() string {
	return "ws://" + s.Addr() + "/plugin/ws"
}

func (s *pluginWSServer) Close() {
	s.server.Close()
	s.listener.Close()
}

func (s *pluginWSServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		slog.Error("plugin ws: accept failed", slog.String("error", err.Error()))
		return
	}
	conn.SetReadLimit(10 * 1024 * 1024)

	ctx := r.Context()

	// Read manifest
	var msg pluginws.Message
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		conn.Close(websocket.StatusProtocolError, "expected manifest")
		return
	}
	if msg.Type != pluginws.TypeManifest {
		conn.Close(websocket.StatusProtocolError, "first message must be manifest")
		return
	}

	var manifest pluginws.Manifest
	if err := json.Unmarshal(msg.Data, &manifest); err != nil {
		conn.Close(websocket.StatusProtocolError, "invalid manifest")
		return
	}

	pc := &pluginWSConn{
		conn:     conn,
		name:     manifest.Name,
		manifest: manifest,
		pending:  make(map[string]chan pluginws.ToolResult),
	}

	// Register tools with WS-based executor
	for _, toolDef := range manifest.Tools {
		toolDef := toolDef
		fullName := tools.ComposeName(manifest.Owner, manifest.Name, manifest.Version, toolDef.Name)

		s.mod.kit.Tools().Register(tools.RegisteredTool{
			Name:        fullName,
			ShortName:   toolDef.Name,
			Owner:       manifest.Owner,
			Package:     manifest.Name,
			Version:     manifest.Version,
			Description: toolDef.Description,
			InputSchema: json.RawMessage(toolDef.InputSchema),
			Local:       true, // plugin tools are local-only — not callable from remote Kit instances
			Executor: &tools.GoFuncExecutor{
				Fn: func(callCtx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
					return pc.callTool(callCtx, toolDef.Name, input, callerID)
				},
			},
		})
	}

	s.mu.Lock()
	s.conns[manifest.Name] = pc
	s.mu.Unlock()

	// Send manifest ack
	ackData, _ := json.Marshal(pluginws.ManifestAck{Registered: true})
	wsjson.Write(ctx, conn, pluginws.Message{
		Type: pluginws.TypeManifestAck,
		Data: ackData,
	})

	slog.Info("plugin ws: registered",
		slog.String("plugin", manifest.Name),
		slog.Int("tools", len(manifest.Tools)))

	// Emit plugin.registered event
	_, _ = s.mod.kit.PublishRaw(ctx, sdk.PluginRegisteredEvent{}.BusTopic(), mustMarshalJSON(sdk.PluginRegisteredEvent{
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Version: manifest.Version,
		Tools:   len(manifest.Tools),
	}))
	s.mod.kit.Audit().PluginRegistered(manifest.Name, manifest.Owner, manifest.Version, len(manifest.Tools))

	// Subscribe to topics declared in manifest
	for _, topic := range manifest.Subscriptions {
		s.subscribeTopic(pc, topic)
	}

	// Start WS ping heartbeat to detect dead plugins
	pc.healthy = true
	pc.lastPong = time.Now()
	pingCtx, cancelPing := context.WithCancel(ctx)
	pc.cancelPing = cancelPing
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				if err := conn.Ping(pingCtx); err != nil {
					if pc.healthy {
						pc.healthy = false
						s.mod.kit.Audit().PluginHealthChanged(pc.name, "unhealthy")
						slog.Warn("plugin ws: heartbeat failed",
							slog.String("plugin", pc.name),
							slog.String("error", err.Error()))
					}
				} else {
					pc.mu.Lock()
					pc.lastPong = time.Now()
					pc.mu.Unlock()
					if !pc.healthy {
						pc.healthy = true
						s.mod.kit.Audit().PluginHealthChanged(pc.name, "healthy")
						slog.Info("plugin ws: heartbeat recovered", slog.String("plugin", pc.name))
					}
				}
			}
		}
	}()

	// Read messages from plugin
	for {
		var respMsg pluginws.Message
		if err := wsjson.Read(ctx, conn, &respMsg); err != nil {
			break // connection closed
		}

		switch respMsg.Type {
		case pluginws.TypeToolResult:
			var result pluginws.ToolResult
			json.Unmarshal(respMsg.Data, &result)

			pc.mu.Lock()
			ch, ok := pc.pending[respMsg.ID]
			if ok {
				delete(pc.pending, respMsg.ID)
			}
			pc.mu.Unlock()

			if ok {
				ch <- result
			}

		case pluginws.TypePublish:
			var pub pluginws.PublishMsg
			json.Unmarshal(respMsg.Data, &pub)
			if len(pub.Metadata) > 0 {
				if rt, ok := pub.Metadata["replyTo"]; ok && rt != "" {
					resolved := s.mod.kit.Remote().ResolvedTopic(rt)
					slog.Info("plugin ws: publish with replyTo",
						slog.String("plugin", pc.name),
						slog.String("topic", pub.Topic),
						slog.String("replyTo_raw", rt),
						slog.String("replyTo_resolved", resolved))
					pub.Metadata["replyTo"] = resolved
				}
				s.mod.kit.Remote().PublishRawWithMeta(ctx, pub.Topic, pub.Payload, pub.Metadata)
			} else {
				_, _ = s.mod.kit.PublishRaw(ctx, pub.Topic, pub.Payload)
			}

		case pluginws.TypeSubscribe:
			var sub pluginws.SubscribeMsg
			json.Unmarshal(respMsg.Data, &sub)
			s.subscribeTopic(pc, sub.Topic)
		}
	}

	// Cleanup on disconnect — cancel heartbeat + all bus subscriptions
	if pc.cancelPing != nil {
		pc.cancelPing()
	}
	pc.mu.Lock()
	for _, unsub := range pc.unsubs {
		unsub()
	}
	pc.mu.Unlock()

	s.mu.Lock()
	delete(s.conns, manifest.Name)
	s.mu.Unlock()

	slog.Info("plugin ws: disconnected", slog.String("plugin", manifest.Name))
}

// subscribeTopic creates a bus subscription and forwards events to the plugin over WS.
// Uses fan-out subscriber so every plugin instance receives all events (not competing
// with command handlers in the queue group).
func (s *pluginWSServer) subscribeTopic(pc *pluginWSConn, topic string) {
	unsub, err := s.mod.kit.Remote().SubscribeRawFanOut(context.Background(), topic, func(msg sdk.Message) {
		evtData, _ := json.Marshal(pluginws.EventMsg{
			Topic:    msg.Topic,
			Payload:  msg.Payload,
			CallerID: msg.CallerID,
		})
		pc.mu.Lock()
		defer pc.mu.Unlock()
		wsjson.Write(context.Background(), pc.conn, pluginws.Message{
			Type: pluginws.TypeEvent,
			Data: evtData,
		})
	})
	if err != nil {
		slog.Warn("plugin ws: subscribe failed",
			slog.String("plugin", pc.name),
			slog.String("topic", topic),
			slog.String("error", err.Error()))
		return
	}
	pc.mu.Lock()
	pc.unsubs = append(pc.unsubs, unsub)
	pc.mu.Unlock()
	slog.Info("plugin ws: subscribed",
		slog.String("plugin", pc.name),
		slog.String("topic", topic))
}

// callTool sends a tool call over WS and waits for the result.
func (pc *pluginWSConn) callTool(ctx context.Context, tool string, input json.RawMessage, callerID string) (json.RawMessage, error) {
	id := uuid.NewString()

	ch := make(chan pluginws.ToolResult, 1)
	pc.mu.Lock()
	pc.pending[id] = ch
	pc.mu.Unlock()

	callData, _ := json.Marshal(pluginws.ToolCall{
		Tool:     tool,
		Input:    input,
		CallerID: callerID,
	})
	if err := wsjson.Write(ctx, pc.conn, pluginws.Message{
		Type: pluginws.TypeToolCall,
		ID:   id,
		Data: callData,
	}); err != nil {
		pc.mu.Lock()
		delete(pc.pending, id)
		pc.mu.Unlock()
		return nil, fmt.Errorf("plugin ws: write tool call: %w", err)
	}

	select {
	case result := <-ch:
		pc.mu.Lock()
		pc.toolCalls++
		if result.Error != "" {
			pc.toolErrors++
		}
		pc.mu.Unlock()
		if result.Error != "" {
			return nil, fmt.Errorf("%s", result.Error)
		}
		return result.Result, nil
	case <-ctx.Done():
		// Host-side ctx cancelled before the plugin replied. Fire a
		// best-effort TypeCancel so the plugin can abort its handler
		// (fetch, subagent, etc.) rather than run to completion.
		pc.mu.Lock()
		delete(pc.pending, id)
		pc.mu.Unlock()
		cancelData, _ := json.Marshal(pluginws.CancelMsg{ToolCallID: id, Reason: ctx.Err().Error()})
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = wsjson.Write(writeCtx, pc.conn, pluginws.Message{
			Type: pluginws.TypeCancel,
			ID:   id,
			Data: cancelData,
		})
		writeCancel()
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		pc.mu.Lock()
		delete(pc.pending, id)
		pc.mu.Unlock()
		return nil, fmt.Errorf("plugin tool %s: timeout", tool)
	}
}
