package engine

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
// Started by the Node when plugins are configured.
type pluginWSServer struct {
	node     *Node
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
	mu       sync.Mutex
}

func newPluginWSServer(node *Node) (*pluginWSServer, error) {
	s := &pluginWSServer{
		node:  node,
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

		s.node.Kernel.Tools.Register(tools.RegisteredTool{
			Name:        fullName,
			ShortName:   toolDef.Name,
			Owner:       manifest.Owner,
			Package:     manifest.Name,
			Version:     manifest.Version,
			Description: toolDef.Description,
			InputSchema: json.RawMessage(toolDef.InputSchema),
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
	s.node.Kernel.publish(ctx, sdk.PluginRegisteredEvent{}.BusTopic(), mustMarshalJSON(sdk.PluginRegisteredEvent{
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Version: manifest.Version,
		Tools:   len(manifest.Tools),
	}))

	// Read responses from plugin
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
			s.node.Kernel.publish(ctx, pub.Topic, pub.Payload)
		}
	}

	// Cleanup on disconnect
	s.mu.Lock()
	delete(s.conns, manifest.Name)
	s.mu.Unlock()

	slog.Info("plugin ws: disconnected", slog.String("plugin", manifest.Name))
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
		if result.Error != "" {
			return nil, fmt.Errorf("%s", result.Error)
		}
		return result.Result, nil
	case <-ctx.Done():
		pc.mu.Lock()
		delete(pc.pending, id)
		pc.mu.Unlock()
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		pc.mu.Lock()
		delete(pc.pending, id)
		pc.mu.Unlock()
		return nil, fmt.Errorf("plugin tool %s: timeout", tool)
	}
}
