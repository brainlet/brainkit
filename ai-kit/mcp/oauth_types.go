// Ported from: packages/mcp/src/tool/oauth-types.ts
package mcp

// OAuthTokens represents an OAuth 2.1 token response.
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    *int   `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// OAuthProtectedResourceMetadata represents OAuth 2.0 Protected Resource Metadata.
type OAuthProtectedResourceMetadata struct {
	Resource                                string   `json:"resource"`
	AuthorizationServers                    []string `json:"authorization_servers,omitempty"`
	JWKSURI                                 string   `json:"jwks_uri,omitempty"`
	ScopesSupported                         []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported                  []string `json:"bearer_methods_supported,omitempty"`
	ResourceSigningAlgValuesSupported       []string `json:"resource_signing_alg_values_supported,omitempty"`
	ResourceName                            string   `json:"resource_name,omitempty"`
	ResourceDocumentation                   string   `json:"resource_documentation,omitempty"`
	ResourcePolicyURI                       string   `json:"resource_policy_uri,omitempty"`
	ResourceTOSURI                          string   `json:"resource_tos_uri,omitempty"`
	TLSClientCertificateBoundAccessTokens   *bool    `json:"tls_client_certificate_bound_access_tokens,omitempty"`
	AuthorizationDetailsTypesSupported      []string `json:"authorization_details_types_supported,omitempty"`
	DPoPSigningAlgValuesSupported           []string `json:"dpop_signing_alg_values_supported,omitempty"`
	DPoPBoundAccessTokensRequired           *bool    `json:"dpop_bound_access_tokens_required,omitempty"`
}

// OAuthMetadata represents OAuth 2.0 Authorization Server Metadata.
type OAuthMetadata struct {
	Issuer                                    string   `json:"issuer"`
	AuthorizationEndpoint                     string   `json:"authorization_endpoint"`
	TokenEndpoint                             string   `json:"token_endpoint"`
	RegistrationEndpoint                      string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                           []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported                    []string `json:"response_types_supported"`
	GrantTypesSupported                       []string `json:"grant_types_supported,omitempty"`
	CodeChallengeMethodsSupported             []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported         []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`
}

// OpenIdProviderMetadata represents OpenID Connect Discovery 1.0 Provider Metadata.
type OpenIdProviderMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint,omitempty"`
	JWKSURI                           string   `json:"jwks_uri"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                   []string `json:"claims_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	// Also includes OAuth fields when merged (for OIDC discovery)
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

// AuthorizationServerMetadata is a union type for OAuth or OIDC metadata.
// In Go, we use a single struct that covers both schemas.
type AuthorizationServerMetadata struct {
	Issuer                                    string   `json:"issuer"`
	AuthorizationEndpoint                     string   `json:"authorization_endpoint"`
	TokenEndpoint                             string   `json:"token_endpoint"`
	RegistrationEndpoint                      string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                           []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported                    []string `json:"response_types_supported,omitempty"`
	GrantTypesSupported                       []string `json:"grant_types_supported,omitempty"`
	CodeChallengeMethodsSupported             []string `json:"code_challenge_methods_supported,omitempty"`
	TokenEndpointAuthMethodsSupported         []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`
	// OIDC fields
	UserinfoEndpoint                  string   `json:"userinfo_endpoint,omitempty"`
	JWKSURI                           string   `json:"jwks_uri,omitempty"`
	SubjectTypesSupported             []string `json:"subject_types_supported,omitempty"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported,omitempty"`
	ClaimsSupported                   []string `json:"claims_supported,omitempty"`
}

// OAuthClientInformation represents registered OAuth client information.
type OAuthClientInformation struct {
	ClientID              string `json:"client_id"`
	ClientSecret          string `json:"client_secret,omitempty"`
	ClientIDIssuedAt      *int64 `json:"client_id_issued_at,omitempty"`
	ClientSecretExpiresAt *int64 `json:"client_secret_expires_at,omitempty"`
}

// OAuthClientMetadata represents OAuth 2.0 Dynamic Client Registration metadata.
type OAuthClientMetadata struct {
	RedirectURIs                 []string    `json:"redirect_uris"`
	TokenEndpointAuthMethod      string      `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes                   []string    `json:"grant_types,omitempty"`
	ResponseTypes                []string    `json:"response_types,omitempty"`
	ClientName                   string      `json:"client_name,omitempty"`
	ClientURI                    string      `json:"client_uri,omitempty"`
	LogoURI                      string      `json:"logo_uri,omitempty"`
	Scope                        string      `json:"scope,omitempty"`
	Contacts                     []string    `json:"contacts,omitempty"`
	TOSURI                       string      `json:"tos_uri,omitempty"`
	PolicyURI                    string      `json:"policy_uri,omitempty"`
	JWKSURI                      string      `json:"jwks_uri,omitempty"`
	JWKS                         interface{} `json:"jwks,omitempty"`
	SoftwareID                   string      `json:"software_id,omitempty"`
	SoftwareVersion              string      `json:"software_version,omitempty"`
	SoftwareStatement            string      `json:"software_statement,omitempty"`
}

// OAuthErrorResponse represents an OAuth 2.0 error response body.
type OAuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// OAuthClientInformationFull is OAuthClientMetadata merged with OAuthClientInformation.
type OAuthClientInformationFull struct {
	OAuthClientMetadata
	OAuthClientInformation
}
