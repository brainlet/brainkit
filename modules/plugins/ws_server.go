package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/pluginws"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

// pluginWSServer hosts a WebSocket endpoint for plugin connections.
// Started by the Module when plugins are configured.
type pluginWSServer struct {
	mod             *Module
	listener        net.Listener
	server          *http.Server
	mu              sync.Mutex
	conns           map[string]*pluginWSConn // name → connection
	sockets         map[*websocket.Conn]struct{}
	closing         bool
	serveDone       chan struct{}
	serveErr        error
	connWG          sync.WaitGroup
	pingWG          sync.WaitGroup
	connWait        sync.Once
	connDone        chan struct{}
	pingWait        sync.Once
	pingDone        chan struct{}
	activeHandlers  atomic.Int64
	activePingLoops atomic.Int64
	closingSubs     []pluginSubscriptionHandle
}

type pluginWSConn struct {
	conn      *websocket.Conn
	name      string
	manifest  pluginws.Manifest
	pending   map[string]chan pluginws.ToolResult // id → result channel
	subs      []pluginSubscriptionHandle          // bus subscription handles
	tools     []string                            // registered tool names owned by this connection
	mu        sync.Mutex
	closeOnce sync.Once
	closed    bool

	// Health tracking
	healthy    bool
	lastPong   time.Time
	toolCalls  int64 // total tool calls handled
	toolErrors int64 // total tool call errors
	cancelPing context.CancelFunc
}

type pluginSubscriptionHandle interface {
	Stop()
	CloseContext(context.Context) error
}

var pluginSubscriptionCloseTimeout = 10 * time.Second

func newPluginWSServer(mod *Module) (*pluginWSServer, error) {
	s := &pluginWSServer{
		mod:       mod,
		conns:     make(map[string]*pluginWSConn),
		sockets:   make(map[*websocket.Conn]struct{}),
		serveDone: make(chan struct{}),
		connDone:  make(chan struct{}),
		pingDone:  make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /plugin/ws", s.handleConnection)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("plugin ws: listen: %w", err)
	}
	s.listener = listener

	s.server = &http.Server{Handler: mux}
	server := s.server
	go func() {
		if err := server.Serve(listener); err != nil && !isExpectedPluginWSServerClose(err) {
			s.mu.Lock()
			s.serveErr = err
			s.mu.Unlock()
		}
		close(s.serveDone)
	}()

	return s, nil
}

func (s *pluginWSServer) Addr() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	listener := s.listener
	s.mu.Unlock()
	if listener == nil {
		return ""
	}
	return listener.Addr().String()
}

func (s *pluginWSServer) URL() string {
	return "ws://" + s.Addr() + "/plugin/ws"
}

func (s *pluginWSServer) Close() {
	_ = s.CloseContext(context.Background())
}

func (s *pluginWSServer) CloseContext(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, pluginSubscriptionCloseTimeout)
		defer cancel()
	}
	s.mu.Lock()
	s.closing = true
	conns := make([]*pluginWSConn, 0, len(s.conns))
	for _, pc := range s.conns {
		conns = append(conns, pc)
	}
	sockets := make(map[*websocket.Conn]struct{}, len(s.sockets)+len(conns))
	for conn := range s.sockets {
		sockets[conn] = struct{}{}
	}
	server := s.server
	listener := s.listener
	serveDone := s.serveDone
	subscriptions := append([]pluginSubscriptionHandle(nil), s.closingSubs...)
	s.closingSubs = nil
	s.conns = map[string]*pluginWSConn{}
	s.sockets = map[*websocket.Conn]struct{}{}
	s.server = nil
	s.listener = nil
	s.mu.Unlock()
	for _, pc := range conns {
		if pc == nil {
			continue
		}
		if pc.conn != nil {
			sockets[pc.conn] = struct{}{}
		}
	}
	for conn := range sockets {
		_ = conn.CloseNow()
	}
	for _, pc := range conns {
		if pc == nil {
			continue
		}
		subscriptions = append(subscriptions, pc.cleanup(s.mod, "plugin server closing")...)
	}
	var err error
	if server != nil {
		err = errors.Join(err, pluginWSServerCloseError(server.Close()))
	}
	if listener != nil {
		err = errors.Join(err, pluginWSServerCloseError(listener.Close()))
	}
	err = errors.Join(err, waitPluginWSDone(ctx, serveDone, "plugin websocket serve"))
	err = errors.Join(err, s.waitConnHandlers(ctx))
	err = errors.Join(err, s.waitPingLoops(ctx))
	failedSubs, subErr := closePluginWSSubscriptions(ctx, subscriptions)
	err = errors.Join(err, subErr)
	s.mu.Lock()
	serveErr := s.serveErr
	if len(failedSubs) > 0 {
		s.closingSubs = append(s.closingSubs, failedSubs...)
	}
	s.mu.Unlock()
	err = errors.Join(err, serveErr)
	return err
}

