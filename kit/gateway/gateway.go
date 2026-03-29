package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

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

// Config configures the HTTP gateway.
type Config struct {
	Listen     string
	Timeout    time.Duration
	Middleware []Middleware
	CORS       *CORSConfig
	NoHealth   bool
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
	rt        sdk.Runtime
	config    Config
	routes    *routeTable
	srv       *http.Server
	ln        net.Listener
	active    atomic.Int64
	busUnsubs []func()
}

// New creates an HTTP gateway.
func New(rt sdk.Runtime, cfg Config) *Gateway {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	return &Gateway{
		rt:     rt,
		config: cfg,
		routes: newRouteTable(),
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

	ln, err := net.Listen("tcp", gw.config.Listen)
	if err != nil {
		return fmt.Errorf("gateway: listen %s: %w", gw.config.Listen, err)
	}
	gw.ln = ln
	gw.srv = &http.Server{Handler: handler}

	go func() {
		if err := gw.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[gateway] serve error: %v", err)
		}
	}()

	gw.subscribeBusCommands()
	log.Printf("[gateway] listening on %s (%d routes)", gw.Addr(), len(gw.routes.routes))
	return nil
}

// Stop gracefully shuts down the HTTP server and unsubscribes bus commands.
func (gw *Gateway) Stop() error {
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
	}
	if json.Unmarshal(resp, &parsed) == nil && parsed.Error != "" {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}
