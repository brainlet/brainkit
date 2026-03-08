// Ported from: packages/core/src/auth/defaults/session/memory.ts
package session

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/auth/authinterfaces"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// MemorySessionProviderOptions
// ---------------------------------------------------------------------------

// MemorySessionProviderOptions holds options for MemorySessionProvider.
type MemorySessionProviderOptions struct {
	// TTL is the session TTL. Defaults to 7 days.
	TTL time.Duration
	// CookieName is the cookie name. Defaults to "mastra_session".
	CookieName string
	// CookiePath is the cookie path. Defaults to "/".
	CookiePath string
	// CleanupInterval is the cleanup interval for expired sessions.
	// Defaults to 60 seconds.
	CleanupInterval time.Duration
}

// ---------------------------------------------------------------------------
// MemorySessionProvider
// ---------------------------------------------------------------------------

// MemorySessionProvider is an in-memory session provider for development.
//
// WARNING: Sessions are lost on server restart. Not for production use.
//
// Stores sessions in a map. Useful for development but not suitable
// for production as sessions are lost on restart.
type MemorySessionProvider struct {
	mu           sync.RWMutex
	sessions     map[string]*authinterfaces.Session
	ttl          time.Duration
	cookieName   string
	cookiePath   string
	cleanupTimer *time.Ticker
	stopCleanup  chan struct{}
}

// NewMemorySessionProvider creates a new MemorySessionProvider.
func NewMemorySessionProvider(opts *MemorySessionProviderOptions) *MemorySessionProvider {
	if opts == nil {
		opts = &MemorySessionProviderOptions{}
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

	cleanupInterval := opts.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 60 * time.Second
	}

	p := &MemorySessionProvider{
		sessions:    make(map[string]*authinterfaces.Session),
		ttl:         ttl,
		cookieName:  cookieName,
		cookiePath:  cookiePath,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup timer
	p.cleanupTimer = time.NewTicker(cleanupInterval)
	go p.cleanupLoop()

	// Log warning
	log.Println("[MemorySessionProvider] Using in-memory sessions. " +
		"Sessions will be lost on server restart. " +
		"Use a persistent session provider in production.")

	return p
}

// cleanupLoop runs the periodic cleanup of expired sessions.
func (p *MemorySessionProvider) cleanupLoop() {
	for {
		select {
		case <-p.cleanupTimer.C:
			p.cleanup()
		case <-p.stopCleanup:
			return
		}
	}
}

// CreateSession creates a new session for a user.
func (p *MemorySessionProvider) CreateSession(userID string, metadata map[string]any) (*authinterfaces.Session, error) {
	now := time.Now()
	session := &authinterfaces.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: now.Add(p.ttl),
		CreatedAt: now,
		Metadata:  metadata,
	}

	p.mu.Lock()
	p.sessions[session.ID] = session
	p.mu.Unlock()

	return session, nil
}

// ValidateSession validates a session and returns it if valid.
// Returns nil if invalid or expired.
func (p *MemorySessionProvider) ValidateSession(sessionID string) (*authinterfaces.Session, error) {
	p.mu.RLock()
	session, ok := p.sessions[sessionID]
	p.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	// Check expiration
	if session.ExpiresAt.Before(time.Now()) {
		p.mu.Lock()
		delete(p.sessions, sessionID)
		p.mu.Unlock()
		return nil, nil
	}

	return session, nil
}

// DestroySession destroys a session (logout).
func (p *MemorySessionProvider) DestroySession(sessionID string) error {
	p.mu.Lock()
	delete(p.sessions, sessionID)
	p.mu.Unlock()
	return nil
}

// RefreshSession refreshes a session, extending its expiry.
// Returns nil if invalid.
func (p *MemorySessionProvider) RefreshSession(sessionID string) (*authinterfaces.Session, error) {
	session, err := p.ValidateSession(sessionID)
	if err != nil || session == nil {
		return nil, err
	}

	// Extend expiration
	session.ExpiresAt = time.Now().Add(p.ttl)

	p.mu.Lock()
	p.sessions[sessionID] = session
	p.mu.Unlock()

	return session, nil
}

// GetSessionIDFromRequest extracts a session ID from an incoming request.
// Returns "" if not present.
func (p *MemorySessionProvider) GetSessionIDFromRequest(r *http.Request) string {
	cookieHeader := r.Header.Get("Cookie")
	if cookieHeader == "" {
		return ""
	}

	escapedName := escapeRegExp(p.cookieName)
	re := regexp.MustCompile(fmt.Sprintf(`%s=([^;]+)`, escapedName))
	match := re.FindStringSubmatch(cookieHeader)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

// GetSessionHeaders creates response headers to set the session cookie.
func (p *MemorySessionProvider) GetSessionHeaders(session *authinterfaces.Session) map[string]string {
	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	cookie := fmt.Sprintf("%s=%s; HttpOnly; SameSite=Lax; Path=%s; Max-Age=%d",
		p.cookieName, session.ID, p.cookiePath, maxAge)
	return map[string]string{"Set-Cookie": cookie}
}

// GetClearSessionHeaders creates response headers to clear the session cookie.
func (p *MemorySessionProvider) GetClearSessionHeaders() map[string]string {
	cookie := fmt.Sprintf("%s=; HttpOnly; SameSite=Lax; Path=%s; Max-Age=0",
		p.cookieName, p.cookiePath)
	return map[string]string{"Set-Cookie": cookie}
}

// cleanup removes expired sessions.
func (p *MemorySessionProvider) cleanup() {
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, session := range p.sessions {
		if session.ExpiresAt.Before(now) {
			delete(p.sessions, id)
		}
	}
}

// Dispose stops the cleanup timer and releases resources.
func (p *MemorySessionProvider) Dispose() {
	if p.cleanupTimer != nil {
		p.cleanupTimer.Stop()
		close(p.stopCleanup)
		p.cleanupTimer = nil
	}
}

// GetSessionCount returns the number of active sessions (for debugging).
func (p *MemorySessionProvider) GetSessionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}

// Compile-time check that MemorySessionProvider implements ISessionProvider.
var _ authinterfaces.ISessionProvider = (*MemorySessionProvider)(nil)