func (s *pluginWSServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		slog.Error("plugin ws: accept failed", slog.String("error", err.Error()))
		return
	}
	if !s.trackAcceptedConnection(conn) {
		_ = conn.CloseNow()
		return
	}
	defer s.untrackAcceptedConnection(conn)
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

		if err := s.mod.registerPluginTool(manifest.Name, tools.RegisteredTool{
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
		}); err != nil {
			s.cleanupConnectionWithoutClosingSocket(pc, "manifest rejected")
			ackData, _ := json.Marshal(pluginws.ManifestAck{Registered: false, Error: err.Error()})
			_ = wsjson.Write(ctx, conn, pluginws.Message{
				Type: pluginws.TypeManifestAck,
				Data: ackData,
			})
			_ = conn.Close(websocket.StatusPolicyViolation, "manifest rejected")
			return
		}
		pc.tools = append(pc.tools, fullName)
	}

	s.mu.Lock()
	previous := s.conns[manifest.Name]
	s.conns[manifest.Name] = pc
	s.mu.Unlock()
	if previous != nil {
		s.cleanupConnection(previous, "plugin connection replaced")
	}

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
	s.mod.announceRegistered(ctx, pluginmsg.PluginRegisteredEvent{
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Version: manifest.Version,
		Tools:   len(manifest.Tools),
	})

	// Subscribe to topics declared in manifest
	for _, topic := range manifest.Subscriptions {
		s.subscribeTopic(ctx, pc, topic)
	}

	// Start WS ping heartbeat to detect dead plugins
	pingCtx, cancelPing := context.WithCancel(ctx)
	pc.mu.Lock()
	if pc.closed {
		pc.mu.Unlock()
		cancelPing()
		return
	}
	pc.healthy = true
	pc.lastPong = time.Now()
	pc.cancelPing = cancelPing
	s.pingWG.Add(1)
	s.activePingLoops.Add(1)
	pc.mu.Unlock()
	go func() {
		defer s.activePingLoops.Add(-1)
		defer s.pingWG.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				if err := conn.Ping(pingCtx); err != nil {
					pc.mu.Lock()
					wasHealthy := pc.healthy
					if wasHealthy {
						pc.healthy = false
					}
					pc.mu.Unlock()
					if wasHealthy {
						s.mod.kit.Audit().PluginHealthChanged(pc.name, "unhealthy")
						slog.Warn("plugin ws: heartbeat failed",
							slog.String("plugin", pc.name),
							slog.String("error", err.Error()))
					}
				} else {
					pc.mu.Lock()
					wasHealthy := pc.healthy
					pc.lastPong = time.Now()
					if !wasHealthy {
						pc.healthy = true
					}
					pc.mu.Unlock()
					if !wasHealthy {
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
			err := s.subscribeTopic(ctx, pc, sub.Topic)
			ack := pluginws.SubscribeAck{Topic: sub.Topic}
			if err != nil {
				ack.Error = err.Error()
			}
			ackData, _ := json.Marshal(ack)
			pc.mu.Lock()
			_ = wsjson.Write(ctx, conn, pluginws.Message{
				Type: pluginws.TypeSubscribeAck,
				ID:   respMsg.ID,
				Data: ackData,
			})
			pc.mu.Unlock()
		}
	}

	s.cleanupConnection(pc, "plugin disconnected")

	s.mu.Lock()
	if s.conns[manifest.Name] == pc {
		delete(s.conns, manifest.Name)
	}
	s.mu.Unlock()

	slog.Info("plugin ws: disconnected", slog.String("plugin", manifest.Name))
}

func (s *pluginWSServer) trackAcceptedConnection(conn *websocket.Conn) bool {
	if s == nil || conn == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return false
	}
	if s.sockets == nil {
		s.sockets = map[*websocket.Conn]struct{}{}
	}
	s.connWG.Add(1)
	s.activeHandlers.Add(1)
	s.sockets[conn] = struct{}{}
	return true
}

func (s *pluginWSServer) untrackAcceptedConnection(conn *websocket.Conn) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.sockets, conn)
	s.mu.Unlock()
	s.activeHandlers.Add(-1)
	s.connWG.Done()
}

func (s *pluginWSServer) waitConnHandlers(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	done := s.connDone
	if done == nil {
		done = make(chan struct{})
		s.connDone = done
	}
	s.connWait.Do(func() {
		go func() {
			s.connWG.Wait()
			close(done)
		}()
	})
	s.mu.Unlock()
	return waitPluginWSDone(ctx, done, "plugin websocket handlers")
}

func (s *pluginWSServer) waitPingLoops(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	done := s.pingDone
	if done == nil {
		done = make(chan struct{})
		s.pingDone = done
	}
	s.pingWait.Do(func() {
		go func() {
			s.pingWG.Wait()
			close(done)
		}()
	})
	s.mu.Unlock()
	return waitPluginWSDone(ctx, done, "plugin websocket pings")
}

func waitPluginWSDone(ctx context.Context, done <-chan struct{}, name string) error {
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", name, ctx.Err())
	}
}

