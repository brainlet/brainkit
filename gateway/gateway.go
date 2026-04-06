package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"github.com/brainlet/brainkit/internal/syncx"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// Middleware is standard Go HTTP middleware.
type Middleware func(http.Handler) http.Handler

// CORSConfig configures CORS headers.
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

// RBACChecker validates bus command permissions for .ts callers.
// When set on Config, gateway bus command handlers check caller permissions.
// Go-originated messages (no callerId metadata) bypass RBAC by design.
type RBACChecker interface {
	CheckCommand(source, command string) error
}

// StreamConfig configures SSE streaming behavior.
type StreamConfig struct {
	HeartbeatInterval time.Duration // Bridge heartbeat frequency. Default: 10s.
	HeartbeatTimeout  time.Duration // No heartbeat = producer dead. Default: 25s.
	MaxDuration       time.Duration // Total stream lifetime cap. Default: 10m.
	MaxEvents         int           // Event count + buffer cap. Default: 10000.
	GracePeriod       time.Duration // Keep session after terminal for late reconnects. Default: 60s.
}

func (c *StreamConfig) withDefaults() StreamConfig {
	out := StreamConfig{
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTimeout:  25 * time.Second,
		MaxDuration:       10 * time.Minute,
		MaxEvents:         10000,
		GracePeriod:       60 * time.Second,
	}
	if c != nil {
		if c.HeartbeatInterval > 0 {
			out.HeartbeatInterval = c.HeartbeatInterval
		}
		if c.HeartbeatTimeout > 0 {
			out.HeartbeatTimeout = c.HeartbeatTimeout
		}
		if c.MaxDuration > 0 {
			out.MaxDuration = c.MaxDuration
		}
		if c.MaxEvents > 0 {
			out.MaxEvents = c.MaxEvents
		}
		if c.GracePeriod > 0 {
			out.GracePeriod = c.GracePeriod
		}
	}
	return out
}

// Config configures the HTTP gateway.
type Config struct {
	Listen      string
	Timeout     time.Duration
	Middleware  []Middleware
	CORS        *CORSConfig
	NoHealth    bool
	Logger      *slog.Logger     // optional — nil = slog.Default()
	Tracer      Tracer           // optional — creates root spans for requests
	RBACChecker RBACChecker      // optional — checks caller permissions on bus commands
	RateLimit   *RateLimitConfig // optional — global rate limiter (429 when exceeded)
	Stream      *StreamConfig    // optional — SSE streaming config. nil = use defaults.
}

// Tracer is a minimal tracing interface to avoid importing kit/tracing.
type Tracer interface {
	StartSpan(name string, attrs map[string]string) TracerSpan
}

// TracerSpan is a span handle.
type TracerSpan interface {
	End(err error)
}

// Drainable is an optional interface for runtimes that support drain state.
type Drainable interface {
	IsDraining() bool
}

// HealthChecker is an optional interface for health checking.
// Health returns any — the gateway JSON-encodes whatever the runtime returns.
// This avoids importing kit types into the gateway package.
type HealthChecker interface {
	Alive(ctx context.Context) bool
	Ready(ctx context.Context) bool
	Health(ctx context.Context) any
}

// Gateway is the HTTP/WS/SSE protocol bridge to the bus.
type Gateway struct {
	rt           sdk.Runtime
	config       Config
	logger       *slog.Logger
	streamConfig StreamConfig
	routes       *routeTable
	srv          *http.Server
	ln           net.Listener
	active       atomic.Int64
	busUnsubs    []func()
	rbacChecker  RBACChecker

	// Stream session management
	sessionsMu  syncx.RWMutex
	sessions    map[string]*streamSession
	sweepCancel context.CancelFunc
}

// New creates an HTTP gateway.
func New(rt sdk.Runtime, cfg Config) *Gateway {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Gateway{
		rt:           rt,
		config:       cfg,
		logger:       logger,
		streamConfig: cfg.Stream.withDefaults(),
		routes:       newRouteTable(),
		rbacChecker:  cfg.RBACChecker,
		sessions:     make(map[string]*streamSession),
	}
}

// Handle registers a request/response route.
func (gw *Gateway) Handle(method, path, topic string, opts ...RouteOption) {
	cfg := routeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	gw.routes.add(&route{Method: method, Path: path, Topic: topic, Type: routeHandle, Owner: cfg.owner, Config: cfg})
}

// HandleStream registers an SSE streaming route.
func (gw *Gateway) HandleStream(method, path, topic string, opts ...RouteOption) {
	cfg := routeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	gw.routes.add(&route{Method: method, Path: path, Topic: topic, Type: routeStream, Owner: cfg.owner, Config: cfg})
}

