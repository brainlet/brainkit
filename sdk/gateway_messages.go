package sdk

// ── Gateway Route Management ──

type GatewayRouteAddMsg struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Topic  string `json:"topic"`
	Type   string `json:"type,omitempty"`  // "handle", "stream", "webhook", "websocket"
	Owner  string `json:"owner,omitempty"`
}

func (GatewayRouteAddMsg) BusTopic() string { return "gateway.http.route.add" }

type GatewayRouteAddResp struct {
	Added bool `json:"added"`
}

type GatewayRouteRemoveMsg struct {
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Owner  string `json:"owner,omitempty"`
}

func (GatewayRouteRemoveMsg) BusTopic() string { return "gateway.http.route.remove" }

type GatewayRouteRemoveResp struct {
	Removed int `json:"removed"`
}

type GatewayRouteListMsg struct{}

func (GatewayRouteListMsg) BusTopic() string { return "gateway.http.route.list" }

type GatewayRouteListResp struct {
	Routes []GatewayRouteInfo `json:"routes"`
}

type GatewayRouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Topic  string `json:"topic"`
	Type   string `json:"type"`
	Owner  string `json:"owner,omitempty"`
}

type GatewayStatusMsg struct{}

func (GatewayStatusMsg) BusTopic() string { return "gateway.http.status" }

type GatewayStatusResp struct {
	Listening         bool   `json:"listening"`
	Address           string `json:"address"`
	RouteCount        int    `json:"routeCount"`
	ActiveConnections int64  `json:"activeConnections"`
}
