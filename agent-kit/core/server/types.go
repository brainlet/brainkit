// Ported from: packages/core/src/server/types.ts
package server

import (
	"net/http"
	"regexp"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ----- Stub types for unported packages -----

// Mastra represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports server (for MastraServerBase), so server cannot import core.
// core.Mastra struct satisfies this interface.
// Server uses this to access agents, tools, workflows, storage, and other
// framework services when handling HTTP requests.
type Mastra interface {
	// GetLogger returns the configured logger instance.
	GetLogger() logger.IMastraLogger
}

// MastraLogger is a type alias to logger.IMastraLogger so that core.Mastra
// satisfies the server.Mastra interface at compile time.
//
// Ported from: packages/core/src/server — uses mastra.getLogger()
type MastraLogger = logger.IMastraLogger

// RequestContext is the request-scoped key-value container.
// Wired to the real requestcontext.RequestContext type. No circular dependency:
// requestcontext does not import server.
type RequestContext = requestcontext.RequestContext

// TODO: IRBACProvider is the role-based access control provider interface.
// Port from packages/core/src/auth/ee/interfaces/rbac.ts
type IRBACProvider interface {
	GetRoles(user any) ([]string, error)
	HasRole(user any, role string) (bool, error)
	GetPermissions(user any) ([]string, error)
	HasPermission(user any, permission string) (bool, error)
	HasAllPermissions(user any, permissions []string) (bool, error)
	HasAnyPermission(user any, permissions []string) (bool, error)
}

// ----- HTTP method type -----

// Method represents an HTTP method.
type Method string

const (
	MethodGET    Method = "GET"
	MethodPOST   Method = "POST"
	MethodPUT    Method = "PUT"
	MethodDELETE Method = "DELETE"
	MethodPATCH  Method = "PATCH"
	MethodALL    Method = "ALL"
)

// ----- Handler types -----

// Handler is an HTTP handler function, analogous to Hono's Handler.
type Handler = http.HandlerFunc

// MiddlewareHandler is an HTTP middleware function.
type MiddlewareHandler func(http.Handler) http.Handler

// ----- ApiRoute -----

// ApiRoute defines a custom API route registration.
// In TypeScript this is a discriminated union: either handler or createHandler is set.
type ApiRoute struct {
	// Path is the URL path for the route.
	Path string
	// Method is the HTTP method for the route.
	Method Method
	// Handler is a direct handler function. Mutually exclusive with CreateHandler.
	Handler Handler
	// CreateHandler is a factory that lazily creates a handler given a Mastra instance.
	// Mutually exclusive with Handler.
	CreateHandler func(mastra Mastra) (Handler, error)
	// Middleware is optional middleware applied to this route.
	Middleware []MiddlewareHandler
	// RequiresAuth indicates whether this route requires authentication.
	// Defaults to true when not explicitly set.
	RequiresAuth *bool
}

// ----- Middleware -----

// Middleware represents either a global middleware or a path-scoped middleware.
type Middleware struct {
	// Path is optional. If empty, the middleware applies globally.
	Path string
	// Handler is the middleware handler.
	Handler MiddlewareHandler
}

// ----- PathRule -----

// PathRule represents a path matching rule used in auth config protected/public lists.
// In TypeScript this is: (RegExp | string | [string, Methods | Methods[]])[]
type PathRule struct {
	// Pattern is a regex pattern to match against. Mutually exclusive with Exact.
	Pattern *regexp.Regexp
	// Exact is an exact path string to match against. Mutually exclusive with Pattern.
	Exact string
	// Methods restricts the rule to specific HTTP methods. If nil, matches all methods.
	Methods []Method
}

// ----- MastraAuthConfig -----

// AuthorizeFunc is the authorization callback type.
// It receives the path, method, authenticated user, and the request.
type AuthorizeFunc func(path string, method string, user any, r *http.Request) (bool, error)

// AuthenticateTokenFunc is the token authentication callback type.
type AuthenticateTokenFunc func(token string, r *http.Request) (any, error)

// AuthRule defines a single authorization rule.
type AuthRule struct {
	// Path is a pattern, exact string, or list of strings to match.
	// Use PathRule for the resolved form.
	PathPattern *regexp.Regexp
	PathExact   string
	PathList    []string
	// Methods restricts the rule to specific HTTP methods.
	Methods []Method
	// Condition is an optional function that checks if the rule applies to the user.
	Condition func(user any) (bool, error)
	// Allow determines the authorization result when this rule matches.
	Allow bool
}

// MastraAuthConfig holds the authentication configuration for the server.
type MastraAuthConfig struct {
	// Protected paths for the server.
	Protected []PathRule
	// Public paths for the server.
	Public []PathRule
	// AuthenticateToken authenticates a token and returns the user payload.
	AuthenticateToken AuthenticateTokenFunc
	// Authorize checks if a user is authorized for a path and method.
	Authorize AuthorizeFunc
	// Rules are ordered authorization rules.
	Rules []AuthRule
}

// ----- HttpLoggingConfig -----

// HttpLogLevel represents the log level for HTTP request logging.
type HttpLogLevel string

const (
	HttpLogLevelDebug HttpLogLevel = "debug"
	HttpLogLevelInfo  HttpLogLevel = "info"
	HttpLogLevelWarn  HttpLogLevel = "warn"
)

// HttpLoggingConfig configures HTTP request logging.
type HttpLoggingConfig struct {
	// Enabled enables HTTP request logging.
	Enabled bool
	// Level is the log level for HTTP requests. Default: "info".
	Level HttpLogLevel
	// ExcludePaths are paths to exclude from logging (e.g., health checks).
	ExcludePaths []string
	// IncludeHeaders includes request headers in logs. Default: false.
	IncludeHeaders bool
	// IncludeQueryParams includes query parameters in logs. Default: false.
	IncludeQueryParams bool
	// RedactHeaders are headers to redact from logs (if IncludeHeaders is true).
	// Default: ["authorization", "cookie"].
	RedactHeaders []string
}

// ----- CORSConfig -----

// CORSConfig holds CORS configuration for the server.
type CORSConfig struct {
	// Origin specifies the allowed origins. Default: "*".
	Origin string
	// AllowMethods specifies the allowed HTTP methods.
	AllowMethods []string
	// AllowHeaders specifies the allowed request headers.
	AllowHeaders []string
	// ExposeHeaders specifies headers exposed to the browser.
	ExposeHeaders []string
	// Credentials indicates whether credentials are allowed.
	Credentials bool
}

// ----- BuildConfig -----

// BuildConfig holds build-time server configuration.
type BuildConfig struct {
	// SwaggerUI enables Swagger UI. Default: false.
	SwaggerUI bool
	// ApiReqLogs enables API request logging.
	// Use ApiReqLogsConfig for advanced configuration.
	ApiReqLogs bool
	// ApiReqLogsConfig is the advanced logging configuration.
	// Takes precedence over ApiReqLogs when non-nil.
	ApiReqLogsConfig *HttpLoggingConfig
	// OpenAPIDocs enables OpenAPI documentation. Default: false.
	OpenAPIDocs bool
}

// ----- TLSConfig -----

// TLSConfig holds TLS certificate configuration for HTTPS.
type TLSConfig struct {
	// Key is the PEM-encoded private key.
	Key []byte
	// Cert is the PEM-encoded certificate.
	Cert []byte
}

// ----- ServerConfig -----

// ErrorHandler is the custom error handler callback type.
type ErrorHandler func(err error, w http.ResponseWriter, r *http.Request)

// ServerConfig holds the full server configuration.
type ServerConfig struct {
	// Port for the server. Default: 4111.
	Port int
	// Host for the server. Default: "localhost".
	Host string
	// StudioBase is the base path for Mastra Studio UI. Default: "/".
	StudioBase string
	// ApiPrefix is the prefix for API routes. Default: "/api".
	ApiPrefix string
	// Timeout for the server in milliseconds.
	Timeout int
	// ApiRoutes are custom API routes for the server.
	ApiRoutes []ApiRoute
	// Middleware is global middleware for the server.
	Middleware []Middleware
	// CORS is the CORS configuration. Set to nil to disable CORS.
	CORS *CORSConfig
	// Build is build-time configuration for the server.
	Build *BuildConfig
	// BodySizeLimit is the body size limit in bytes. Default: 4_718_592 (4.5 MB).
	BodySizeLimit int
	// Auth is the authentication configuration.
	// Can be set to a MastraAuthConfig or a MastraAuthProvider.
	// Use AuthConfig or AuthProvider fields respectively.
	AuthConfig   *MastraAuthConfig
	AuthProvider MastraAuthProvider
	// RBAC is the role-based access control provider (Enterprise Edition).
	RBAC IRBACProvider
	// TLS holds HTTPS certificate configuration.
	TLS *TLSConfig
	// OnError is a custom error handler called when an unhandled error occurs.
	OnError ErrorHandler
}
