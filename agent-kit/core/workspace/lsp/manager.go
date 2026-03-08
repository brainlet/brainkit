// Ported from: packages/core/src/workspace/lsp/manager.ts
package lsp

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// =============================================================================
// LSP Manager
// =============================================================================

// LSPManager is a per-workspace manager that owns LSP server clients.
// NOT a singleton -- each Workspace instance creates its own LSPManager.
//
// Resolves the project root per-file by walking up from the file's directory
// using language-specific markers defined on each server (e.g. tsconfig.json
// for TypeScript, go.mod for Go). Falls back to the default root when
// walkup finds nothing.
type LSPManager struct {
	mu             sync.Mutex
	clients        map[string]*LSPClient
	initPromises   map[string]chan struct{}
	initErrors     map[string]error
	fileLocks      map[string]chan struct{}
	processManager ProcessSpawner
	root           string
	config         *LSPConfig
	serverDefs     map[string]*LSPServerDef
	filesystem     ExistsFunc // optional filesystem for remote walkup
}

// NewLSPManager creates a new LSPManager.
func NewLSPManager(processManager ProcessSpawner, root string, config *LSPConfig, filesystem ExistsFunc) *LSPManager {
	if config == nil {
		config = &LSPConfig{}
	}
	return &LSPManager{
		clients:        make(map[string]*LSPClient),
		initPromises:   make(map[string]chan struct{}),
		initErrors:     make(map[string]error),
		fileLocks:      make(map[string]chan struct{}),
		processManager: processManager,
		root:           root,
		config:         config,
		serverDefs:     BuildServerDefs(config),
		filesystem:     filesystem,
	}
}

// Root returns the default project root (fallback when per-file walkup finds nothing).
func (m *LSPManager) Root() string {
	return m.root
}

// resolveRoot resolves the project root for a given file path using the server's markers.
func (m *LSPManager) resolveRoot(filePath string, markers []string) (string, error) {
	fileDir := filepath.Dir(filePath)
	if m.filesystem != nil {
		result, err := WalkUpAsync(fileDir, markers, m.filesystem)
		if err != nil {
			return m.root, nil
		}
		if result == "" {
			return m.root, nil
		}
		return result, nil
	}
	result := WalkUp(fileDir, markers)
	if result == "" {
		return m.root, nil
	}
	return result, nil
}

// acquireFileLock acquires a per-file lock so that concurrent getDiagnostics calls
// for the same file are serialized. Returns a release function.
func (m *LSPManager) acquireFileLock(filePath string) func() {
	for {
		m.mu.Lock()
		ch, locked := m.fileLocks[filePath]
		if !locked {
			// No lock held -- acquire it
			m.fileLocks[filePath] = make(chan struct{})
			m.mu.Unlock()
			return func() {
				m.mu.Lock()
				ch := m.fileLocks[filePath]
				delete(m.fileLocks, filePath)
				m.mu.Unlock()
				close(ch)
			}
		}
		m.mu.Unlock()

		// Wait for existing lock to be released
		<-ch
	}
}

// initClient initializes an LSP client for the given server definition and project root.
// Handles timeout, deduplication of concurrent init calls, and caching.
func (m *LSPManager) initClient(serverDef *LSPServerDef, projectRoot, key string) (*LSPClient, error) {
	m.mu.Lock()

	// In-progress initialization -- wait for it
	if ch, ok := m.initPromises[key]; ok {
		m.mu.Unlock()
		<-ch
		m.mu.Lock()
		client := m.clients[key]
		err := m.initErrors[key]
		delete(m.initErrors, key)
		m.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return client, nil
	}

	// Start initialization
	ch := make(chan struct{})
	m.initPromises[key] = ch
	m.mu.Unlock()

	initTimeout := 15 * time.Second
	if m.config.InitTimeout > 0 {
		initTimeout = time.Duration(m.config.InitTimeout) * time.Millisecond
	}

	client := NewLSPClient(serverDef, projectRoot, m.processManager)

	// Initialize with timeout
	done := make(chan error, 1)
	go func() {
		done <- client.Initialize(initTimeout)
	}()

	var initErr error
	select {
	case initErr = <-done:
	case <-time.After(initTimeout + 1*time.Second):
		initErr = fmt.Errorf("LSP client initialization timed out")
		go func() {
			_ = client.Shutdown()
		}()
	}

	m.mu.Lock()
	if initErr != nil {
		delete(m.clients, key)
		m.initErrors[key] = initErr

		command := serverDef.Command(projectRoot)
		hint := ""
		if override, ok := m.config.BinaryOverrides[serverDef.ID]; ok {
			hint = fmt.Sprintf(" (using binaryOverrides: %q)", override)
		} else if command != "" {
			hint = fmt.Sprintf(" (command: %q)", command)
		}
		fmt.Printf("[LSP] Failed to start %s%s: %v\n", serverDef.Name, hint, initErr)
	} else {
		m.clients[key] = client
	}
	delete(m.initPromises, key)
	m.mu.Unlock()

	close(ch)

	if initErr != nil {
		return nil, initErr
	}
	return client, nil
}

