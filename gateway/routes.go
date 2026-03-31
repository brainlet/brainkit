package gateway

import (
	"strings"
	"sync"
)

type routeType int

const (
	routeHandle    routeType = iota
	routeStream
	routeWebhook
	routeWebSocket
)

type route struct {
	Method string
	Path   string
	Topic  string
	Type   routeType
	Owner  string
	Config routeConfig
}

// RouteInfo is the public view of a route.
type RouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Topic  string `json:"topic"`
	Type   string `json:"type"`
	Owner  string `json:"owner,omitempty"`
}

type routeTable struct {
	mu     sync.RWMutex
	routes []*route
}

func newRouteTable() *routeTable {
	return &routeTable{}
}

func (t *routeTable) add(r *route) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, existing := range t.routes {
		if existing.Method == r.Method && existing.Path == r.Path {
			t.routes[i] = r
			return
		}
	}
	t.routes = append(t.routes, r)
}

func (t *routeTable) remove(method, path string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, r := range t.routes {
		if r.Method == method && r.Path == path {
			t.routes = append(t.routes[:i], t.routes[i+1:]...)
			return true
		}
	}
	return false
}

func (t *routeTable) removeByOwner(owner string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	count := 0
	filtered := t.routes[:0]
	for _, r := range t.routes {
		if r.Owner == owner {
			count++
		} else {
			filtered = append(filtered, r)
		}
	}
	t.routes = filtered
	return count
}

func (t *routeTable) list() []RouteInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]RouteInfo, len(t.routes))
	for i, r := range t.routes {
		result[i] = RouteInfo{
			Method: r.Method, Path: r.Path, Topic: r.Topic,
			Type: routeTypeName(r.Type), Owner: r.Owner,
		}
	}
	return result
}

func (t *routeTable) match(method, path string) (*route, map[string]string) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, r := range t.routes {
		if r.Method != method && r.Method != "*" {
			continue
		}
		if params, ok := matchPath(r.Path, path); ok {
			return r, params
		}
	}
	return nil, nil
}

func matchPath(pattern, path string) (map[string]string, bool) {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return nil, false
	}
	params := map[string]string{}
	for i, pp := range patternParts {
		if strings.HasPrefix(pp, "{") && strings.HasSuffix(pp, "}") {
			params[pp[1:len(pp)-1]] = pathParts[i]
		} else if pp != pathParts[i] {
			return nil, false
		}
	}
	return params, true
}

func routeTypeName(t routeType) string {
	switch t {
	case routeHandle:
		return "handle"
	case routeStream:
		return "stream"
	case routeWebhook:
		return "webhook"
	case routeWebSocket:
		return "websocket"
	default:
		return "unknown"
	}
}

func routeTypeFromName(name string) routeType {
	switch name {
	case "stream":
		return routeStream
	case "webhook":
		return routeWebhook
	case "websocket":
		return routeWebSocket
	default:
		return routeHandle
	}
}
