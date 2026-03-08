// Ported from: packages/core/src/auth/defaults/session/cookie.ts
package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/auth/authinterfaces"

	"github.com/google/uuid"
)

// escapeRegExp escapes special regex characters in a string.
func escapeRegExp(str string) string {
	re := regexp.MustCompile(`[.*+?^${}()|[\]\\]`)
	return re.ReplaceAllStringFunc(str, func(s string) string {
		return `\` + s
	})
}

// ---------------------------------------------------------------------------
// CookieSessionProviderOptions
// ---------------------------------------------------------------------------

// CookieSessionProviderOptions holds options for CookieSessionProvider.
type CookieSessionProviderOptions struct {
	// Secret is the secret for signing cookies (required, min 32 characters).
	Secret string
	// TTL is the session TTL. Defaults to 7 days.
	TTL time.Duration
	// CookieName is the cookie name. Defaults to "mastra_session".
	CookieName string
	// CookiePath is the cookie path. Defaults to "/".
	CookiePath string
	// CookieDomain is the optional cookie domain.
	CookieDomain string
	// Secure indicates whether to use secure cookies. Defaults to false.
	Secure *bool
}

// ---------------------------------------------------------------------------
// cookieSessionData
// ---------------------------------------------------------------------------

// cookieSessionData is the session data stored in the cookie.
type cookieSessionData struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	ExpiresAt int64          `json:"expiresAt"` // Unix millis
	CreatedAt int64          `json:"createdAt"` // Unix millis
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// CookieSessionProvider
// ---------------------------------------------------------------------------

// CookieSessionProvider is a signed cookie session provider.
//
// Stores session data in signed cookies. No server-side storage required.
// The session is validated by verifying the HMAC-SHA256 signature on each request.
type CookieSessionProvider struct {
	secret       string
	ttl          time.Duration
	cookieName   string
	cookiePath   string
	cookieDomain string
	secure       bool
}

// NewCookieSessionProvider creates a new CookieSessionProvider.
// Returns an error if the secret is shorter than 32 characters.
func NewCookieSessionProvider(opts CookieSessionProviderOptions) (*CookieSessionProvider, error) {
	if len(opts.Secret) < 32 {
		return nil, errors.New("CookieSessionProvider requires a secret of at least 32 characters")
	}

	ttl := opts.TTL
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour // 7 days
	}

	cookieName := opts.CookieName
	if cookieName == "" {
		cookieName = "mastra_session"
	}

	cookiePath := opts.CookiePath
	if cookiePath == "" {
		cookiePath = "/"
	}

	secure := false
	if opts.Secure != nil {
		secure = *opts.Secure
	}

	return &CookieSessionProvider{
		secret:       opts.Secret,
		ttl:          ttl,
		cookieName:   cookieName,
		cookiePath:   cookiePath,
		cookieDomain: opts.CookieDomain,
		secure:       secure,
	}, nil
}

// CreateSession creates a new session for a user.
func (p *CookieSessionProvider) CreateSession(userID string, metadata map[string]any) (*authinterfaces.Session, error) {
	now := time.Now()
	session := &authinterfaces.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: now.Add(p.ttl),
		CreatedAt: now,
		Metadata:  metadata,
	}
	return session, nil
}

// ValidateSession validates a session by ID.
// For cookie sessions, validation happens in GetSessionFromCookie.
// This method returns nil for interface compliance.
func (p *CookieSessionProvider) ValidateSession(_ string) (*authinterfaces.Session, error) {
	// For cookie sessions, validation happens in GetSessionFromCookie.
	// This method is here for interface compliance.
	return nil, nil
}

// DestroySession destroys a session.
// Cookie sessions are destroyed by clearing the cookie client-side.
// This is a no-op on the server side.
func (p *CookieSessionProvider) DestroySession(_ string) error {
	return nil
}

// RefreshSession refreshes a session.
// For cookie sessions, we need the full session to refresh.
// This would be called with the session from GetSessionFromCookie.
func (p *CookieSessionProvider) RefreshSession(_ string) (*authinterfaces.Session, error) {
	return nil, nil
}

// GetSessionIDFromRequest extracts a session ID from an incoming request.
// Returns "" if not present or invalid.
func (p *CookieSessionProvider) GetSessionIDFromRequest(r *http.Request) string {
	session := p.GetSessionFromCookie(r)
	if session == nil {
		return ""
	}
	return session.ID
}

// GetSessionFromCookie extracts and validates the full session from the cookie.
// Returns nil if the cookie is missing, invalid, or expired.
func (p *CookieSessionProvider) GetSessionFromCookie(r *http.Request) *authinterfaces.Session {
	cookieHeader := r.Header.Get("Cookie")
	if cookieHeader == "" {
		return nil
	}

	escapedName := escapeRegExp(p.cookieName)
	re := regexp.MustCompile(fmt.Sprintf(`%s=([^;]+)`, escapedName))
	match := re.FindStringSubmatch(cookieHeader)
	if len(match) < 2 || match[1] == "" {
		return nil
	}

	decoded := p.decodeAndVerify(match[1])
	if decoded == nil {
		return nil
	}

	// Check expiration
	if decoded.ExpiresAt < time.Now().UnixMilli() {
		return nil
	}

	return &authinterfaces.Session{
		ID:        decoded.ID,
		UserID:    decoded.UserID,
		ExpiresAt: time.UnixMilli(decoded.ExpiresAt),
		CreatedAt: time.UnixMilli(decoded.CreatedAt),
		Metadata:  decoded.Metadata,
	}
}

// GetSessionHeaders creates response headers to set the session cookie.
func (p *CookieSessionProvider) GetSessionHeaders(session *authinterfaces.Session) map[string]string {
	data := &cookieSessionData{
		ID:        session.ID,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt.UnixMilli(),
		CreatedAt: session.CreatedAt.UnixMilli(),
		Metadata:  session.Metadata,
	}

	encoded := p.signAndEncode(data)
	maxAge := int(time.Until(session.ExpiresAt).Seconds())

	cookie := fmt.Sprintf("%s=%s; HttpOnly; SameSite=Lax; Path=%s; Max-Age=%d",
		p.cookieName, encoded, p.cookiePath, maxAge)

	if p.cookieDomain != "" {
		cookie += "; Domain=" + p.cookieDomain
	}

	if p.secure {
		cookie += "; Secure"
	}

	return map[string]string{"Set-Cookie": cookie}
}

// GetClearSessionHeaders creates response headers to clear the session cookie.
func (p *CookieSessionProvider) GetClearSessionHeaders() map[string]string {
	cookie := fmt.Sprintf("%s=; HttpOnly; SameSite=Lax; Path=%s; Max-Age=0",
		p.cookieName, p.cookiePath)

	if p.cookieDomain != "" {
		cookie += "; Domain=" + p.cookieDomain
	}

	return map[string]string{"Set-Cookie": cookie}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// signAndEncode signs and encodes session data into a cookie value.
func (p *CookieSessionProvider) signAndEncode(data *cookieSessionData) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	jsonStr := string(jsonBytes)
	signature := p.sign(jsonStr)
	payload := base64Encode(jsonBytes) + "." + signature
	return url.QueryEscape(payload)
}

// decodeAndVerify decodes and verifies a session cookie value.
// Returns nil if invalid.
func (p *CookieSessionProvider) decodeAndVerify(cookie string) *cookieSessionData {
	decoded, err := url.QueryUnescape(cookie)
	if err != nil {
		return nil
	}

	parts := strings.SplitN(decoded, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}

	data := parts[0]
	signature := parts[1]

	jsonBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		// Try RawStdEncoding (no padding)
		jsonBytes, err = base64.RawStdEncoding.DecodeString(data)
		if err != nil {
			return nil
		}
	}

	jsonStr := string(jsonBytes)
	expectedSignature := p.sign(jsonStr)

	// Constant-time comparison
	if !secureCompare(signature, expectedSignature) {
		return nil
	}

	var result cookieSessionData
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil
	}

	return &result
}

// sign creates an HMAC-SHA256 signature using base64url encoding (no padding).
func (p *CookieSessionProvider) sign(data string) string {
	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// base64Encode encodes bytes to standard base64.
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// secureCompare performs a constant-time string comparison.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// Compile-time check that CookieSessionProvider implements ISessionProvider.
var _ authinterfaces.ISessionProvider = (*CookieSessionProvider)(nil)