// GetClient gets or creates an LSP client for a file path.
// Resolves the project root per-file using the server's markers.
// Returns nil if no server is available.
func (m *LSPManager) GetClient(filePath string) (*LSPClient, error) {
	servers := GetServersForFile(filePath, m.config.DisableServers, m.serverDefs)
	if len(servers) == 0 {
		return nil, nil
	}

	// Prefer well-known language servers
	var serverDef *LSPServerDef
	for _, s := range servers {
		for _, langID := range s.LanguageIDs {
			if langID == "typescript" || langID == "javascript" || langID == "python" || langID == "go" {
				serverDef = s
				break
			}
		}
		if serverDef != nil {
			break
		}
	}
	if serverDef == nil {
		serverDef = servers[0]
	}

	projectRoot, err := m.resolveRoot(filePath, serverDef.Markers)
	if err != nil {
		return nil, err
	}

	// Check if the server's command is available at this root
	if serverDef.Command(projectRoot) == "" {
		return nil, nil
	}

	key := fmt.Sprintf("%s:%s", serverDef.Name, projectRoot)

	// Existing client -- check liveness before returning
	m.mu.Lock()
	if existing, ok := m.clients[key]; ok {
		if !existing.IsAlive() {
			delete(m.clients, key)
			m.mu.Unlock()
			go func() { _ = existing.Shutdown() }()
		} else {
			m.mu.Unlock()
			return existing, nil
		}
	} else {
		m.mu.Unlock()
	}

	return m.initClient(serverDef, projectRoot, key)
}

// GetDiagnostics is a convenience method: open file, send content, wait for diagnostics,
// return normalized results. Returns nil when no LSP client is available.
// Uses a per-file lock to serialize concurrent calls for the same file.
func (m *LSPManager) GetDiagnostics(filePath, content string) ([]LSPDiagnostic, error) {
	release := m.acquireFileLock(filePath)
	defer release()

	client, err := m.GetClient(filePath)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, nil
	}

	languageID := GetLanguageId(filePath)
	if languageID == "" {
		return []LSPDiagnostic{}, nil
	}

	// Open + change -> triggers diagnostics
	client.NotifyOpen(filePath, content, languageID)
	client.NotifyChange(filePath, content, 1)

	diagnosticTimeout := 5 * time.Second
	if m.config.DiagnosticTimeout > 0 {
		diagnosticTimeout = time.Duration(m.config.DiagnosticTimeout) * time.Millisecond
	}

	rawDiagnostics := client.WaitForDiagnostics(filePath, diagnosticTimeout, false, 500)
	client.NotifyClose(filePath)

	return mapRawDiagnostics(rawDiagnostics), nil
}

