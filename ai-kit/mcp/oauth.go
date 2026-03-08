// Ported from: packages/mcp/src/tool/oauth.ts
package mcp

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// AuthResult represents the result of an OAuth authentication attempt.
type AuthResult string

const (
	AuthResultAuthorized AuthResult = "AUTHORIZED"
	AuthResultRedirect   AuthResult = "REDIRECT"
)

// OAuthClientProvider is the interface for managing OAuth state.
type OAuthClientProvider interface {
	// Tokens returns current access tokens if present; nil otherwise.
	Tokens() (*OAuthTokens, error)
	SaveTokens(tokens *OAuthTokens) error
	RedirectToAuthorization(authorizationURL *url.URL) error
	SaveCodeVerifier(codeVerifier string) error
	CodeVerifier() (string, error)

	// AddClientAuthentication adds custom client authentication to OAuth token requests.
	// Optional: if nil, default authentication logic is used.
	AddClientAuthentication(headers http.Header, params url.Values, tokenURL string, metadata *AuthorizationServerMetadata) error

	// InvalidateCredentials invalidates the specified credentials.
	// scope can be "all", "client", "tokens", or "verifier".
	// Optional: may return nil if not supported.
	InvalidateCredentials(scope string) error

	RedirectURL() string
	ClientMetadata() *OAuthClientMetadata
	ClientInformation() (*OAuthClientInformation, error)
	SaveClientInformation(info *OAuthClientInformation) error

	// State returns an optional state parameter for the authorization request.
	// Optional: may return "" if not needed.
	State() (string, error)

	// ValidateResourceURL validates a resource URL.
	// Optional: may return nil, nil if not supported.
	ValidateResourceURL(serverURL string, resource string) (*url.URL, error)
}

// UnauthorizedError represents an unauthorized error.
type UnauthorizedError struct {
	message string
}

func NewUnauthorizedError(message string) *UnauthorizedError {
	if message == "" {
		message = "Unauthorized"
	}
	return &UnauthorizedError{message: message}
}

func (e *UnauthorizedError) Error() string {
	return e.message
}

// ExtractResourceMetadataURL extracts the OAuth 2.0 Protected Resource Metadata
// URL from a WWW-Authenticate header (RFC9728).
func ExtractResourceMetadataURL(resp *http.Response) *url.URL {
	header := resp.Header.Get("WWW-Authenticate")
	if header == "" {
		header = resp.Header.Get("www-authenticate")
	}
	if header == "" {
		return nil
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) < 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil
	}

	// Look for resource_metadata="..." parameter
	idx := strings.Index(header, `resource_metadata="`)
	if idx == -1 {
		return nil
	}
	rest := header[idx+len(`resource_metadata="`):]
	endIdx := strings.Index(rest, `"`)
	if endIdx == -1 {
		return nil
	}
	rawURL := rest[:endIdx]

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	return u
}

// buildWellKnownPath constructs the well-known path for auth-related metadata discovery.
func buildWellKnownPath(wellKnownPrefix, pathname string, prependPathname bool) string {
	if strings.HasSuffix(pathname, "/") {
		pathname = pathname[:len(pathname)-1]
	}
	if prependPathname {
		return pathname + "/.well-known/" + wellKnownPrefix
	}
	return "/.well-known/" + wellKnownPrefix + pathname
}

// fetchWithCORSRetry tries to fetch a URL, retrying without custom headers on TypeError.
// In Go we don't have CORS issues, but we keep the retry logic for parity.
func fetchWithCORSRetry(rawURL string, headers map[string]string, client *http.Client) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Retry without headers (CORS fallback behavior)
		if headers != nil {
			return fetchWithCORSRetry(rawURL, nil, client)
		}
		return nil, nil
	}
	return resp, nil
}

// tryMetadataDiscovery tries to discover metadata at a specific URL.
func tryMetadataDiscovery(rawURL string, protocolVersion string, client *http.Client) (*http.Response, error) {
	headers := map[string]string{
		"MCP-Protocol-Version": protocolVersion,
	}
	return fetchWithCORSRetry(rawURL, headers, client)
}

// shouldAttemptFallback determines if fallback to root discovery should be attempted.
func shouldAttemptFallback(resp *http.Response, pathname string) bool {
	return resp == nil ||
		(resp.StatusCode >= 400 && resp.StatusCode < 500 && pathname != "/")
}

