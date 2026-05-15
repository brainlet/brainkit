package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	browsercap "github.com/brainlet/brainkit/modulecap/browser"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/google/uuid"
)

const defaultLaunchTimeout = 15 * time.Second

// Config controls the browser lifecycle owner.
type Config struct {
	ExecutablePath string
	ProfileRoot    string
	Headless       bool
	LaunchTimeout  time.Duration
	ExtraArgs      []string
}

// Manager owns local browser processes and CDP sessions.
type Manager struct {
	cfg Config

	mu       sync.RWMutex
	sessions map[string]*session
	closed   atomic.Bool
	closing  atomic.Bool
}

type session struct {
	info         browsercap.SessionInfo
	cmd          *exec.Cmd
	done         chan error
	ownedProfile bool
}

// NewManager creates a browser lifecycle manager.
func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:      cfg,
		sessions: map[string]*session{},
	}
}

// Launch starts a local CDP-capable browser and tracks it until Close.
func (m *Manager) Launch(ctx context.Context, req browsercap.LaunchRequest) (browsercap.SessionInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m.closed.Load() {
		return browsercap.SessionInfo{}, fmt.Errorf("browser: manager is closed")
	}
	scope := req.Scope
	if scope == "" {
		scope = browsercap.ScopeThread
	}
	if scope != browsercap.ScopeThread && scope != browsercap.ScopeShared {
		return browsercap.SessionInfo{}, &sdkerrors.ValidationError{Field: "scope", Message: "must be thread or shared"}
	}
	executable, err := m.resolveExecutable(req.ExecutablePath)
	if err != nil {
		return browsercap.SessionInfo{}, err
	}
	profileDir, ownedProfile, err := m.resolveProfileDir(req.ProfileDir)
	if err != nil {
		return browsercap.SessionInfo{}, err
	}
	port, err := reserveLocalPort()
	if err != nil {
		if ownedProfile {
			_ = os.RemoveAll(profileDir)
		}
		return browsercap.SessionInfo{}, err
	}
	headless := m.cfg.Headless
	if req.Headless != nil {
		headless = *req.Headless
	}

	args := browserArgs(port, profileDir, headless, append(append([]string{}, m.cfg.ExtraArgs...), req.Args...))
	cmd := exec.Command(executable, args...)
	configureBrowserCommand(cmd)
	if err := cmd.Start(); err != nil {
		if ownedProfile {
			_ = os.RemoveAll(profileDir)
		}
		return browsercap.SessionInfo{}, fmt.Errorf("browser: launch %s: %w", executable, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	launchCtx := ctx
	if _, ok := launchCtx.Deadline(); !ok {
		timeout := m.cfg.LaunchTimeout
		if timeout <= 0 {
			timeout = defaultLaunchTimeout
		}
		var cancel context.CancelFunc
		launchCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cdpURL, wsURL, err := waitForCDP(launchCtx, port)
	if err != nil {
		_ = closeProcess(context.Background(), cmd, done)
		if ownedProfile {
			_ = os.RemoveAll(profileDir)
		}
		return browsercap.SessionInfo{}, err
	}

	info := browsercap.SessionInfo{
		ID:                   uuid.NewString(),
		Provider:             strings.TrimSpace(req.Provider),
		ThreadID:             strings.TrimSpace(req.ThreadID),
		Scope:                scope,
		CDPURL:               cdpURL,
		WebSocketDebuggerURL: wsURL,
		ProfileDir:           profileDir,
		ExecutablePath:       executable,
		PID:                  cmd.Process.Pid,
		CreatedAt:            time.Now().UTC(),
	}
	s := &session{info: info, cmd: cmd, done: done, ownedProfile: ownedProfile}

	m.mu.Lock()
	if m.closed.Load() {
		m.mu.Unlock()
		_ = s.close(context.Background())
		return browsercap.SessionInfo{}, fmt.Errorf("browser: manager is closed")
	}
	m.sessions[info.ID] = s
	m.mu.Unlock()
	return info, nil
}

// Close releases a tracked browser session.
func (m *Manager) Close(ctx context.Context, id string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return &sdkerrors.ValidationError{Field: "id", Message: "is required"}
	}
	m.mu.Lock()
	s := m.sessions[id]
	if s != nil {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if s == nil {
		return nil
	}
	return s.close(ctx)
}

// List returns a stable snapshot of tracked browser sessions.
func (m *Manager) List() []browsercap.SessionInfo {
	m.mu.RLock()
	out := make([]browsercap.SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.info)
	}
	m.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// DebugSnapshot returns lifecycle counters.
func (m *Manager) DebugSnapshot() browsercap.DebugSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := browsercap.DebugSnapshot{
		Closed:   m.closed.Load(),
		Closing:  m.closing.Load(),
		Sessions: len(m.sessions),
	}
	for _, s := range m.sessions {
		if s.cmd != nil {
			snap.OwnedProcesses++
		}
		if s.ownedProfile {
			snap.OwnedProfiles++
		}
		if s.info.External {
			snap.ExternalSessions++
		}
	}
	return snap
}

// CloseContext closes every browser resource owned by the manager.
func (m *Manager) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.closing.Store(true)
	defer m.closing.Store(false)

	m.mu.Lock()
	sessions := make([]*session, 0, len(m.sessions))
	for id, s := range m.sessions {
		sessions = append(sessions, s)
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	var err error
	var failed []*session
	for _, s := range sessions {
		if closeErr := s.close(ctx); closeErr != nil {
			err = errors.Join(err, closeErr)
			failed = append(failed, s)
		}
	}
	if err != nil {
		m.mu.Lock()
		for _, s := range failed {
			m.sessions[s.info.ID] = s
		}
		m.mu.Unlock()
		return err
	}
	m.closed.Store(true)
	return nil
}

func (s *session) close(ctx context.Context) error {
	var err error
	if s.cmd != nil {
		err = errors.Join(err, closeProcess(ctx, s.cmd, s.done))
		s.cmd = nil
	}
	if s.ownedProfile && s.info.ProfileDir != "" {
		err = errors.Join(err, os.RemoveAll(s.info.ProfileDir))
		s.ownedProfile = false
	}
	return err
}

func (m *Manager) resolveExecutable(explicit string) (string, error) {
	candidates := []string{
		strings.TrimSpace(explicit),
		strings.TrimSpace(m.cfg.ExecutablePath),
		strings.TrimSpace(os.Getenv("BRAINKIT_BROWSER_EXECUTABLE")),
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, path)
		}
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", &sdkerrors.NotConfiguredError{Feature: "browser executable"}
}

func (m *Manager) resolveProfileDir(explicit string) (string, bool, error) {
	if strings.TrimSpace(explicit) != "" {
		if err := os.MkdirAll(explicit, 0o700); err != nil {
			return "", false, err
		}
		return explicit, false, nil
	}
	root := strings.TrimSpace(m.cfg.ProfileRoot)
	if root == "" {
		dir, err := os.MkdirTemp("", "brainkit-browser-profile-*")
		return dir, true, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", false, err
	}
	dir, err := os.MkdirTemp(root, "profile-*")
	return dir, true, err
}

func reserveLocalPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func browserArgs(port int, profileDir string, headless bool, extra []string) []string {
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--remote-debugging-address=127.0.0.1",
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-component-update",
		"--disable-extensions",
		"--disable-sync",
		"--disable-popup-blocking",
		"--password-store=basic",
		"--use-mock-keychain",
	}
	if headless {
		args = append(args, "--headless=new", "--disable-gpu")
	}
	args = append(args, extra...)
	return append(args, "about:blank")
}