// HandleWebSocket registers a WebSocket route.
func (gw *Gateway) HandleWebSocket(path, topic string, opts ...RouteOption) {
	cfg := routeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	gw.routes.add(&route{Method: "GET", Path: path, Topic: topic, Type: routeWebSocket, Owner: cfg.owner, Config: cfg})
}

// HandleWebhook registers a fire-and-forget route.
func (gw *Gateway) HandleWebhook(method, path, topic string, opts ...RouteOption) {
	cfg := routeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	gw.routes.add(&route{Method: method, Path: path, Topic: topic, Type: routeWebhook, Owner: cfg.owner, Config: cfg})
}

// Remove removes a route by method and path.
func (gw *Gateway) Remove(method, path string) bool {
	return gw.routes.remove(method, path)
}

// RemoveByOwner removes all routes owned by a deployment source.
func (gw *Gateway) RemoveByOwner(owner string) int {
	return gw.routes.removeByOwner(owner)
}

// ListRoutes returns all registered routes.
func (gw *Gateway) ListRoutes() []RouteInfo {
	return gw.routes.list()
}

// Start begins listening for HTTP connections and subscribes to bus route commands.
func (gw *Gateway) Start() error {
	mux := http.NewServeMux()
	if !gw.config.NoHealth {
		registerHealthRoutes(mux, gw.rt)
	}
	mux.HandleFunc("/", gw.dispatch)

	var handler http.Handler = mux
	if gw.config.CORS != nil {
		handler = corsMiddleware(gw.config.CORS, handler)
	}
	for i := len(gw.config.Middleware) - 1; i >= 0; i-- {
		handler = gw.config.Middleware[i](handler)
	}
	// Rate limiter wraps outermost — applies before all other middleware
	if gw.config.RateLimit != nil {
		handler = RateLimiter(*gw.config.RateLimit)(handler)
	}

	ln, err := net.Listen("tcp", gw.config.Listen)
	if err != nil {
		return fmt.Errorf("gateway: listen %s: %w", gw.config.Listen, err)
	}
	gw.ln = ln
	gw.srv = &http.Server{Handler: handler}

	go func() {
		if err := gw.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			gw.logger.Error("serve error", slog.String("error", err.Error()))
		}
	}()

	gw.subscribeBusCommands()

	// Start session sweep goroutine
	sweepCtx, sweepCancel := context.WithCancel(context.Background())
	gw.sweepCancel = sweepCancel
	go gw.sweepSessions(sweepCtx)

	gw.logger.Info("gateway listening", slog.String("address", gw.Addr()), slog.Int("routes", len(gw.routes.routes)))
	return nil
}

// Stop gracefully shuts down the HTTP server and unsubscribes bus commands.
func (gw *Gateway) Stop() error {
	// Stop session sweep
	if gw.sweepCancel != nil {
		gw.sweepCancel()
	}

	// Cleanup all stream sessions
	gw.sessionsMu.Lock()
	for id, session := range gw.sessions {
		session.terminate("server_shutdown")
		delete(gw.sessions, id)
	}
	gw.sessionsMu.Unlock()

	// Unsubscribe bus commands
	for _, unsub := range gw.busUnsubs {
		unsub()
	}
	gw.busUnsubs = nil

	if gw.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return gw.srv.Shutdown(ctx)
}

// Addr returns the actual listen address (including resolved port for :0).
func (gw *Gateway) Addr() string {
	if gw.ln != nil {
		return gw.ln.Addr().String()
	}
	return gw.config.Listen
}

func (gw *Gateway) dispatch(w http.ResponseWriter, r *http.Request) {
	if d, ok := gw.rt.(Drainable); ok && d.IsDraining() {
		matched, _ := gw.routes.match(r.Method, r.URL.Path)
		if matched != nil && matched.Type == routeWebhook {
			gw.handleWebhook(w, r, matched, nil)
			return
		}
		http.Error(w, "service unavailable (draining)", http.StatusServiceUnavailable)
		return
	}

	matched, pathParams := gw.routes.match(r.Method, r.URL.Path)
	if matched == nil {
		http.NotFound(w, r)
		return
	}

	gw.active.Add(1)
	defer gw.active.Add(-1)

	// Root span for the request. Reads X-Trace-ID from header if present.
	var span TracerSpan
	if gw.config.Tracer != nil {
		attrs := map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"topic":  matched.Topic,
		}
		if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
			attrs["traceId"] = traceID
		}
		span = gw.config.Tracer.StartSpan("gateway.request", attrs)
		defer span.End(nil)
	}

	switch matched.Type {
	case routeHandle:
		gw.handleRequest(w, r, matched, pathParams)
	case routeStream:
		gw.handleStream(w, r, matched, pathParams)
	case routeWebhook:
		gw.handleWebhook(w, r, matched, pathParams)
	case routeWebSocket:
		gw.handleWebSocket(w, r, matched, pathParams)
	}
}