// discoverMetadataWithFallback is a generic function for discovering OAuth metadata with fallback support.
func discoverMetadataWithFallback(
	serverURL string,
	wellKnownType string,
	client *http.Client,
	metadataURL string,
	metadataServerURL string,
	protocolVersion string,
) (*http.Response, error) {
	issuer, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	if protocolVersion == "" {
		protocolVersion = LatestProtocolVersion
	}

	var targetURL string
	if metadataURL != "" {
		targetURL = metadataURL
	} else {
		wellKnownPath := buildWellKnownPath(wellKnownType, issuer.Path, false)
		base := metadataServerURL
		if base == "" {
			base = issuer.String()
		}
		baseURL, err := url.Parse(base)
		if err != nil {
			return nil, err
		}
		ref, _ := url.Parse(wellKnownPath)
		resolved := baseURL.ResolveReference(ref)
		resolved.RawQuery = issuer.RawQuery
		targetURL = resolved.String()
	}

	resp, err := tryMetadataDiscovery(targetURL, protocolVersion, client)
	if err != nil {
		return nil, err
	}

	if metadataURL == "" && shouldAttemptFallback(resp, issuer.Path) {
		rootRef, _ := url.Parse("/.well-known/" + wellKnownType)
		rootURL := issuer.ResolveReference(rootRef)
		resp, err = tryMetadataDiscovery(rootURL.String(), protocolVersion, client)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// DiscoverOAuthProtectedResourceMetadata discovers OAuth 2.0 Protected Resource Metadata.
func DiscoverOAuthProtectedResourceMetadata(
	serverURL string,
	resourceMetadataURL string,
	client *http.Client,
) (*OAuthProtectedResourceMetadata, error) {
	resp, err := discoverMetadataWithFallback(
		serverURL,
		"oauth-protected-resource",
		client,
		resourceMetadataURL,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("Resource server does not implement OAuth 2.0 Protected Resource Metadata.")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d trying to load well-known OAuth protected resource metadata.", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var metadata OAuthProtectedResourceMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// DiscoveryURL represents a discovery URL with its type.
type DiscoveryURL struct {
	URL  *url.URL
	Type string // "oauth" or "oidc"
}

// BuildDiscoveryURLs builds a list of discovery URLs to try for authorization server metadata.
func BuildDiscoveryURLs(authorizationServerURL string) ([]DiscoveryURL, error) {
	u, err := url.Parse(authorizationServerURL)
	if err != nil {
		return nil, err
	}
	hasPath := u.Path != "/"

	var urls []DiscoveryURL

	if !hasPath {
		oauthURL, _ := url.Parse(u.Scheme + "://" + u.Host + "/.well-known/oauth-authorization-server")
		urls = append(urls, DiscoveryURL{URL: oauthURL, Type: "oauth"})

		oidcURL, _ := url.Parse(u.Scheme + "://" + u.Host + "/.well-known/openid-configuration")
		urls = append(urls, DiscoveryURL{URL: oidcURL, Type: "oidc"})

		return urls, nil
	}

	pathname := u.Path
	if strings.HasSuffix(pathname, "/") {
		pathname = pathname[:len(pathname)-1]
	}

	u1, _ := url.Parse(u.Scheme + "://" + u.Host + "/.well-known/oauth-authorization-server" + pathname)
	urls = append(urls, DiscoveryURL{URL: u1, Type: "oauth"})

	u2, _ := url.Parse(u.Scheme + "://" + u.Host + "/.well-known/oauth-authorization-server")
	urls = append(urls, DiscoveryURL{URL: u2, Type: "oauth"})

	u3, _ := url.Parse(u.Scheme + "://" + u.Host + "/.well-known/openid-configuration" + pathname)
	urls = append(urls, DiscoveryURL{URL: u3, Type: "oidc"})

	u4, _ := url.Parse(u.Scheme + "://" + u.Host + pathname + "/.well-known/openid-configuration")
	urls = append(urls, DiscoveryURL{URL: u4, Type: "oidc"})

	return urls, nil
}

// DiscoverAuthorizationServerMetadata discovers authorization server metadata.
func DiscoverAuthorizationServerMetadata(
	authorizationServerURL string,
	client *http.Client,
	protocolVersion string,
) (*AuthorizationServerMetadata, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if protocolVersion == "" {
		protocolVersion = LatestProtocolVersion
	}

	headers := map[string]string{
		"MCP-Protocol-Version": protocolVersion,
	}

	urlsToTry, err := BuildDiscoveryURLs(authorizationServerURL)
	if err != nil {
		return nil, err
	}

	for _, du := range urlsToTry {
		resp, err := fetchWithCORSRetry(du.URL.String(), headers, client)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			continue
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			resp.Body.Close()
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			typeName := "OAuth"
			if du.Type == "oidc" {
				typeName = "OpenID provider"
			}
			return nil, fmt.Errorf("HTTP %d trying to load %s metadata from %s", resp.StatusCode, typeName, du.URL.String())
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var metadata AuthorizationServerMetadata
		if err := json.Unmarshal(body, &metadata); err != nil {
			return nil, err
		}

		if du.Type == "oidc" {
			// MCP spec requires OIDC providers to support S256 PKCE
			supported := false
			for _, m := range metadata.CodeChallengeMethodsSupported {
				if m == "S256" {
					supported = true
					break
				}
			}
			if !supported {
				return nil, fmt.Errorf("Incompatible OIDC provider at %s: does not support S256 code challenge method required by MCP specification", du.URL.String())
			}
		}

		return &metadata, nil
	}

	return nil, nil
}

// PKCEChallenge generates a PKCE code verifier and challenge.
func PKCEChallenge() (codeVerifier, codeChallenge string, err error) {
	// Generate 32 random bytes for the code verifier
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	codeVerifier = base64.RawURLEncoding.EncodeToString(b)

	// S256: SHA-256 hash of the verifier, base64url-encoded
	h := sha256.Sum256([]byte(codeVerifier))
	codeChallenge = base64.RawURLEncoding.EncodeToString(h[:])

	return codeVerifier, codeChallenge, nil
}

// StartAuthorization starts the OAuth 2.0 authorization flow.
func StartAuthorization(
	authorizationServerURL string,
	metadata *AuthorizationServerMetadata,
	clientInformation *OAuthClientInformation,
	redirectURL string,
	scope string,
	state string,
	resource *url.URL,
) (authorizationURL *url.URL, codeVerifier string, err error) {
	responseType := "code"
	codeChallengeMethod := "S256"

	if metadata != nil {
		authorizationURL, err = url.Parse(metadata.AuthorizationEndpoint)
		if err != nil {
			return nil, "", err
		}

		supported := false
		for _, rt := range metadata.ResponseTypesSupported {
			if rt == responseType {
				supported = true
				break
			}
		}
		if !supported {
			return nil, "", fmt.Errorf("Incompatible auth server: does not support response type %s", responseType)
		}

		ccmSupported := false
		for _, ccm := range metadata.CodeChallengeMethodsSupported {
			if ccm == codeChallengeMethod {
				ccmSupported = true
				break
			}
		}
		if !ccmSupported {
			return nil, "", fmt.Errorf("Incompatible auth server: does not support code challenge method %s", codeChallengeMethod)
		}
	} else {
		authorizationURL, err = url.Parse(authorizationServerURL + "/authorize")
		if err != nil {
			return nil, "", err
		}
	}

	codeVerifier, codeChallenge, err := PKCEChallenge()
	if err != nil {
		return nil, "", err
	}

	q := authorizationURL.Query()
	q.Set("response_type", responseType)
	q.Set("client_id", clientInformation.ClientID)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", codeChallengeMethod)
	q.Set("redirect_uri", redirectURL)

	if state != "" {
		q.Set("state", state)
	}
	if scope != "" {
		q.Set("scope", scope)
	}
	if strings.Contains(scope, "offline_access") {
		q.Add("prompt", "consent")
	}
	if resource != nil {
		q.Set("resource", resource.String())
	}

	authorizationURL.RawQuery = q.Encode()

	return authorizationURL, codeVerifier, nil
}

// clientAuthMethod represents an OAuth client authentication method.
type clientAuthMethod string

const (
	clientAuthBasic clientAuthMethod = "client_secret_basic"
	clientAuthPost  clientAuthMethod = "client_secret_post"
	clientAuthNone  clientAuthMethod = "none"
)

// selectClientAuthMethod determines the best client authentication method.
func selectClientAuthMethod(info *OAuthClientInformation, supportedMethods []string) clientAuthMethod {
	hasSecret := info.ClientSecret != ""

	if len(supportedMethods) == 0 {
		if hasSecret {
			return clientAuthPost
		}
		return clientAuthNone
	}

	if hasSecret && contains(supportedMethods, string(clientAuthBasic)) {
		return clientAuthBasic
	}
	if hasSecret && contains(supportedMethods, string(clientAuthPost)) {
		return clientAuthPost
	}
	if contains(supportedMethods, string(clientAuthNone)) {
		return clientAuthNone
	}
	if hasSecret {
		return clientAuthPost
	}
	return clientAuthNone
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// applyClientAuthentication applies client authentication to the request.
func applyClientAuthentication(method clientAuthMethod, info *OAuthClientInformation, headers http.Header, params url.Values) error {
	switch method {
	case clientAuthBasic:
		if info.ClientSecret == "" {
			return fmt.Errorf("client_secret_basic authentication requires a client_secret")
		}
		credentials := base64.StdEncoding.EncodeToString([]byte(info.ClientID + ":" + info.ClientSecret))
		headers.Set("Authorization", "Basic "+credentials)
		return nil
	case clientAuthPost:
		params.Set("client_id", info.ClientID)
		if info.ClientSecret != "" {
			params.Set("client_secret", info.ClientSecret)
		}
		return nil
	case clientAuthNone:
		params.Set("client_id", info.ClientID)
		return nil
	default:
		return fmt.Errorf("Unsupported client authentication method: %s", method)
	}
}

// ParseErrorResponse parses an OAuth error response from an HTTP response or string.
func ParseErrorResponse(resp *http.Response) error {
	var statusCode int
	var body string

	if resp != nil {
		statusCode = resp.StatusCode
		b, _ := io.ReadAll(resp.Body)
		body = string(b)
	}

	var errResp OAuthErrorResponse
	if err := json.Unmarshal([]byte(body), &errResp); err == nil {
		constructor, ok := OAuthErrors[errResp.Error]
		if ok {
			return constructor(MCPClientOAuthErrorOptions{
				Message: errResp.ErrorDescription,
			})
		}
		return NewServerError(MCPClientOAuthErrorOptions{
			Message: errResp.ErrorDescription,
		})
	}

	errorMessage := fmt.Sprintf("HTTP %d: Invalid OAuth error response. Raw body: %s", statusCode, body)
	return NewServerError(MCPClientOAuthErrorOptions{Message: errorMessage})
}

// ParseErrorResponseFromString parses an OAuth error response from a string.
func ParseErrorResponseFromString(body string) error {
	var errResp OAuthErrorResponse
	if err := json.Unmarshal([]byte(body), &errResp); err == nil {
		constructor, ok := OAuthErrors[errResp.Error]
		if ok {
			return constructor(MCPClientOAuthErrorOptions{
				Message: errResp.ErrorDescription,
			})
		}
		return NewServerError(MCPClientOAuthErrorOptions{
			Message: errResp.ErrorDescription,
		})
	}

	return NewServerError(MCPClientOAuthErrorOptions{
		Message: fmt.Sprintf("Invalid OAuth error response. Raw body: %s", body),
	})
}

// ExchangeAuthorization exchanges an authorization code for an access token.
func ExchangeAuthorization(
	authorizationServerURL string,
	metadata *AuthorizationServerMetadata,
	clientInformation *OAuthClientInformation,
	authorizationCode string,
	codeVerifier string,
	redirectURI string,
	resource *url.URL,
	addClientAuth func(http.Header, url.Values, string, *AuthorizationServerMetadata) error,
	client *http.Client,
) (*OAuthTokens, error) {
	if client == nil {
		client = http.DefaultClient
	}

	grantType := "authorization_code"

	var tokenURL string
	if metadata != nil && metadata.TokenEndpoint != "" {
		tokenURL = metadata.TokenEndpoint
	} else {
		tokenURL = authorizationServerURL + "/token"
	}

	if metadata != nil && metadata.GrantTypesSupported != nil {
		if !contains(metadata.GrantTypesSupported, grantType) {
			return nil, fmt.Errorf("Incompatible auth server: does not support grant type %s", grantType)
		}
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set("Accept", "application/json")

	params := url.Values{}
	params.Set("grant_type", grantType)
	params.Set("code", authorizationCode)
	params.Set("code_verifier", codeVerifier)
	params.Set("redirect_uri", redirectURI)

	if addClientAuth != nil {
		if err := addClientAuth(headers, params, authorizationServerURL, metadata); err != nil {
			return nil, err
		}
	} else {
		supportedMethods := []string{}
		if metadata != nil {
			supportedMethods = metadata.TokenEndpointAuthMethodsSupported
		}
		method := selectClientAuthMethod(clientInformation, supportedMethods)
		if err := applyClientAuthentication(method, clientInformation, headers, params); err != nil {
			return nil, err
		}
	}

	if resource != nil {
		params.Set("resource", resource.String())
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tokens OAuthTokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, err
	}
	return &tokens, nil
}

// RefreshAuthorization refreshes an access token using a refresh token.
func RefreshAuthorization(
	authorizationServerURL string,
	metadata *AuthorizationServerMetadata,
	clientInformation *OAuthClientInformation,
	refreshToken string,
	resource *url.URL,
	addClientAuth func(http.Header, url.Values, string, *AuthorizationServerMetadata) error,
	client *http.Client,
) (*OAuthTokens, error) {
	if client == nil {
		client = http.DefaultClient
	}

	grantType := "refresh_token"

	var tokenURL string
	if metadata != nil && metadata.TokenEndpoint != "" {
		tokenURL = metadata.TokenEndpoint

		if metadata.GrantTypesSupported != nil {
			if !contains(metadata.GrantTypesSupported, grantType) {
				return nil, fmt.Errorf("Incompatible auth server: does not support grant type %s", grantType)
			}
		}
	} else {
		tokenURL = authorizationServerURL + "/token"
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	headers.Set("Accept", "application/json")

	params := url.Values{}
	params.Set("grant_type", grantType)
	params.Set("refresh_token", refreshToken)

	if addClientAuth != nil {
		if err := addClientAuth(headers, params, authorizationServerURL, metadata); err != nil {
			return nil, err
		}
	} else {
		supportedMethods := []string{}
		if metadata != nil {
			supportedMethods = metadata.TokenEndpointAuthMethodsSupported
		}
		method := selectClientAuthMethod(clientInformation, supportedMethods)
		if err := applyClientAuthentication(method, clientInformation, headers, params); err != nil {
			return nil, err
		}
	}

	if resource != nil {
		params.Set("resource", resource.String())
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tokens OAuthTokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, err
	}
	// Preserve original refresh token if not replaced
	if tokens.RefreshToken == "" {
		tokens.RefreshToken = refreshToken
	}
	return &tokens, nil
}

// RegisterClient performs OAuth 2.0 Dynamic Client Registration according to RFC 7591.
func RegisterClient(
	authorizationServerURL string,
	metadata *AuthorizationServerMetadata,
	clientMetadata *OAuthClientMetadata,
	client *http.Client,
) (*OAuthClientInformationFull, error) {
	if client == nil {
		client = http.DefaultClient
	}

	var registrationURL string
	if metadata != nil {
		if metadata.RegistrationEndpoint == "" {
			return nil, fmt.Errorf("Incompatible auth server: does not support dynamic client registration")
		}
		registrationURL = metadata.RegistrationEndpoint
	} else {
		registrationURL = authorizationServerURL + "/register"
	}

	body, err := json.Marshal(clientMetadata)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, registrationURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseErrorResponse(resp)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var info OAuthClientInformationFull
	if err := json.Unmarshal(respBody, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// AuthOptions holds options for the Auth function.
type AuthOptions struct {
	ServerURL           *url.URL
	AuthorizationCode   string
	Scope               string
	ResourceMetadataURL *url.URL
	HTTPClient          *http.Client
}

// Auth performs the OAuth authentication flow.
func Auth(provider OAuthClientProvider, options AuthOptions) (AuthResult, error) {
	result, err := authInternal(provider, options)
	if err != nil {
		var invalidClientErr *InvalidClientError
		var unauthorizedClientErr *UnauthorizedClientError
		var invalidGrantErr *InvalidGrantError

		if errors.As(err, &invalidClientErr) || errors.As(err, &unauthorizedClientErr) {
			_ = provider.InvalidateCredentials("all")
			return authInternal(provider, options)
		}
		if errors.As(err, &invalidGrantErr) {
			_ = provider.InvalidateCredentials("tokens")
			return authInternal(provider, options)
		}
		return "", err
	}
	return result, nil
}

// SelectResourceURL selects the appropriate resource URL for the auth flow.
func SelectResourceURL(serverURL string, provider OAuthClientProvider, resourceMetadata *OAuthProtectedResourceMetadata) (*url.URL, error) {
	defaultResource, err := ResourceURLFromServerURL(serverURL)
	if err != nil {
		return nil, err
	}

	validated, err := provider.ValidateResourceURL(defaultResource.String(), "")
	if err == nil && validated != nil {
		return validated, nil
	}

	if resourceMetadata == nil {
		return nil, nil
	}

	validated, err = provider.ValidateResourceURL(defaultResource.String(), resourceMetadata.Resource)
	if err == nil && validated != nil {
		return validated, nil
	}

	allowed, err := CheckResourceAllowed(defaultResource.String(), resourceMetadata.Resource)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("Protected resource %s does not match expected %s (or origin)", resourceMetadata.Resource, defaultResource.String())
	}

	return url.Parse(resourceMetadata.Resource)
}

func authInternal(provider OAuthClientProvider, options AuthOptions) (AuthResult, error) {
	serverURL := options.ServerURL.String()

	var resourceMetadata *OAuthProtectedResourceMetadata
	var authorizationServerURL string

	resourceMetadataURL := ""
	if options.ResourceMetadataURL != nil {
		resourceMetadataURL = options.ResourceMetadataURL.String()
	}

	rm, err := DiscoverOAuthProtectedResourceMetadata(serverURL, resourceMetadataURL, options.HTTPClient)
	if err == nil && rm != nil {
		resourceMetadata = rm
		if len(rm.AuthorizationServers) > 0 {
			authorizationServerURL = rm.AuthorizationServers[0]
		}
	}

	if authorizationServerURL == "" {
		authorizationServerURL = serverURL
	}

	resource, err := SelectResourceURL(serverURL, provider, resourceMetadata)
	if err != nil {
		return "", err
	}

	metadata, err := DiscoverAuthorizationServerMetadata(authorizationServerURL, options.HTTPClient, "")
	if err != nil {
		return "", err
	}

	clientInformation, err := provider.ClientInformation()
	if err != nil {
		return "", err
	}

	if clientInformation == nil {
		if options.AuthorizationCode != "" {
			return "", fmt.Errorf("Existing OAuth client information is required when exchanging an authorization code")
		}

		fullInformation, err := RegisterClient(
			authorizationServerURL,
			metadata,
			provider.ClientMetadata(),
			options.HTTPClient,
		)
		if err != nil {
			return "", err
		}

		if err := provider.SaveClientInformation(&fullInformation.OAuthClientInformation); err != nil {
			return "", err
		}
		clientInformation = &fullInformation.OAuthClientInformation
	}

	// Exchange authorization code for tokens
	if options.AuthorizationCode != "" {
		cv, err := provider.CodeVerifier()
		if err != nil {
			return "", err
		}
		tokens, err := ExchangeAuthorization(
			authorizationServerURL,
			metadata,
			clientInformation,
			options.AuthorizationCode,
			cv,
			provider.RedirectURL(),
			resource,
			provider.AddClientAuthentication,
			options.HTTPClient,
		)
		if err != nil {
			return "", err
		}
		if err := provider.SaveTokens(tokens); err != nil {
			return "", err
		}
		return AuthResultAuthorized, nil
	}

	tokens, err := provider.Tokens()
	if err != nil {
		return "", err
	}

	// Handle token refresh or new authorization
	if tokens != nil && tokens.RefreshToken != "" {
		newTokens, refreshErr := RefreshAuthorization(
			authorizationServerURL,
			metadata,
			clientInformation,
			tokens.RefreshToken,
			resource,
			provider.AddClientAuthentication,
			options.HTTPClient,
		)
		if refreshErr != nil {
			// If ServerError or non-OAuth error, swallow and continue to new auth
			var serverErr *ServerError
			var oauthErr *MCPClientOAuthError
			if errors.As(refreshErr, &serverErr) || !errors.As(refreshErr, &oauthErr) {
				// Could not refresh, continue
			} else {
				return "", refreshErr
			}
		} else {
			if err := provider.SaveTokens(newTokens); err != nil {
				return "", err
			}
			return AuthResultAuthorized, nil
		}
	}

	st, _ := provider.State()

	scope := options.Scope
	if scope == "" {
		cm := provider.ClientMetadata()
		if cm != nil {
			scope = cm.Scope
		}
	}

	authURL, cv, err := StartAuthorization(
		authorizationServerURL,
		metadata,
		clientInformation,
		provider.RedirectURL(),
		scope,
		st,
		resource,
	)
	if err != nil {
		return "", err
	}

	if err := provider.SaveCodeVerifier(cv); err != nil {
		return "", err
	}
	if err := provider.RedirectToAuthorization(authURL); err != nil {
		return "", err
	}
	return AuthResultRedirect, nil
}