func closePluginWSSubscriptions(ctx context.Context, subs []pluginSubscriptionHandle) ([]pluginSubscriptionHandle, error) {
	var err error
	var failed []pluginSubscriptionHandle
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if closeErr := sub.CloseContext(ctx); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("plugin websocket subscription: %w", closeErr))
			failed = append(failed, sub)
		}
	}
	return failed, err
}

func pluginWSServerCloseError(err error) error {
	if err == nil || isExpectedPluginWSServerClose(err) {
		return nil
	}
	return err
}

func isExpectedPluginWSServerClose(err error) bool {
	return errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed)
}

func (pc *pluginWSConn) cleanup(mod *Module, reason string) []pluginSubscriptionHandle {
	var subs []pluginSubscriptionHandle
	pc.closeOnce.Do(func() {
		pc.mu.Lock()
		pc.closed = true
		if pc.cancelPing != nil {
			pc.cancelPing()
			pc.cancelPing = nil
		}
		subs = append([]pluginSubscriptionHandle(nil), pc.subs...)
		pc.subs = nil
		tools := append([]string(nil), pc.tools...)
		pc.tools = nil
		for id, ch := range pc.pending {
			select {
			case ch <- pluginws.ToolResult{Error: reason}:
			default:
			}
			delete(pc.pending, id)
		}
		pluginName := pc.name
		pc.mu.Unlock()

		for _, sub := range subs {
			sub.Stop()
		}
		if mod != nil {
			mod.unregisterPluginToolNames(pluginName, tools)
		}
	})
	return subs
}

func (pc *pluginWSConn) trackSubscription(sub pluginSubscriptionHandle) bool {
	if sub == nil {
		return false
	}
	pc.mu.Lock()
	if pc.closed {
		pc.mu.Unlock()
		sub.Stop()
		return false
	}
	pc.subs = append(pc.subs, sub)
	pc.mu.Unlock()
	return true
}

func (s *pluginWSServer) cleanupConnection(pc *pluginWSConn, reason string) {
	s.cleanupConnectionWithSocketClose(pc, reason, true)
}

func (s *pluginWSServer) cleanupConnectionWithoutClosingSocket(pc *pluginWSConn, reason string) {
	s.cleanupConnectionWithSocketClose(pc, reason, false)
}

func (s *pluginWSServer) cleanupConnectionWithSocketClose(pc *pluginWSConn, reason string, closeSocket bool) {
	if pc == nil {
		return
	}
	if closeSocket && pc.conn != nil {
		_ = pc.conn.CloseNow()
	}
	s.closeSubscriptionsAfterCleanup(reason, pc.cleanup(s.mod, reason))
}

func (s *pluginWSServer) closeSubscriptionsAfterCleanup(reason string, subs []pluginSubscriptionHandle) {
	if len(subs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), pluginSubscriptionCloseTimeout)
	failed, err := closePluginWSSubscriptions(ctx, subs)
	cancel()
	if err != nil {
		slog.Warn("plugin ws: close subscriptions failed",
			slog.String("reason", reason),
			slog.String("error", err.Error()))
	}
	if len(failed) == 0 {
		return
	}
	s.mu.Lock()
	s.closingSubs = append(s.closingSubs, failed...)
	s.mu.Unlock()
}

type pluginSubscriptionLease struct {
	inner  pluginSubscriptionHandle
	cancel context.CancelFunc
}

func (h pluginSubscriptionLease) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	if h.inner != nil {
		h.inner.Stop()
	}
}

func (h pluginSubscriptionLease) CloseContext(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	if h.inner == nil {
		return nil
	}
	return h.inner.CloseContext(ctx)
}

// subscribeTopic creates a bus subscription and forwards events to the plugin over WS.
// Uses fan-out subscriber so every plugin instance receives all events (not competing
// with command handlers in the queue group).
func (s *pluginWSServer) subscribeTopic(ctx context.Context, pc *pluginWSConn, topic string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	subCtx, cancel := context.WithCancel(ctx)
	sub, err := s.mod.kit.Remote().SubscribeRawFanOutHandle(subCtx, topic, func(msg sdk.Message) {
		evtData, _ := json.Marshal(pluginws.EventMsg{
			Topic:    msg.Topic,
			Payload:  msg.Payload,
			CallerID: msg.CallerID,
			Metadata: msg.Metadata,
		})
		pc.mu.Lock()
		defer pc.mu.Unlock()
		_ = wsjson.Write(subCtx, pc.conn, pluginws.Message{
			Type: pluginws.TypeEvent,
			Data: evtData,
		})
	})
	if err != nil {
		cancel()
		slog.Warn("plugin ws: subscribe failed",
			slog.String("plugin", pc.name),
			slog.String("topic", topic),
			slog.String("error", err.Error()))
		return err
	}
	lease := pluginSubscriptionLease{inner: sub, cancel: cancel}
	if !pc.trackSubscription(lease) {
		s.closeSubscriptionsAfterCleanup("plugin connection already closed", []pluginSubscriptionHandle{lease})
		return fmt.Errorf("plugin ws: connection %q closed", pc.name)
	}
	slog.Info("plugin ws: subscribed",
		slog.String("plugin", pc.name),
		slog.String("topic", topic))
	return nil
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