func buildPayload(r *http.Request, matched *route, pathParams map[string]string) (json.RawMessage, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	if matched.Config.httpContext {
		headers := make(map[string]string)
		for k := range r.Header {
			headers[k] = r.Header.Get(k)
		}
		wrapped := map[string]any{
			"body":    json.RawMessage(body),
			"headers": headers,
			"method":  r.Method,
			"path":    r.URL.Path,
			"params":  pathParams,
			"query":   r.URL.Query(),
		}
		return json.Marshal(wrapped)
	}

	if len(matched.Config.params) > 0 && len(pathParams) > 0 {
		var payload map[string]any
		if len(body) > 0 {
			json.Unmarshal(body, &payload)
		}
		if payload == nil {
			payload = make(map[string]any)
		}
		for urlParam, fieldName := range matched.Config.params {
			if val, ok := pathParams[urlParam]; ok {
				payload[fieldName] = val
			}
		}
		return json.Marshal(payload)
	}

	if len(body) == 0 {
		if len(r.URL.Query()) > 0 {
			params := make(map[string]string)
			for k, v := range r.URL.Query() {
				if len(v) > 0 {
					params[k] = v[0]
				}
			}
			return json.Marshal(params)
		}
		return json.RawMessage("null"), nil
	}

	return json.RawMessage(body), nil
}

func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return uuid.NewString()
}

func mapHTTPStatus(resp []byte, err error) int {
	if err != nil {
		return http.StatusBadGateway
	}
	if len(resp) == 0 {
		return http.StatusNoContent
	}
	var parsed struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if json.Unmarshal(resp, &parsed) == nil && parsed.Error != "" {
		switch parsed.Code {
		case "VALIDATION_ERROR", "DECODE_ERROR":
			return http.StatusBadRequest
		case "PERMISSION_DENIED":
			return http.StatusForbidden
		case "NOT_FOUND":
			return http.StatusNotFound
		case "ALREADY_EXISTS":
			return http.StatusConflict
		case "RATE_LIMITED":
			return http.StatusTooManyRequests
		case "NOT_CONFIGURED":
			return http.StatusNotImplemented
		case "TIMEOUT":
			return http.StatusGatewayTimeout
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusOK
}

// sanitizeErrorPayload redacts sensitive information from error JSON payloads
// before sending to HTTP clients. Delegates to messaging.SanitizeErrorMessage.
func sanitizeErrorPayload(payload []byte) []byte {
	var parsed map[string]any
	if json.Unmarshal(payload, &parsed) != nil {
		return payload // not JSON — return as-is
	}
	if errMsg, ok := parsed["error"].(string); ok {
		parsed["error"] = messaging.SanitizeErrorMessage(errMsg)
	}
	sanitized, err := json.Marshal(parsed)
	if err != nil {
		return payload
	}
	return sanitized
}

// --- Stream session management (streamSession type defined in stream.go) ---

func (gw *Gateway) sweepSessions(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			gw.cleanExpiredSessions()
		case <-ctx.Done():
			return
		}
	}
}

func (gw *Gateway) cleanExpiredSessions() {
	now := time.Now()
	gw.sessionsMu.Lock()
	defer gw.sessionsMu.Unlock()
	for id, session := range gw.sessions {
		session.mu.RLock()
		expired := session.terminal && !session.terminalAt.IsZero() &&
			now.Sub(session.terminalAt) > gw.streamConfig.GracePeriod
		session.mu.RUnlock()
		if expired {
			if session.unsub != nil {
				session.unsub()
			}
			delete(gw.sessions, id)
		}
	}
}

func (gw *Gateway) findSession(token string) *streamSession {
	gw.sessionsMu.RLock()
	session := gw.sessions[token]
	gw.sessionsMu.RUnlock()
	if session == nil {
		return nil
	}
	// Check grace period expiry inline (don't wait for sweep)
	session.mu.RLock()
	expired := session.terminal && !session.terminalAt.IsZero() &&
		time.Since(session.terminalAt) > gw.streamConfig.GracePeriod
	session.mu.RUnlock()
	if expired {
		return nil
	}
	return session
}

func (gw *Gateway) registerSession(session *streamSession) {
	gw.sessionsMu.Lock()
	gw.sessions[session.id] = session
	gw.sessionsMu.Unlock()
}