// GetDiagnosticsMulti gets diagnostics from ALL matching language servers for a file.
// Deduplicates results by (line, character, message).
// Individual server failures don't block other servers.
func (m *LSPManager) GetDiagnosticsMulti(filePath, content string) ([]LSPDiagnostic, error) {
	servers := GetServersForFile(filePath, m.config.DisableServers, m.serverDefs)
	if len(servers) == 0 {
		return []LSPDiagnostic{}, nil
	}

	release := m.acquireFileLock(filePath)
	defer release()

	languageID := GetLanguageId(filePath)
	if languageID == "" {
		return []LSPDiagnostic{}, nil
	}

	var allDiagnostics []LSPDiagnostic
	var wg sync.WaitGroup
	var resultMu sync.Mutex

	for _, serverDef := range servers {
		wg.Add(1)
		go func(sd *LSPServerDef) {
			defer wg.Done()

			projectRoot, err := m.resolveRoot(filePath, sd.Markers)
			if err != nil {
				return
			}
			if sd.Command(projectRoot) == "" {
				return
			}

			key := fmt.Sprintf("%s:%s", sd.Name, projectRoot)

			// Get or create client
			var client *LSPClient
			m.mu.Lock()
			if existing, ok := m.clients[key]; ok {
				if !existing.IsAlive() {
					delete(m.clients, key)
					m.mu.Unlock()
					go func() { _ = existing.Shutdown() }()
					var initErr error
					client, initErr = m.initClient(sd, projectRoot, key)
					if initErr != nil {
						return
					}
				} else {
					m.mu.Unlock()
					client = existing
				}
			} else {
				m.mu.Unlock()
				var initErr error
				client, initErr = m.initClient(sd, projectRoot, key)
				if initErr != nil {
					return
				}
			}

			if client == nil {
				return
			}

			diags := m.collectDiagnostics(client, filePath, content, languageID)
			resultMu.Lock()
			allDiagnostics = append(allDiagnostics, diags...)
			resultMu.Unlock()
		}(serverDef)
	}

	wg.Wait()

	// Deduplicate by (line, character, message)
	seen := make(map[string]bool)
	var deduped []LSPDiagnostic
	for _, d := range allDiagnostics {
		key := fmt.Sprintf("%d:%d:%s", d.Line, d.Character, d.Message)
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, d)
		}
	}

	return deduped, nil
}

// collectDiagnostics collects diagnostics from a single client for a file.
func (m *LSPManager) collectDiagnostics(client *LSPClient, filePath, content, languageID string) []LSPDiagnostic {
	client.NotifyOpen(filePath, content, languageID)
	client.NotifyChange(filePath, content, 1)

	diagnosticTimeout := 5 * time.Second
	if m.config.DiagnosticTimeout > 0 {
		diagnosticTimeout = time.Duration(m.config.DiagnosticTimeout) * time.Millisecond
	}

	rawDiagnostics := client.WaitForDiagnostics(filePath, diagnosticTimeout, false, 500)
	client.NotifyClose(filePath)

	return mapRawDiagnostics(rawDiagnostics)
}

// ShutdownAll shuts down all managed LSP clients.
func (m *LSPManager) ShutdownAll() {
	m.mu.Lock()
	clients := make([]*LSPClient, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	m.clients = make(map[string]*LSPClient)
	m.initPromises = make(map[string]chan struct{})
	m.fileLocks = make(map[string]chan struct{})
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, c := range clients {
		wg.Add(1)
		go func(client *LSPClient) {
			defer wg.Done()
			_ = client.Shutdown()
		}(c)
	}
	wg.Wait()
}

// =============================================================================
// Helpers
// =============================================================================

// mapRawDiagnostics converts raw diagnostic data to LSPDiagnostic structs.
func mapRawDiagnostics(raw []interface{}) []LSPDiagnostic {
	if raw == nil {
		return []LSPDiagnostic{}
	}

	result := make([]LSPDiagnostic, 0, len(raw))
	for _, r := range raw {
		d, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		severity := 2 // default: warning
		if s, ok := d["severity"].(float64); ok {
			severity = int(s)
		}

		message := ""
		if m, ok := d["message"].(string); ok {
			message = m
		}

		line := 1
		character := 1
		if rangeVal, ok := d["range"].(map[string]interface{}); ok {
			if start, ok := rangeVal["start"].(map[string]interface{}); ok {
				if l, ok := start["line"].(float64); ok {
					line = int(l) + 1 // LSP is 0-indexed, we report 1-indexed
				}
				if c, ok := start["character"].(float64); ok {
					character = int(c) + 1
				}
			}
		}

		source := ""
		if s, ok := d["source"].(string); ok {
			source = s
		}

		result = append(result, LSPDiagnostic{
			Severity:  MapSeverity(severity),
			Message:   message,
			Line:      line,
			Character: character,
			Source:    source,
		})
	}

	return result
}
