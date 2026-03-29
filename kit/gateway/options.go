package gateway

// RouteOption configures a route.
type RouteOption func(*routeConfig)

type routeConfig struct {
	owner        string
	params       map[string]string // URL param name → payload field name
	httpContext  bool
	statusMapper func(resp []byte, err error) int
}

// OwnedBy tags the route with a deployment source for cleanup.
func OwnedBy(source string) RouteOption {
	return func(c *routeConfig) { c.owner = source }
}

// WithParam extracts a URL path parameter and merges it into the payload.
func WithParam(paramName, fieldName string) RouteOption {
	return func(c *routeConfig) {
		if c.params == nil {
			c.params = make(map[string]string)
		}
		c.params[paramName] = fieldName
	}
}

// WithHTTPContext wraps the payload with HTTP context: {body, headers, method, path, params, query}.
func WithHTTPContext() RouteOption {
	return func(c *routeConfig) { c.httpContext = true }
}

// WithStatusMapper overrides the default HTTP status mapping for this route.
func WithStatusMapper(fn func(resp []byte, err error) int) RouteOption {
	return func(c *routeConfig) { c.statusMapper = fn }
}