func waitForCDP(ctx context.Context, port int) (string, string, error) {
	cdpURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	versionURL := cdpURL + "/json/version"
	client := &http.Client{Timeout: 500 * time.Millisecond}
	var lastErr error
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
		if err != nil {
			return "", "", err
		}
		resp, err := client.Do(req)
		if err == nil {
			var body struct {
				WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&body)
			_ = resp.Body.Close()
			if decodeErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 && body.WebSocketDebuggerURL != "" {
				return cdpURL, body.WebSocketDebuggerURL, nil
			}
			if decodeErr != nil {
				lastErr = decodeErr
			} else {
				lastErr = fmt.Errorf("browser: CDP status %d missing webSocketDebuggerUrl", resp.StatusCode)
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return "", "", fmt.Errorf("browser: wait for CDP: %w: %v", ctx.Err(), lastErr)
			}
			return "", "", fmt.Errorf("browser: wait for CDP: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func closeProcess(ctx context.Context, cmd *exec.Cmd, done <-chan error) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	default:
	}
	_ = terminateBrowserProcess(cmd.Process)
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		_ = killBrowserProcess(cmd.Process)
		select {
		case <-done:
		default:
		}
		return ctx.Err()
	case <-time.After(2 * time.Second):
		_ = killBrowserProcess(cmd.Process)
		select {
		case <-done:
			return nil
		case <-time.After(2 * time.Second):
			return fmt.Errorf("browser: process %d did not exit", cmd.Process.Pid)
		}
	}
}
